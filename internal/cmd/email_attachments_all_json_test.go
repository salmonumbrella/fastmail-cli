package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func TestDownloadAllAttachments_JSONMode_IsSingleJSONDocument(t *testing.T) {
	tmp := t.TempDir()

	emailID := "E1"
	atts := []jmap.Attachment{
		{BlobID: "B1", Name: "report.txt", Type: "text/plain", Size: 5},
	}

	mock := &jmap.MockEmailService{
		GetEmailAttachmentsFunc: func(ctx context.Context, id string) ([]jmap.Attachment, error) {
			if id != emailID {
				t.Fatalf("unexpected emailID: %q", id)
			}
			return atts, nil
		},
		DownloadBlobFunc: func(ctx context.Context, blobID string) (io.ReadCloser, error) {
			if blobID != "B1" {
				t.Fatalf("unexpected blobID: %q", blobID)
			}
			return io.NopCloser(bytes.NewBufferString("hello")), nil
		},
	}

	app := &App{Flags: &rootFlags{}}
	cmd := &cobra.Command{}

	ctx := context.Background()
	ctx = context.WithValue(ctx, outputModeKey, outfmt.JSON)
	ctx = context.WithValue(ctx, queryKey, "")
	cmd.SetContext(ctx)

	stdout := captureStdout(t, func() {
		if err := downloadAllAttachments(cmd, mock, app, emailID, tmp); err != nil {
			t.Fatalf("downloadAllAttachments returned error: %v", err)
		}
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("stdout is not valid JSON: %v; stdout=%q", err, stdout)
	}

	if payload["emailId"] != emailID {
		t.Fatalf("expected emailId %q, got %v", emailID, payload["emailId"])
	}

	// Ensure the file was actually written where expected.
	wantPath := filepath.Join(tmp, "report.txt")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("expected attachment file to exist at %s: %v", wantPath, err)
	}
}
