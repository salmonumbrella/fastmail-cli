package caldav

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	baseURL := "https://caldav.fastmail.com"
	username := "test@example.com"
	token := "test-token-123"

	client := NewClient(baseURL, username, token)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.BaseURL != baseURL {
		t.Errorf("BaseURL = %q, want %q", client.BaseURL, baseURL)
	}

	if client.Username != username {
		t.Errorf("Username = %q, want %q", client.Username, username)
	}

	if client.httpClient == nil {
		t.Error("httpClient is nil")
	}
}

func TestClient_String(t *testing.T) {
	baseURL := "https://caldav.fastmail.com"
	username := "test@example.com"
	token := "super-secret-token-that-should-never-appear"

	client := NewClient(baseURL, username, token)
	str := client.String()

	// Verify the string contains expected public fields
	if !strings.Contains(str, baseURL) {
		t.Errorf("String() should contain BaseURL, got: %s", str)
	}
	if !strings.Contains(str, username) {
		t.Errorf("String() should contain Username, got: %s", str)
	}

	// Verify the token is NOT in the output
	if strings.Contains(str, token) {
		t.Errorf("String() must NOT contain token for security, got: %s", str)
	}
	if strings.Contains(str, "secret") {
		t.Errorf("String() must NOT contain any part of token, got: %s", str)
	}
}

func TestClient_CalendarHomeURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		username string
		want     string
	}{
		{
			name:     "standard email",
			baseURL:  "https://caldav.fastmail.com",
			username: "user@example.com",
			want:     "https://caldav.fastmail.com/dav/calendars/user/user%40example.com/",
		},
		{
			name:     "trailing slash in baseURL",
			baseURL:  "https://caldav.fastmail.com/",
			username: "user@example.com",
			want:     "https://caldav.fastmail.com/dav/calendars/user/user%40example.com/",
		},
		{
			name:     "simple username",
			baseURL:  "https://caldav.fastmail.com",
			username: "testuser",
			want:     "https://caldav.fastmail.com/dav/calendars/user/testuser/",
		},
		{
			name:     "email with plus sign",
			baseURL:  "https://caldav.fastmail.com",
			username: "user+tag@example.com",
			want:     "https://caldav.fastmail.com/dav/calendars/user/user%2Btag%40example.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.baseURL, tt.username, "token")
			got := client.CalendarHomeURL()
			if got != tt.want {
				t.Errorf("CalendarHomeURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClient_AddressBookHomeURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		username string
		want     string
	}{
		{
			name:     "standard email",
			baseURL:  "https://caldav.fastmail.com",
			username: "user@example.com",
			want:     "https://caldav.fastmail.com/dav/addressbooks/user/user%40example.com/",
		},
		{
			name:     "trailing slash in baseURL",
			baseURL:  "https://caldav.fastmail.com/",
			username: "user@example.com",
			want:     "https://caldav.fastmail.com/dav/addressbooks/user/user%40example.com/",
		},
		{
			name:     "simple username",
			baseURL:  "https://caldav.fastmail.com",
			username: "testuser",
			want:     "https://caldav.fastmail.com/dav/addressbooks/user/testuser/",
		},
		{
			name:     "email with plus sign",
			baseURL:  "https://caldav.fastmail.com",
			username: "user+tag@example.com",
			want:     "https://caldav.fastmail.com/dav/addressbooks/user/user%2Btag%40example.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.baseURL, tt.username, "token")
			got := client.AddressBookHomeURL()
			if got != tt.want {
				t.Errorf("AddressBookHomeURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClient_doRequest(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		contentType    string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		checkRequest   func(t *testing.T, r *http.Request)
	}{
		{
			name:        "successful GET request",
			method:      "GET",
			body:        "",
			contentType: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			},
			wantErr: false,
			checkRequest: func(t *testing.T, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Method = %q, want GET", r.Method)
				}
			},
		},
		{
			name:        "successful PUT request with body",
			method:      "PUT",
			body:        "<calendar-data/>",
			contentType: "text/calendar",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				if string(body) != "<calendar-data/>" {
					t.Errorf("Body = %q, want %q", string(body), "<calendar-data/>")
				}
				if r.Header.Get("Content-Type") != "text/calendar" {
					t.Errorf("Content-Type = %q, want text/calendar", r.Header.Get("Content-Type"))
				}
				w.WriteHeader(http.StatusCreated)
			},
			wantErr: false,
			checkRequest: func(t *testing.T, r *http.Request) {
				if r.Method != "PUT" {
					t.Errorf("Method = %q, want PUT", r.Method)
				}
			},
		},
		{
			name:        "basic auth header present",
			method:      "GET",
			body:        "",
			contentType: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				auth := r.Header.Get("Authorization")
				if !strings.HasPrefix(auth, "Basic ") {
					t.Errorf("Authorization header = %q, want Basic auth", auth)
				}
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
			checkRequest: func(t *testing.T, r *http.Request) {
				auth := r.Header.Get("Authorization")
				if auth == "" {
					t.Error("Authorization header is empty")
				}
			},
		},
		{
			name:        "server error",
			method:      "GET",
			body:        "",
			contentType: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("server error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.checkRequest != nil {
					tt.checkRequest(t, r)
				}
				tt.serverResponse(w, r)
			}))
			defer server.Close()

			client := NewClient(server.URL, "testuser", "testtoken")
			ctx := context.Background()

			var bodyReader io.Reader
			if tt.body != "" {
				bodyReader = strings.NewReader(tt.body)
			}

			resp, err := client.doRequest(ctx, tt.method, server.URL+"/test", bodyReader, tt.contentType)

			if tt.wantErr {
				if err == nil {
					t.Error("doRequest() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("doRequest() error = %v, want nil", err)
				return
			}

			if resp == nil {
				t.Fatal("doRequest() returned nil response")
			}

			defer resp.Body.Close()
		})
	}
}

