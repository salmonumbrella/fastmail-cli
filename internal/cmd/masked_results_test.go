package cmd

import (
	"strings"
	"testing"
)

func TestPrintMaskedBulkResults_Success(t *testing.T) {
	out := captureStdout(t, func() {
		printMaskedBulkResults("enabled", 3, 0, "example.com", nil)
	})

	if strings.TrimSpace(out) != "Successfully enabled 3 aliases for example.com" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestPrintMaskedBulkResults_Failures(t *testing.T) {
	out := captureStdout(t, func() {
		printMaskedBulkResults("disabled", 1, 2, "example.com", []string{"a@example.com: boom", "b@example.com: nope"})
	})

	if !strings.Contains(out, "Partially disabled 1 aliases, 2 failed:") {
		t.Fatalf("missing header: %q", out)
	}
	if !strings.Contains(out, "  a@example.com: boom") || !strings.Contains(out, "  b@example.com: nope") {
		t.Fatalf("missing errors: %q", out)
	}
}
