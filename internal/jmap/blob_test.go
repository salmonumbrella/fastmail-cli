package jmap

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
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
