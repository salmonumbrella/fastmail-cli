package webdav

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
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

// NewClient creates a new WebDAV client with the provided API token
func NewClient(token string) *Client {
	return &Client{
		baseURL: DefaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		token: token,
	}
}

// NewClientWithBaseURL creates a new WebDAV client with a custom base URL
func NewClientWithBaseURL(token, baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		token: token,
	}
}

// List lists files and directories at the specified path
func (c *Client) List(ctx context.Context, filePath string) ([]FileInfo, error) {
	// Normalize path
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}
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

	req, err := http.NewRequestWithContext(ctx, "PROPFIND", url, bytes.NewBufferString(propfindBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Depth", "1")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing PROPFIND: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusMultiStatus {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
		return nil, fmt.Errorf("PROPFIND failed with status %d: %s", resp.StatusCode, string(body))
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

	// Normalize remote path
	if !strings.HasPrefix(remotePath, "/") {
		remotePath = "/" + remotePath
	}

	url := c.baseURL + remotePath

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, file)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.ContentLength = stat.Size()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("uploading file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Download downloads a remote file to the local path
func (c *Client) Download(ctx context.Context, remotePath, localPath string) error {
	// Normalize remote path
	if !strings.HasPrefix(remotePath, "/") {
		remotePath = "/" + remotePath
	}

	url := c.baseURL + remotePath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
		return fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))
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
	// Normalize path
	if !strings.HasPrefix(dirPath, "/") {
		dirPath = "/" + dirPath
	}
	if !strings.HasSuffix(dirPath, "/") {
		dirPath = dirPath + "/"
	}

	url := c.baseURL + dirPath

	req, err := http.NewRequestWithContext(ctx, "MKCOL", url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
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
		return fmt.Errorf("mkdir failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Delete deletes a file or directory at the specified path
func (c *Client) Delete(ctx context.Context, filePath string) error {
	// Normalize path
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	url := c.baseURL + filePath

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("deleting: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("file or directory not found")
		}
		return fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Move moves or renames a file or directory
func (c *Client) Move(ctx context.Context, source, destination string) error {
	// Normalize paths
	if !strings.HasPrefix(source, "/") {
		source = "/" + source
	}
	if !strings.HasPrefix(destination, "/") {
		destination = "/" + destination
	}

	sourceURL := c.baseURL + source
	destinationURL := c.baseURL + destination

	req, err := http.NewRequestWithContext(ctx, "MOVE", sourceURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Destination", destinationURL)
	req.Header.Set("Overwrite", "F") // Don't overwrite existing files

	resp, err := c.httpClient.Do(req)
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
		return fmt.Errorf("move failed with status %d: %s", resp.StatusCode, string(body))
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
