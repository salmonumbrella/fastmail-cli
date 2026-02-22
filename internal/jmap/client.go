package jmap

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/salmonumbrella/fastmail-cli/internal/transport"
)

const (
	// DefaultBaseURL is the Fastmail JMAP API base URL
	DefaultBaseURL = "https://api.fastmail.com"

	// SessionPath is the path to the JMAP session endpoint
	SessionPath = "/jmap/session"

	// Default retry configuration values (shared with transport)
	DefaultMaxRetries   = transport.DefaultMaxRetries
	DefaultInitialDelay = transport.DefaultInitialDelay
	DefaultMaxDelay     = transport.DefaultMaxDelay

	// MaxUploadSize is the maximum size for blob uploads (50MB)
	MaxUploadSize = 50 * 1024 * 1024

	// Default circuit breaker configuration values
	DefaultCircuitBreakerThreshold  = 5
	DefaultCircuitBreakerResetAfter = 30 * time.Second
)

// RetryConfig configures retry behavior for JMAP requests.
type RetryConfig = transport.RetryConfig

// DefaultRetryConfig returns a RetryConfig with sensible defaults
func DefaultRetryConfig() RetryConfig {
	return transport.DefaultRetryConfig()
}

// circuitBreaker implements a circuit breaker pattern to prevent cascading failures
type circuitBreaker struct {
	mu          sync.Mutex
	failures    int
	lastFailure time.Time
	threshold   int           // number of failures before opening circuit
	resetAfter  time.Duration // duration after which to reset the circuit
}

// newCircuitBreaker creates a new circuit breaker with default settings
func newCircuitBreaker() *circuitBreaker {
	return &circuitBreaker{
		threshold:  DefaultCircuitBreakerThreshold,
		resetAfter: DefaultCircuitBreakerResetAfter,
	}
}

// isOpen returns true if the circuit breaker is open (blocking requests)
func (cb *circuitBreaker) isOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.failures >= cb.threshold {
		if time.Since(cb.lastFailure) > cb.resetAfter {
			// Reset circuit after timeout
			cb.failures = 0
			return false
		}
		return true
	}
	return false
}

// recordSuccess resets the failure count on successful request
func (cb *circuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
}

// recordFailure increments the failure count and updates last failure time
func (cb *circuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = time.Now()
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
	token          string
	baseURL        string
	session        *Session
	sessionFetch   time.Time
	sessionTTL     time.Duration
	sessionMu      sync.RWMutex
	http           *http.Client
	retry          RetryConfig
	circuitBreaker *circuitBreaker
}

// Compile-time interface compliance checks
var (
	_ EmailService       = (*Client)(nil)
	_ MaskedEmailService = (*Client)(nil)
	_ VacationService    = (*Client)(nil)
	_ QuotaService       = (*Client)(nil)
)

// newSecureHTTPClient creates an HTTP client with secure TLS configuration.
func newSecureHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}
}

// NewClient creates a new JMAP client with the provided API token
func NewClient(token string) *Client {
	return &Client{
		token:          token,
		baseURL:        DefaultBaseURL,
		sessionTTL:     1 * time.Hour,
		http:           newSecureHTTPClient(),
		retry:          DefaultRetryConfig(),
		circuitBreaker: newCircuitBreaker(),
	}
}

// NewClientWithBaseURL creates a new JMAP client with a custom base URL
func NewClientWithBaseURL(token, baseURL string) *Client {
	return &Client{
		token:          token,
		baseURL:        baseURL,
		sessionTTL:     1 * time.Hour,
		http:           newSecureHTTPClient(),
		retry:          DefaultRetryConfig(),
		circuitBreaker: newCircuitBreaker(),
	}
}

// SetRetryConfig sets a custom retry configuration (zero values use defaults).
func (c *Client) SetRetryConfig(cfg RetryConfig) {
	c.retry = cfg
}

// generateIdempotencyKey generates a random 16-byte hex string for idempotency
func generateIdempotencyKey() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based key if crypto/rand fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b)
}

// isWriteOperation checks if a JMAP method is a write operation that needs idempotency
func isWriteOperation(methodName string) bool {
	return strings.HasSuffix(methodName, "/set") || strings.HasSuffix(methodName, "/send")
}

