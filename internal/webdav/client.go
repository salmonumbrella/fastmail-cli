package webdav

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/salmonumbrella/fastmail-cli/internal/transport"
)

const (
	// DefaultBaseURL is the Fastmail WebDAV URL for file storage
	DefaultBaseURL = "https://myfiles.fastmail.com"
)

// Client is a WebDAV client for interacting with Fastmail file storage
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	retry      transport.RetryConfig
}

// FileInfo represents information about a file or directory
type FileInfo struct {
	Path         string    `json:"path"`
	Name         string    `json:"name"`
	IsDirectory  bool      `json:"isDirectory"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"contentType,omitempty"`
	LastModified time.Time `json:"lastModified"`
}

// multistatusResponse represents the XML response from a PROPFIND request
type multistatusResponse struct {
	XMLName   xml.Name   `xml:"multistatus"`
	Responses []response `xml:"response"`
}

// response represents a single resource in the PROPFIND response
type response struct {
	Href     string   `xml:"href"`
	PropStat propStat `xml:"propstat"`
}

// propStat contains the properties of a resource
type propStat struct {
	Prop   prop   `xml:"prop"`
	Status string `xml:"status"`
}

// prop contains the WebDAV properties
type prop struct {
	DisplayName      string       `xml:"displayname"`
	GetContentLength int64        `xml:"getcontentlength"`
	GetContentType   string       `xml:"getcontenttype"`
	GetLastModified  string       `xml:"getlastmodified"`
	ResourceType     resourceType `xml:"resourcetype"`
}

// resourceType indicates if a resource is a collection (directory)
type resourceType struct {
	Collection *struct{} `xml:"collection"`
}

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

// NewClient creates a new WebDAV client with the provided API token
func NewClient(token string) *Client {
	return &Client{
		baseURL:    DefaultBaseURL,
		httpClient: newSecureHTTPClient(),
		token:      token,
		retry:      transport.DefaultRetryConfig(),
	}
}

// NewClientWithBaseURL creates a new WebDAV client with a custom base URL
func NewClientWithBaseURL(token, baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: newSecureHTTPClient(),
		token:      token,
		retry:      transport.DefaultRetryConfig(),
	}
}

// SetRetryConfig sets a custom retry configuration (zero values use defaults).
func (c *Client) SetRetryConfig(cfg transport.RetryConfig) {
	c.retry = cfg
}

// validateRemotePath validates and cleans a remote path to prevent traversal attacks.
// It ensures the path starts with / and doesn't contain parent directory references.
func validateRemotePath(remotePath string) (string, error) {
	// Ensure path starts with /
	if !strings.HasPrefix(remotePath, "/") {
		remotePath = "/" + remotePath
	}

	// Explicit check for .. patterns to prevent path traversal attacks
	if strings.Contains(remotePath, "..") {
		return "", fmt.Errorf("invalid path: parent directory reference not allowed")
	}

	// Clean the path to resolve . components
	cleaned := path.Clean(remotePath)

	// Ensure cleaned path still starts with / (prevents traversal above root)
	if !strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("invalid path: traversal attempt detected")
	}

	return cleaned, nil
}

