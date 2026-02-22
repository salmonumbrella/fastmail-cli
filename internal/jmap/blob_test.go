package jmap

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUploadBlob(t *testing.T) {
	tests := []struct {
		name           string
		content        []byte
		contentType    string
		serverResponse string
		serverStatus   int
		wantBlobID     string
		wantErr        bool
	}{
		{
			name:        "successful upload",
			content:     []byte("test file content"),
			contentType: "application/octet-stream",
			serverResponse: `{
				"accountId": "acc123",
				"blobId": "blob-uploaded-123",
				"type": "application/octet-stream",
				"size": 17
			}`,
			serverStatus: http.StatusCreated,
			wantBlobID:   "blob-uploaded-123",
			wantErr:      false,
		},
		{
			name:        "upload with specific content type",
			content:     []byte("PDF content"),
			contentType: "application/pdf",
			serverResponse: `{
				"accountId": "acc123",
				"blobId": "blob-pdf-456",
				"type": "application/pdf",
				"size": 11
			}`,
			serverStatus: http.StatusCreated,
			wantBlobID:   "blob-pdf-456",
			wantErr:      false,
		},
		{
			name:           "server error",
			content:        []byte("test"),
			contentType:    "application/octet-stream",
			serverResponse: `{"error": "internal error"}`,
			serverStatus:   http.StatusInternalServerError,
			wantBlobID:     "",
			wantErr:        true,
		},
		{
			name:        "exceeds size limit",
			content:     make([]byte, MaxUploadSize+1), // 50MB + 1 byte
			contentType: "application/octet-stream",
			wantBlobID:  "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server for upload endpoint
			uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if ct := r.Header.Get("Content-Type"); ct != tt.contentType {
					t.Errorf("expected Content-Type %s, got %s", tt.contentType, ct)
				}
				w.WriteHeader(tt.serverStatus)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer uploadServer.Close()

			// Create test server for session endpoint
			sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{
					"apiUrl": "` + uploadServer.URL + `",
					"uploadUrl": "` + uploadServer.URL + `/{accountId}/",
					"downloadUrl": "` + uploadServer.URL + `",
					"accounts": {"acc123": {}}
				}`))
			}))
			defer sessionServer.Close()

			client := NewClientWithBaseURL("test-token", sessionServer.URL)

			result, err := client.UploadBlob(context.Background(), bytes.NewReader(tt.content), tt.contentType)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result.BlobID != tt.wantBlobID {
				t.Errorf("got blobId %s, want %s", result.BlobID, tt.wantBlobID)
			}
		})
	}
}

func TestDownloadBlob(t *testing.T) {
	tests := []struct {
		name           string
		blobID         string
		serverResponse string
		serverStatus   int
		wantContent    string
		wantErr        bool
	}{
		{
			name:           "successful download",
			blobID:         "Gabcdef123",
			serverResponse: "file content here",
			serverStatus:   http.StatusOK,
			wantContent:    "file content here",
			wantErr:        false,
		},
		{
			name:           "blob not found",
			blobID:         "Gnonexistent",
			serverResponse: "",
			serverStatus:   http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track what URL was actually requested
			var requestedURL string

			// Create test server for download endpoint
			downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestedURL = r.URL.Path + "?" + r.URL.RawQuery
				if r.Method != http.MethodGet {
					t.Errorf("expected GET, got %s", r.Method)
				}
				w.WriteHeader(tt.serverStatus)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer downloadServer.Close()

			// Create test server for session endpoint with template URL format (RFC 8620)
			sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Use RFC 8620 template format with placeholders
				downloadURL := downloadServer.URL + "/{accountId}/{blobId}/{name}?type={type}"
				_, _ = w.Write([]byte(`{
					"apiUrl": "` + downloadServer.URL + `",
					"uploadUrl": "` + downloadServer.URL + `/{accountId}/",
					"downloadUrl": "` + downloadURL + `",
					"accounts": {"acc123": {}}
				}`))
			}))
			defer sessionServer.Close()

			client := NewClientWithBaseURL("test-token", sessionServer.URL)

			reader, err := client.DownloadBlob(context.Background(), tt.blobID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			defer reader.Close()

			content, err := io.ReadAll(reader)
			if err != nil {
				t.Errorf("failed to read response: %v", err)
				return
			}

			if string(content) != tt.wantContent {
				t.Errorf("got content %q, want %q", string(content), tt.wantContent)
			}

			// Verify template placeholders were replaced correctly
			if !strings.Contains(requestedURL, "/acc123/") {
				t.Errorf("accountId not replaced in URL: %s", requestedURL)
			}
			if !strings.Contains(requestedURL, "/"+tt.blobID+"/") {
				t.Errorf("blobId not replaced in URL: %s", requestedURL)
			}
			if strings.Contains(requestedURL, "{") {
				t.Errorf("template placeholders not replaced in URL: %s", requestedURL)
			}
		})
	}
}
