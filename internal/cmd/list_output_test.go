package cmd

import (
	"strings"
	"testing"
)

func TestPrintList(t *testing.T) {
	out := captureStdout(t, func() {
		printList("Header", []string{"a", "b"})
	})

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), out)
	}
	if lines[0] != "Header" {
		t.Fatalf("unexpected header: %q", lines[0])
	}
	if lines[1] != "  - a" || lines[2] != "  - b" {
		t.Fatalf("unexpected items: %q", lines[1:])
	}
}
