package cmd

import (
	"strings"
	"testing"
)

func TestPrintCancelled(t *testing.T) {
	out := captureStderr(t, func() {
		printCancelled()
	})

	if strings.TrimSpace(out) != "Cancelled" {
		t.Fatalf("unexpected output: %q", out)
	}
}
