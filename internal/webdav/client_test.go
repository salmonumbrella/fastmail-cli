package webdav

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-token")
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}
	if client.token != "test-token" {
		t.Errorf("expected token 'test-token', got %s", client.token)
	}
	if client.baseURL != DefaultBaseURL {
		t.Errorf("expected baseURL %s, got %s", DefaultBaseURL, client.baseURL)
	}
}

func TestNewClientWithBaseURL(t *testing.T) {
	customURL := "https://custom.example.com"
	client := NewClientWithBaseURL("test-token", customURL)
	if client.baseURL != customURL {
		t.Errorf("expected baseURL %s, got %s", customURL, client.baseURL)
	}
}

func TestList(t *testing.T) {
	// Create a test server that responds to PROPFIND
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PROPFIND" {
			t.Errorf("expected PROPFIND, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer token, got %s", r.Header.Get("Authorization"))
		}

		// Return a simple multistatus response
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusMultiStatus)
		w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:">
  <D:response>
    <D:href>/test/</D:href>
    <D:propstat>
      <D:prop>
        <D:displayname>test</D:displayname>
        <D:resourcetype><D:collection/></D:resourcetype>
        <D:getlastmodified>Mon, 18 Dec 2023 12:00:00 GMT</D:getlastmodified>
      </D:prop>
      <D:status>HTTP/1.1 200 OK</D:status>
    </D:propstat>
  </D:response>
  <D:response>
    <D:href>/test/file.txt</D:href>
    <D:propstat>
      <D:prop>
        <D:displayname>file.txt</D:displayname>
        <D:getcontentlength>1024</D:getcontentlength>
        <D:getcontenttype>text/plain</D:getcontenttype>
        <D:resourcetype/>
        <D:getlastmodified>Mon, 18 Dec 2023 13:00:00 GMT</D:getlastmodified>
      </D:prop>
      <D:status>HTTP/1.1 200 OK</D:status>
    </D:propstat>
  </D:response>
</D:multistatus>`))
	}))
	defer server.Close()

	client := NewClientWithBaseURL("test-token", server.URL)
	files, err := client.List(context.Background(), "/test")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	file := files[0]
	if file.Name != "file.txt" {
		t.Errorf("expected name 'file.txt', got %s", file.Name)
	}
	if file.Size != 1024 {
		t.Errorf("expected size 1024, got %d", file.Size)
	}
	if file.ContentType != "text/plain" {
		t.Errorf("expected content type 'text/plain', got %s", file.ContentType)
	}
	if file.IsDirectory {
		t.Error("expected file to not be a directory")
	}
}

func TestUpload(t *testing.T) {
	// Create a temporary file to upload
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "upload.txt")
	content := []byte("test content")
	if err := os.WriteFile(localFile, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a test server that receives the upload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer token, got %s", r.Header.Get("Authorization"))
		}

		// Read and verify content
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		if string(body) != string(content) {
			t.Errorf("expected content %s, got %s", string(content), string(body))
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClientWithBaseURL("test-token", server.URL)
	err := client.Upload(context.Background(), localFile, "/upload.txt")
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
}

func TestDownload(t *testing.T) {
	content := []byte("test download content")

	// Create a test server that serves the file
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer token, got %s", r.Header.Get("Authorization"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	// Create a temporary directory for the download
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "download.txt")

	client := NewClientWithBaseURL("test-token", server.URL)
	err := client.Download(context.Background(), "/remote.txt", localFile)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	// Verify downloaded content
	downloaded, err := os.ReadFile(localFile)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(downloaded) != string(content) {
		t.Errorf("expected content %s, got %s", string(content), string(downloaded))
	}
}

func TestMkdir(t *testing.T) {
	// Create a test server that handles MKCOL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "MKCOL" {
			t.Errorf("expected MKCOL, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer token, got %s", r.Header.Get("Authorization"))
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClientWithBaseURL("test-token", server.URL)
	err := client.Mkdir(context.Background(), "/newdir")
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}
}

func TestDelete(t *testing.T) {
	// Create a test server that handles DELETE
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer token, got %s", r.Header.Get("Authorization"))
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClientWithBaseURL("test-token", server.URL)
	err := client.Delete(context.Background(), "/file.txt")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestMove(t *testing.T) {
	// Create a test server that handles MOVE
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "MOVE" {
			t.Errorf("expected MOVE, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer token, got %s", r.Header.Get("Authorization"))
		}

		destination := r.Header.Get("Destination")
		if !strings.HasSuffix(destination, "/newfile.txt") {
			t.Errorf("expected destination to end with /newfile.txt, got %s", destination)
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClientWithBaseURL("test-token", server.URL)
	err := client.Move(context.Background(), "/oldfile.txt", "/newfile.txt")
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}
}

func TestParseWebDAVTime(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
		wantErr  bool
	}{
		{
			input:    "Mon, 18 Dec 2023 12:00:00 GMT",
			expected: time.Date(2023, 12, 18, 12, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			input:    "2023-12-18T12:00:00Z",
			expected: time.Date(2023, 12, 18, 12, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		result, err := parseWebDAVTime(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("expected error for input %s, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for input %s: %v", tt.input, err)
			}
			if !result.Equal(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		}
	}
}
