package jmap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultBaseURL is the Fastmail JMAP API base URL
	DefaultBaseURL = "https://api.fastmail.com"

	// SessionPath is the path to the JMAP session endpoint
	SessionPath = "/jmap/session"

	// Default retry configuration values
	DefaultMaxRetries  = 3
	DefaultInitialDelay = 1 * time.Second
	DefaultMaxDelay     = 30 * time.Second

	// MaxUploadSize is the maximum size for blob uploads (50MB)
	MaxUploadSize = 50 * 1024 * 1024
)

// RetryConfig configures retry behavior for JMAP requests
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts (default: 3)
	MaxRetries int

	// InitialDelay is the initial delay before the first retry (default: 1s)
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries (default: 30s)
	MaxDelay time.Duration
}

// DefaultRetryConfig returns a RetryConfig with sensible defaults
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:   DefaultMaxRetries,
		InitialDelay: DefaultInitialDelay,
		MaxDelay:     DefaultMaxDelay,
	}
}

// Session represents a JMAP session with API endpoints and account information
type Session struct {
	APIUrl       string         `json:"apiUrl"`
	AccountID    string         `json:"accountId"`
	Capabilities map[string]any `json:"capabilities"`
	DownloadURL  string         `json:"downloadUrl"`
	UploadURL    string         `json:"uploadUrl"`
}

// Request represents a JMAP request
type Request struct {
	Using       []string     `json:"using"`
	MethodCalls []MethodCall `json:"methodCalls"`
}

// MethodCall represents a single JMAP method call [methodName, args, callId]
type MethodCall [3]any

// Response represents a JMAP response
type Response struct {
	MethodResponses []MethodResponse `json:"methodResponses"`
	SessionState    string           `json:"sessionState"`
}

// MethodResponse represents a single JMAP method response [methodName, result, callId]
type MethodResponse [3]any

// Client is a JMAP client for interacting with the Fastmail API
type Client struct {
	token        string
	baseURL      string
	session      *Session
	sessionFetch time.Time
	sessionTTL   time.Duration
	sessionMu    sync.RWMutex
	http         *http.Client
	retry        *RetryConfig
}

// Compile-time interface compliance checks
var _ EmailService = (*Client)(nil)
var _ MaskedEmailService = (*Client)(nil)
var _ VacationService = (*Client)(nil)

// NewClient creates a new JMAP client with the provided API token
func NewClient(token string) *Client {
	return &Client{
		token:      token,
		baseURL:    DefaultBaseURL,
		sessionTTL: 1 * time.Hour,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		retry: DefaultRetryConfig(),
	}
}

// NewClientWithBaseURL creates a new JMAP client with a custom base URL
func NewClientWithBaseURL(token, baseURL string) *Client {
	return &Client{
		token:      token,
		baseURL:    baseURL,
		sessionTTL: 1 * time.Hour,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		retry: DefaultRetryConfig(),
	}
}

// SetRetryConfig sets a custom retry configuration (nil = use defaults)
func (c *Client) SetRetryConfig(config *RetryConfig) {
	if config == nil {
		c.retry = DefaultRetryConfig()
	} else {
		c.retry = config
	}
}

// isRetriableHTTPError checks if an error should trigger a retry (network errors only)
func isRetriableHTTPError(err error) bool {
	// Network timeout errors are retriable
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	return false
}

// isRetriableStatus checks if an HTTP status code should trigger a retry
func isRetriableStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError,      // 500
		http.StatusBadGateway,                // 502
		http.StatusServiceUnavailable,        // 503
		http.StatusGatewayTimeout:            // 504
		return true
	default:
		return false
	}
}

// getRetryDelay calculates the delay before the next retry, respecting Retry-After header
func (c *Client) getRetryDelay(attempt int, resp *http.Response) time.Duration {
	// Check for Retry-After header
	if resp != nil {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			// Try parsing as seconds (integer)
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				delay := time.Duration(seconds) * time.Second
				if delay > c.retry.MaxDelay {
					return c.retry.MaxDelay
				}
				return delay
			}
			// Try parsing as HTTP-date (we'll just use exponential backoff if this fails)
		}
	}

	// Exponential backoff: initialDelay * 2^attempt
	delay := c.retry.InitialDelay * (1 << uint(attempt))

	// Add jitter (±20%) to prevent thundering herd
	jitterRange := int64(delay) / 5 // 20% of delay
	if jitterRange > 0 {
		jitter := time.Duration(rand.Int63n(jitterRange*2) - jitterRange)
		delay = delay + jitter
	}

	if delay > c.retry.MaxDelay {
		delay = c.retry.MaxDelay
	}
	return delay
}

