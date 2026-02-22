package caldav

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/salmonumbrella/fastmail-cli/internal/transport"
)

const (
	// DefaultBaseURL is the Fastmail CalDAV base URL
	DefaultBaseURL = "https://caldav.fastmail.com"
)

// Client is a CalDAV client for interacting with Fastmail calendars and contacts.
// WARNING: This struct contains credentials and should never be serialized or logged.
type Client struct {
	BaseURL    string
	Username   string
	token      string // unexported - security sensitive
	httpClient *http.Client
	retry      transport.RetryConfig
}

// String implements fmt.Stringer with redacted sensitive fields.
// This prevents accidental token exposure in logs or debug output.
func (c *Client) String() string {
	return fmt.Sprintf("CalDAV{BaseURL: %s, Username: %s}", c.BaseURL, c.Username)
}

// NewClient creates a new CalDAV client with the provided credentials
func NewClient(baseURL, username, token string) *Client {
	return &Client{
		BaseURL:  baseURL,
		Username: username,
		token:    token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		retry: transport.DefaultRetryConfig(),
	}
}

// SetRetryConfig sets a custom retry configuration (zero values use defaults).
func (c *Client) SetRetryConfig(cfg transport.RetryConfig) {
	c.retry = cfg
}

// CalendarHomeURL returns the CalDAV calendar home URL for the user
// Format: {baseURL}/dav/calendars/user/{username}/
func (c *Client) CalendarHomeURL() string {
	baseURL := strings.TrimSuffix(c.BaseURL, "/")
	return fmt.Sprintf("%s/dav/calendars/user/%s/", baseURL, url.QueryEscape(c.Username))
}

// AddressBookHomeURL returns the CalDAV address book home URL for the user
// Format: {baseURL}/dav/addressbooks/user/{username}/
func (c *Client) AddressBookHomeURL() string {
	baseURL := strings.TrimSuffix(c.BaseURL, "/")
	return fmt.Sprintf("%s/dav/addressbooks/user/%s/", baseURL, url.QueryEscape(c.Username))
}

// doRequest performs an authenticated HTTP request using basic auth
// The caller is responsible for closing the response body on success.
func (c *Client) doRequest(ctx context.Context, method, url string, body io.Reader, contentType string) (*http.Response, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("reading request body: %w", err)
		}
	}

	reqFn := func(ctx context.Context) (*http.Request, error) {
		var reader io.Reader
		if bodyBytes != nil {
			reader = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reader)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		// Set Basic Auth header (username:token)
		auth := c.Username + ":" + c.token
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		req.Header.Set("Authorization", "Basic "+encodedAuth)

		// Set content type if provided
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		return req, nil
	}

	resp, err := transport.DoWithRetry(ctx, c.httpClient, c.retry, reqFn, func(_ int, resp *http.Response) (bool, error) {
		if resp.StatusCode < 400 {
			return false, nil
		}
		if transport.IsRetriableStatus(resp.StatusCode) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
		_ = resp.Body.Close()
		return nil, transport.NewHTTPError("CalDAV "+method, resp, bodyBytes)
	}

	return resp, nil
}

// CreateEvent creates a new calendar event via CalDAV PUT.
func (c *Client) CreateEvent(ctx context.Context, calendarName string, event *Event) error {
	if event.UID == "" {
		return fmt.Errorf("event UID is required")
	}

	url := fmt.Sprintf("%s%s/%s.ics", c.CalendarHomeURL(), calendarName, event.UID)
	ics := event.ToICS()

	resp, err := c.doRequest(ctx, "PUT", url, strings.NewReader(ics), "text/calendar; charset=utf-8")
	if err != nil {
		return fmt.Errorf("CalDAV request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
		return fmt.Errorf("CalDAV PUT failed: %s - %s", resp.Status, string(body))
	}

	return nil
}
