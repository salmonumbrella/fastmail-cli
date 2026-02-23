package cmd

import (
	"strings"
	"testing"
)

func TestPrintBulkResults_NoFailures_NoTarget(t *testing.T) {
	out := captureStdout(t, func() {
		printBulkResults("Deleted", "", 3, 0, nil)
	})
	if strings.TrimSpace(out) != "Deleted 3" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestPrintBulkResults_NoFailures_WithTarget(t *testing.T) {
	out := captureStdout(t, func() {
		printBulkResults("Moved", "to Inbox", 2, 0, nil)
	})
	if strings.TrimSpace(out) != "Moved 2 to Inbox" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestPrintBulkResults_WithFailures(t *testing.T) {
	out := captureStdout(t, func() {
		printBulkResults("Marked", "as read", 1, 2, map[string]string{"id1": "boom", "id2": "nope"})
	})
	if !strings.Contains(out, "Marked 1 as read, 2 failed:") {
		t.Fatalf("missing header: %q", out)
	}
	if !strings.Contains(out, "id1: boom") || !strings.Contains(out, "id2: nope") {
		t.Fatalf("missing failure lines: %q", out)
	}
}

func TestPrintBulkResults_WithFailures_NoTarget(t *testing.T) {
	out := captureStdout(t, func() {
		printBulkResults("Deleted", "", 2, 1, map[string]string{"id1": "boom"})
	})
	if !strings.Contains(out, "Deleted 2, 1 failed:") {
		t.Fatalf("missing header: %q", out)
	}
	if !strings.Contains(out, "id1: boom") {
		t.Fatalf("missing failure line: %q", out)
	}
}

func TestPrintBulkResults_WithFailures_SortedByID(t *testing.T) {
	out := captureStdout(t, func() {
		printBulkResults("Moved", "emails", 1, 2, map[string]string{
			"id9": "late",
			"id1": "early",
		})
	})

	id1Pos := strings.Index(out, "id1: early")
	id9Pos := strings.Index(out, "id9: late")
	if id1Pos == -1 || id9Pos == -1 {
		t.Fatalf("missing expected failure rows: %q", out)
	}
	if id1Pos > id9Pos {
		t.Fatalf("expected failures sorted by ID, got: %q", out)
	}
}