// GetSession fetches the JMAP session from the server and caches it for reuse
func (c *Client) GetSession(ctx context.Context) (*Session, error) {
	// Check circuit breaker
	if c.circuitBreaker.isOpen() {
		return nil, &CircuitBreakerError{}
	}

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
	reqFn := func(ctx context.Context) (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, sessionURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating session request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Request-ID", uuid.New().String())
		return req, nil
	}

	resp, err := transport.DoWithRetry(ctx, c.http, c.retry, reqFn, func(attempt int, resp *http.Response) (bool, error) {
		if resp.StatusCode == http.StatusOK {
			return false, nil
		}
		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			c.circuitBreaker.recordFailure()
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			if attempt < c.retry.MaxRetries {
				return true, nil
			}
			retryAfter := transport.RetryDelay(c.retry, attempt, resp)
			return false, &RateLimitError{RetryAfter: retryAfter}
		}
		if transport.IsRetriableStatus(resp.StatusCode) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("fetching session: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
		return nil, transport.NewHTTPError("session request", resp, body)
	}

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

	// Extract the first account ID deterministically (Fastmail typically has one account)
	accountIDs := make([]string, 0, len(sessionData.Accounts))
	for id := range sessionData.Accounts {
		accountIDs = append(accountIDs, id)
	}
	if len(accountIDs) == 0 {
		return nil, ErrNoAccounts
	}
	sort.Strings(accountIDs)
	accountID := accountIDs[0]

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

	// Record success in circuit breaker
	c.circuitBreaker.recordSuccess()

	return c.session, nil
}

// MakeRequest executes a JMAP request and returns the response
func (c *Client) MakeRequest(ctx context.Context, req *Request) (*Response, error) {
	// Check circuit breaker
	if c.circuitBreaker.isOpen() {
		return nil, &CircuitBreakerError{}
	}

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

	// Generate idempotency key for write operations
	var idempotencyKey string
	for _, methodCall := range req.MethodCalls {
		if len(methodCall) > 0 {
			if methodName, ok := methodCall[0].(string); ok {
				if isWriteOperation(methodName) {
					idempotencyKey = generateIdempotencyKey()
					break
				}
			}
		}
	}

	reqFn := func(ctx context.Context) (*http.Request, error) {
		httpReq, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, session.APIUrl, bytes.NewReader(body))
		if reqErr != nil {
			return nil, fmt.Errorf("creating request: %w", reqErr)
		}
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-Request-ID", uuid.New().String())
		if idempotencyKey != "" {
			httpReq.Header.Set("X-Idempotency-Key", idempotencyKey)
		}
		return httpReq, nil
	}

	httpResp, err := transport.DoWithRetry(ctx, c.http, c.retry, reqFn, func(attempt int, resp *http.Response) (bool, error) {
		if resp.StatusCode == http.StatusOK {
			return false, nil
		}
		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			c.circuitBreaker.recordFailure()
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			if attempt < c.retry.MaxRetries {
				return true, nil
			}
			retryAfter := transport.RetryDelay(c.retry, attempt, resp)
			return false, &RateLimitError{RetryAfter: retryAfter}
		}
		if transport.IsRetriableStatus(resp.StatusCode) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body) //nolint:errcheck // best-effort read for error message
		return nil, transport.NewHTTPError("JMAP request", httpResp, bodyBytes)
	}

	var response Response
	if err := json.NewDecoder(httpResp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Record success in circuit breaker
	c.circuitBreaker.recordSuccess()

	return &response, nil
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
// Download URL is a template per RFC 8620: {accountId}, {blobId}, {name}, {type} placeholders.
func (c *Client) DownloadBlob(ctx context.Context, blobID string) (io.ReadCloser, error) {
	// Ensure we have a session
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting session: %w", err)
	}

	// Build download URL by replacing template placeholders (RFC 8620 / RFC 6570)
	// Template format: https://.../{accountId}/{blobId}/{name}?type={type}
	downloadURL := session.DownloadURL
	downloadURL = strings.Replace(downloadURL, "{accountId}", session.AccountID, 1)
	downloadURL = strings.Replace(downloadURL, "{blobId}", blobID, 1)
	downloadURL = strings.Replace(downloadURL, "{name}", "attachment", 1)
	downloadURL = strings.Replace(downloadURL, "{type}", "application/octet-stream", 1)

	reqFn := func(ctx context.Context) (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
		if reqErr != nil {
			return nil, fmt.Errorf("creating download request: %w", reqErr)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("X-Request-ID", uuid.New().String())
		return req, nil
	}

	resp, err := transport.DoWithRetry(ctx, c.http, c.retry, reqFn, func(_ int, resp *http.Response) (bool, error) {
		if resp.StatusCode == http.StatusOK {
			return false, nil
		}
		if transport.IsRetriableStatus(resp.StatusCode) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("downloading blob: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
		_ = resp.Body.Close()
		return nil, transport.NewHTTPError("download", resp, body)
	}

	// Success - return the body as a ReadCloser (caller closes it).
	return resp.Body, nil
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

	reqFn := func(ctx context.Context) (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(content))
		if reqErr != nil {
			return nil, fmt.Errorf("creating upload request: %w", reqErr)
		}

		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("X-Request-ID", uuid.New().String())
		return req, nil
	}

	resp, err := transport.DoWithRetry(ctx, c.http, c.retry, reqFn, func(_ int, resp *http.Response) (bool, error) {
		if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
			return false, nil
		}
		if transport.IsRetriableStatus(resp.StatusCode) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("uploading blob: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
		return nil, transport.NewHTTPError("upload", resp, body)
	}

	var result UploadBlobResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding upload response: %w", err)
	}

	return &result, nil
}