// List lists files and directories at the specified path
func (c *Client) List(ctx context.Context, filePath string) ([]FileInfo, error) {
	// Validate and normalize path
	validPath, err := validateRemotePath(filePath)
	if err != nil {
		return nil, err
	}
	filePath = validPath

	// Ensure trailing slash for directory listing
	if !strings.HasSuffix(filePath, "/") && filePath != "/" {
		filePath = filePath + "/"
	}

	url := c.baseURL + filePath

	// Create PROPFIND request body
	propfindBody := `<?xml version="1.0" encoding="utf-8"?>
<D:propfind xmlns:D="DAV:">
  <D:prop>
    <D:displayname/>
    <D:getcontentlength/>
    <D:getcontenttype/>
    <D:getlastmodified/>
    <D:resourcetype/>
  </D:prop>
</D:propfind>`

	reqFn := func(ctx context.Context) (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, "PROPFIND", url, bytes.NewBufferString(propfindBody))
		if reqErr != nil {
			return nil, fmt.Errorf("creating request: %w", reqErr)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", "application/xml")
		req.Header.Set("Depth", "1")
		return req, nil
	}

	resp, err := transport.DoWithRetry(ctx, c.httpClient, c.retry, reqFn, func(_ int, resp *http.Response) (bool, error) {
		if resp.StatusCode == http.StatusMultiStatus {
			return false, nil
		}
		if transport.IsRetriableStatus(resp.StatusCode) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("executing PROPFIND: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusMultiStatus {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
		return nil, transport.NewHTTPError("PROPFIND", resp, body)
	}

	// Parse XML response
	var multistatus multistatusResponse
	if err := xml.NewDecoder(resp.Body).Decode(&multistatus); err != nil {
		return nil, fmt.Errorf("decoding PROPFIND response: %w", err)
	}

	// Convert responses to FileInfo
	var files []FileInfo
	for _, r := range multistatus.Responses {
		// Skip the parent directory (same as requested path)
		if strings.TrimSuffix(r.Href, "/") == strings.TrimSuffix(filePath, "/") {
			continue
		}

		// Parse last modified time
		// If parsing fails, use zero time (server may not provide this field)
		lastModified, err := parseWebDAVTime(r.PropStat.Prop.GetLastModified)
		if err != nil {
			lastModified = time.Time{}
		}

		// Extract file name from href
		name := path.Base(strings.TrimSuffix(r.Href, "/"))

		fileInfo := FileInfo{
			Path:         r.Href,
			Name:         name,
			IsDirectory:  r.PropStat.Prop.ResourceType.Collection != nil,
			Size:         r.PropStat.Prop.GetContentLength,
			ContentType:  r.PropStat.Prop.GetContentType,
			LastModified: lastModified,
		}

		files = append(files, fileInfo)
	}

	return files, nil
}

// Upload uploads a local file to the remote path
func (c *Client) Upload(ctx context.Context, localPath, remotePath string) error {
	// Validate remote path
	validPath, err := validateRemotePath(remotePath)
	if err != nil {
		return err
	}
	remotePath = validPath

	// Open local file
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("opening local file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Get file info for size
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("getting file info: %w", err)
	}

	url := c.baseURL + remotePath

	reqFn := func(ctx context.Context) (*http.Request, error) {
		if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
			return nil, fmt.Errorf("rewinding file: %w", seekErr)
		}

		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPut, url, file)
		if reqErr != nil {
			return nil, fmt.Errorf("creating request: %w", reqErr)
		}

		req.Header.Set("Authorization", "Bearer "+c.token)
		req.ContentLength = stat.Size()
		return req, nil
	}

	resp, err := transport.DoWithRetry(ctx, c.httpClient, c.retry, reqFn, func(_ int, resp *http.Response) (bool, error) {
		if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
			return false, nil
		}
		if transport.IsRetriableStatus(resp.StatusCode) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("uploading file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
		return transport.NewHTTPError("upload", resp, body)
	}

	return nil
}

// Download downloads a remote file to the local path
func (c *Client) Download(ctx context.Context, remotePath, localPath string) error {
	// Validate remote path
	validPath, err := validateRemotePath(remotePath)
	if err != nil {
		return err
	}
	remotePath = validPath

	url := c.baseURL + remotePath

	reqFn := func(ctx context.Context) (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if reqErr != nil {
			return nil, fmt.Errorf("creating request: %w", reqErr)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		return req, nil
	}

	resp, err := transport.DoWithRetry(ctx, c.httpClient, c.retry, reqFn, func(_ int, resp *http.Response) (bool, error) {
		if resp.StatusCode == http.StatusOK {
			return false, nil
		}
		if transport.IsRetriableStatus(resp.StatusCode) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("downloading file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
		return transport.NewHTTPError("download", resp, body)
	}

	// Create local file
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("creating local file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Copy content
	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("writing file content: %w", err)
	}

	return nil
}

// Mkdir creates a directory at the specified path
func (c *Client) Mkdir(ctx context.Context, dirPath string) error {
	// Validate path
	validPath, err := validateRemotePath(dirPath)
	if err != nil {
		return err
	}
	dirPath = validPath

	// Ensure trailing slash for directory
	if !strings.HasSuffix(dirPath, "/") {
		dirPath = dirPath + "/"
	}

	url := c.baseURL + dirPath

	reqFn := func(ctx context.Context) (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, "MKCOL", url, nil)
		if reqErr != nil {
			return nil, fmt.Errorf("creating request: %w", reqErr)
		}

		req.Header.Set("Authorization", "Bearer "+c.token)
		return req, nil
	}

	resp, err := transport.DoWithRetry(ctx, c.httpClient, c.retry, reqFn, func(_ int, resp *http.Response) (bool, error) {
		if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
			return false, nil
		}
		if transport.IsRetriableStatus(resp.StatusCode) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
		// 405 Method Not Allowed typically means directory already exists
		if resp.StatusCode == http.StatusMethodNotAllowed {
			return fmt.Errorf("directory may already exist or path is invalid")
		}
		return transport.NewHTTPError("mkdir", resp, body)
	}

	return nil
}

// Delete deletes a file or directory at the specified path
func (c *Client) Delete(ctx context.Context, filePath string) error {
	// Validate path
	validPath, err := validateRemotePath(filePath)
	if err != nil {
		return err
	}
	filePath = validPath

	url := c.baseURL + filePath

	reqFn := func(ctx context.Context) (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
		if reqErr != nil {
			return nil, fmt.Errorf("creating request: %w", reqErr)
		}

		req.Header.Set("Authorization", "Bearer "+c.token)
		return req, nil
	}

	resp, err := transport.DoWithRetry(ctx, c.httpClient, c.retry, reqFn, func(_ int, resp *http.Response) (bool, error) {
		if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
			return false, nil
		}
		if transport.IsRetriableStatus(resp.StatusCode) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("deleting: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("file or directory not found")
		}
		return transport.NewHTTPError("delete", resp, body)
	}

	return nil
}

// Move moves or renames a file or directory
func (c *Client) Move(ctx context.Context, source, destination string) error {
	// Validate source path
	validSource, err := validateRemotePath(source)
	if err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	source = validSource

	// Validate destination path
	validDest, err := validateRemotePath(destination)
	if err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}
	destination = validDest

	sourceURL := c.baseURL + source
	destinationURL := c.baseURL + destination

	reqFn := func(ctx context.Context) (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, "MOVE", sourceURL, nil)
		if reqErr != nil {
			return nil, fmt.Errorf("creating request: %w", reqErr)
		}

		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Destination", destinationURL)
		req.Header.Set("Overwrite", "F") // Don't overwrite existing files
		return req, nil
	}

	resp, err := transport.DoWithRetry(ctx, c.httpClient, c.retry, reqFn, func(_ int, resp *http.Response) (bool, error) {
		if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
			return false, nil
		}
		if transport.IsRetriableStatus(resp.StatusCode) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("moving: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("source not found")
		}
		if resp.StatusCode == http.StatusPreconditionFailed {
			return fmt.Errorf("destination already exists")
		}
		return transport.NewHTTPError("move", resp, body)
	}

	return nil
}

// parseWebDAVTime parses a WebDAV date/time string
func parseWebDAVTime(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, nil
	}

	// WebDAV typically uses RFC 1123 format
	formats := []string{
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		"Mon, 02 Jan 2006 15:04:05 MST",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
}
