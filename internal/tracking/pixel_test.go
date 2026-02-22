package tracking

import (
	"strings"
	"testing"
)

func TestGeneratePixelURL(t *testing.T) {
	key, _ := GenerateKey()
	cfg := &Config{
		Enabled:     true,
		WorkerURL:   "https://test.workers.dev",
		TrackingKey: key,
	}

	pixelURL, blob, err := GeneratePixelURL(cfg, "test@example.com", "Test Subject")
	if err != nil {
		t.Fatalf("GeneratePixelURL: %v", err)
	}

	if !strings.HasPrefix(pixelURL, "https://test.workers.dev/p/") {
		t.Errorf("unexpected URL prefix: %s", pixelURL)
	}
	if !strings.HasSuffix(pixelURL, ".gif") {
		t.Errorf("URL should end with .gif: %s", pixelURL)
	}
	if blob == "" {
		t.Error("blob should not be empty")
	}
}

func TestGeneratePixelURLNotConfigured(t *testing.T) {
	cfg := &Config{Enabled: false}

	_, _, err := GeneratePixelURL(cfg, "test@example.com", "Subject")
	if err == nil {
		t.Error("expected error when not configured")
	}
}

func TestGeneratePixelHTML(t *testing.T) {
	html := GeneratePixelHTML("https://example.com/p/abc.gif")

	if !strings.Contains(html, `<img`) {
		t.Error("should contain img tag")
	}
	if !strings.Contains(html, `src="https://example.com/p/abc.gif"`) {
		t.Error("should contain src attribute")
	}
	if !strings.Contains(html, `width="1"`) {
		t.Error("should have width=1")
	}
}

func TestHashSubject(t *testing.T) {
	h1 := hashSubject("Test Subject")
	h2 := hashSubject("Test Subject")
	h3 := hashSubject("Different Subject")

	if h1 != h2 {
		t.Error("same subject should produce same hash")
	}
	if h1 == h3 {
		t.Error("different subjects should produce different hashes")
	}
	if len(h1) != 6 {
		t.Errorf("hash should be 6 chars, got %d", len(h1))
	}
}