// GetSession fetches the JMAP session from the server and caches it for reuse
func (c *Client) GetSession(ctx context.Context) (*Session, error) {
	// Read lock for checking cache
	c.sessionMu.RLock()
	if c.session != nil && time.Since(c.sessionFetch) < c.sessionTTL {
		session := c.session
		c.sessionMu.RUnlock()
		return session, nil
	}
	c.sessionMu.RUnlock()

	// Write lock for fetching new session
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()

	// Double-check after acquiring write lock (another goroutine may have fetched)
	if c.session != nil && time.Since(c.sessionFetch) < c.sessionTTL {
		return c.session, nil
	}

	// Build session URL
	sessionURL := c.baseURL + SessionPath

	var lastErr error
	var resp *http.Response

	// Retry loop
	for attempt := 0; attempt <= c.retry.MaxRetries; attempt++ {
		// Create request
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, sessionURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating session request: %w", err)
		}

		// Add authorization header
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", "application/json")

		// Execute request
		resp, err = c.http.Do(req)
		if err != nil {
			// Check if error is retriable
			if !isRetriableHTTPError(err) {
				return nil, fmt.Errorf("fetching session: %w", err)
			}

			lastErr = err
			if attempt < c.retry.MaxRetries {
				delay := c.getRetryDelay(attempt, nil)
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
				}
			}
			continue
		}

		// Check response status
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			// Check if status is retriable
			if !isRetriableStatus(resp.StatusCode) {
				return nil, fmt.Errorf("session request failed with status %d: %s", resp.StatusCode, string(body))
			}

			lastErr = fmt.Errorf("session request failed with status %d: %s", resp.StatusCode, string(body))
			if attempt < c.retry.MaxRetries {
				delay := c.getRetryDelay(attempt, resp)
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
				}
			}
			continue
		}

		// Success - parse session response
		defer resp.Body.Close()

		var sessionData struct {
			APIUrl       string                    `json:"apiUrl"`
			Accounts     map[string]map[string]any `json:"accounts"`
			Capabilities map[string]any            `json:"capabilities"`
			DownloadURL  string                    `json:"downloadUrl"`
			UploadURL    string                    `json:"uploadUrl"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&sessionData); err != nil {
			return nil, fmt.Errorf("decoding session response: %w", err)
		}

		// Extract the first account ID (Fastmail typically has one account)
		var accountID string
		for id := range sessionData.Accounts {
			accountID = id
			break
		}

		if accountID == "" {
			return nil, ErrNoAccounts
		}

		// Build and cache session
		c.session = &Session{
			APIUrl:       sessionData.APIUrl,
			AccountID:    accountID,
			Capabilities: sessionData.Capabilities,
			DownloadURL:  sessionData.DownloadURL,
			UploadURL:    sessionData.UploadURL,
		}

		// Record the time of successful session fetch
		c.sessionFetch = time.Now()

		return c.session, nil
	}

	// All retries exhausted
	return nil, fmt.Errorf("session request failed after %d retries: %w", c.retry.MaxRetries, lastErr)
}

// MakeRequest executes a JMAP request and returns the response
func (c *Client) MakeRequest(ctx context.Context, req *Request) (*Response, error) {
	// Ensure we have a session
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting session: %w", err)
	}

	// Marshal request body once (reuse for retries)
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	var lastErr error
	var httpResp *http.Response

	// Retry loop
	for attempt := 0; attempt <= c.retry.MaxRetries; attempt++ {
		// Create HTTP request with fresh body reader for each attempt
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, session.APIUrl, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		// Add headers
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
		httpReq.Header.Set("Content-Type", "application/json")

		// Execute request
		httpResp, err = c.http.Do(httpReq)
		if err != nil {
			// Check if error is retriable
			if !isRetriableHTTPError(err) {
				return nil, fmt.Errorf("executing request: %w", err)
			}

			lastErr = err
			if attempt < c.retry.MaxRetries {
				delay := c.getRetryDelay(attempt, nil)
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
				}
			}
			continue
		}

		// Check response status
		if httpResp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(httpResp.Body)
			httpResp.Body.Close()

			// Check if status is retriable
			if !isRetriableStatus(httpResp.StatusCode) {
				return nil, fmt.Errorf("JMAP request failed with status %d: %s", httpResp.StatusCode, string(bodyBytes))
			}

			lastErr = fmt.Errorf("JMAP request failed with status %d: %s", httpResp.StatusCode, string(bodyBytes))
			if attempt < c.retry.MaxRetries {
				delay := c.getRetryDelay(attempt, httpResp)
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
				}
			}
			continue
		}

		// Success - parse response
		defer httpResp.Body.Close()

		var response Response
		if err := json.NewDecoder(httpResp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("decoding response: %w", err)
		}

		return &response, nil
	}

	// All retries exhausted
	return nil, fmt.Errorf("JMAP request failed after %d retries: %w", c.retry.MaxRetries, lastErr)
}

// ClearSession clears the cached session, forcing a new session fetch on next request
func (c *Client) ClearSession() {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	c.session = nil
}

// SetSessionTTL configures the session cache time-to-live duration
func (c *Client) SetSessionTTL(ttl time.Duration) {
	c.sessionTTL = ttl
}

// SetHTTPClient sets a custom HTTP client for the JMAP client
func (c *Client) SetHTTPClient(httpClient *http.Client) {
	c.http = httpClient
}

// DownloadBlob downloads a blob (attachment) by ID and returns a ReadCloser for the content.
// The caller is responsible for closing the returned ReadCloser.
// Download URL format: {downloadUrl}/{accountId}/{blobId}/{name}?accept={type}
func (c *Client) DownloadBlob(ctx context.Context, blobID string) (io.ReadCloser, error) {
	// Ensure we have a session
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting session: %w", err)
	}

	// Build download URL
	// Format: {downloadUrl}/{accountId}/{blobId}/{name}?accept={type}
	// We use a generic name since we may not know the actual filename at this point
	downloadURL := fmt.Sprintf("%s/%s/%s/attachment", session.DownloadURL, session.AccountID, blobID)

	var lastErr error
	var resp *http.Response

	// Retry loop
	for attempt := 0; attempt <= c.retry.MaxRetries; attempt++ {
		// Create HTTP request
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating download request: %w", err)
		}

		// Add authorization header
		req.Header.Set("Authorization", "Bearer "+c.token)

		// Execute request
		resp, err = c.http.Do(req)
		if err != nil {
			// Check if error is retriable
			if !isRetriableHTTPError(err) {
				return nil, fmt.Errorf("downloading blob: %w", err)
			}

			lastErr = err
			if attempt < c.retry.MaxRetries {
				delay := c.getRetryDelay(attempt, nil)
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
				}
			}
			continue
		}

		// Check response status
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()

			// Check if status is retriable
			if !isRetriableStatus(resp.StatusCode) {
				return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
			}

			lastErr = fmt.Errorf("download failed with status %d", resp.StatusCode)
			if attempt < c.retry.MaxRetries {
				delay := c.getRetryDelay(attempt, resp)
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
				}
			}
			continue
		}

		// Success - return the body as a ReadCloser
		// The caller is responsible for closing it
		return resp.Body, nil
	}

	// All retries exhausted
	return nil, fmt.Errorf("download failed after %d retries: %w", c.retry.MaxRetries, lastErr)
}

// UploadBlobResult contains the response from a blob upload
type UploadBlobResult struct {
	AccountID string `json:"accountId"`
	BlobID    string `json:"blobId"`
	Type      string `json:"type"`
	Size      int64  `json:"size"`
}

// UploadBlob uploads binary data and returns the blob ID for use in email attachments.
// The contentType should be the MIME type of the file (e.g., "application/pdf", "image/png").
// Upload URL format: {uploadUrl}/{accountId}/
func (c *Client) UploadBlob(ctx context.Context, reader io.Reader, contentType string) (*UploadBlobResult, error) {
	if contentType == "" {
		return nil, fmt.Errorf("contentType is required")
	}

	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting session: %w", err)
	}

	// Build upload URL by replacing {accountId} placeholder
	uploadURL := strings.Replace(session.UploadURL, "{accountId}", session.AccountID, 1)

	var lastErr error
	var resp *http.Response

	// Read content into buffer for potential retries, with size limit
	limitedReader := io.LimitReader(reader, MaxUploadSize+1)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("reading upload content: %w", err)
	}

	// Check if content exceeds size limit
	if len(content) > MaxUploadSize {
		return nil, fmt.Errorf("upload content size exceeds maximum allowed size of %d bytes (50MB)", MaxUploadSize)
	}

	// Retry loop
	for attempt := 0; attempt <= c.retry.MaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(content))
		if err != nil {
			return nil, fmt.Errorf("creating upload request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", contentType)

		resp, err = c.http.Do(req)
		if err != nil {
			if !isRetriableHTTPError(err) {
				return nil, fmt.Errorf("uploading blob: %w", err)
			}

			lastErr = err
			if attempt < c.retry.MaxRetries {
				delay := c.getRetryDelay(attempt, nil)
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
				}
			}
			continue
		}

		// Defer closing the response body immediately after successful request
		defer resp.Body.Close()

		// Check response status (201 Created is success for uploads)
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)

			if !isRetriableStatus(resp.StatusCode) {
				return nil, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
			}

			lastErr = fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
			if attempt < c.retry.MaxRetries {
				delay := c.getRetryDelay(attempt, resp)
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
				}
			}
			continue
		}

		// Success - parse response

		var result UploadBlobResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("decoding upload response: %w", err)
		}

		return &result, nil
	}

	return nil, fmt.Errorf("upload failed after %d retries: %w", c.retry.MaxRetries, lastErr)
}