func TestClient_CreateEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method
		if r.Method != "PUT" {
			t.Errorf("Method = %q, want PUT", r.Method)
		}

		// Verify content-type
		ct := r.Header.Get("Content-Type")
		if ct != "text/calendar; charset=utf-8" {
			t.Errorf("Content-Type = %q, want 'text/calendar; charset=utf-8'", ct)
		}

		// Verify auth header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			t.Errorf("Authorization = %q, want Basic auth", auth)
		}

		// Verify body contains iCalendar data
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		if !strings.Contains(bodyStr, "BEGIN:VCALENDAR") {
			t.Error("Body missing BEGIN:VCALENDAR")
		}
		if !strings.Contains(bodyStr, "BEGIN:VEVENT") {
			t.Error("Body missing BEGIN:VEVENT")
		}
		if !strings.Contains(bodyStr, "UID:test-event-123") {
			t.Error("Body missing UID")
		}

		// Verify URL contains event UID
		if !strings.HasSuffix(r.URL.Path, "/test-event-123.ics") {
			t.Errorf("URL path = %q, want to end with /test-event-123.ics", r.URL.Path)
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClient(server.URL, "testuser@example.com", "testtoken")
	ctx := context.Background()

	event := &Event{
		UID:     "test-event-123",
		Summary: "Test Event",
		Start:   time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		End:     time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
	}

	err := client.CreateEvent(ctx, "Default", event)
	if err != nil {
		t.Errorf("CreateEvent() error = %v, want nil", err)
	}
}

func TestClient_CreateEvent_MissingUID(t *testing.T) {
	client := NewClient("https://caldav.example.com", "testuser", "testtoken")
	ctx := context.Background()

	event := &Event{
		Summary: "Test Event",
		Start:   time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		End:     time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
		// UID is empty
	}

	err := client.CreateEvent(ctx, "Default", event)
	if err == nil {
		t.Error("CreateEvent() error = nil, want error for missing UID")
	}

	if !strings.Contains(err.Error(), "UID is required") {
		t.Errorf("Error message = %q, want to contain 'UID is required'", err.Error())
	}
}

func TestClient_CreateEvent_ServerError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "404 Not Found",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:       "403 Forbidden",
			statusCode: http.StatusForbidden,
			wantErr:    true,
		},
		{
			name:       "201 Created (success)",
			statusCode: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:       "204 No Content (success)",
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.statusCode >= 400 {
					w.Write([]byte("error response"))
				}
			}))
			defer server.Close()

			client := NewClient(server.URL, "testuser", "testtoken")
			ctx := context.Background()

			event := &Event{
				UID:     "test-event-123",
				Summary: "Test Event",
				Start:   time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
				End:     time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
			}

			err := client.CreateEvent(ctx, "Default", event)

			if tt.wantErr && err == nil {
				t.Error("CreateEvent() error = nil, want error")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("CreateEvent() error = %v, want nil", err)
			}

			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), "CalDAV") {
					t.Errorf("Error message = %q, want to contain 'CalDAV'", err.Error())
				}
			}
		})
	}
}
