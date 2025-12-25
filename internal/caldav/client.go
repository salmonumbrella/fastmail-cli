package caldav

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the Fastmail CalDAV base URL
	DefaultBaseURL = "https://caldav.fastmail.com"
)

// Client is a CalDAV client for interacting with Fastmail calendars and contacts
type Client struct {
	BaseURL    string
	Username   string
	Token      string
	httpClient *http.Client
}

// NewClient creates a new CalDAV client with the provided credentials
func NewClient(baseURL, username, token string) *Client {
	return &Client{
		BaseURL:  baseURL,
		Username: username,
		Token:    token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CalendarHomeURL returns the CalDAV calendar home URL for the user
// Format: {baseURL}/dav/calendars/user/{username}/
func (c *Client) CalendarHomeURL() string {
	baseURL := strings.TrimSuffix(c.BaseURL, "/")
	return fmt.Sprintf("%s/dav/calendars/user/%s/", baseURL, c.Username)
}

// AddressBookHomeURL returns the CalDAV address book home URL for the user
// Format: {baseURL}/dav/addressbooks/user/{username}/
func (c *Client) AddressBookHomeURL() string {
	baseURL := strings.TrimSuffix(c.BaseURL, "/")
	return fmt.Sprintf("%s/dav/addressbooks/user/%s/", baseURL, c.Username)
}

// doRequest performs an authenticated HTTP request using basic auth
func (c *Client) doRequest(ctx context.Context, method, url, body, contentType string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = bytes.NewBufferString(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set Basic Auth header (username:token)
	auth := c.Username + ":" + c.Token
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Authorization", "Basic "+encodedAuth)

	// Set content type if provided
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}
