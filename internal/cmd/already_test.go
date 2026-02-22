package cmd

import (
	"strings"
	"testing"
)

func TestPrintAlready(t *testing.T) {
	out := captureStdout(t, func() {
		printAlready("Already done")
	})

	if strings.TrimSpace(out) != "Already done" {
		t.Fatalf("unexpected output: %q", out)
	}
}
