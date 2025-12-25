package caldav

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

			resp, err := client.doRequest(ctx, tt.method, server.URL+"/test", tt.body, tt.contentType)

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
