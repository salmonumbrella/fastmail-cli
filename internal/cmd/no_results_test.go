package cmd

import (
	"strings"
	"testing"
)

func TestPrintNoResults(t *testing.T) {
	out := captureStderr(t, func() {
		printNoResults("No widgets found for %s", "test")
	})

	if strings.TrimSpace(out) != "No widgets found for test" {
		t.Fatalf("unexpected output: %q", out)
	}
}
