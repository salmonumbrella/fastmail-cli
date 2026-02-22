package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func TestPrintDryRunList_Text(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	app := newTestApp()

	out := captureStdout(t, func() {
		err := printDryRunList(app, cmd, "Would delete 2 emails:", "wouldDelete", []string{"a", "b"}, nil)
		if err != nil {
			t.Fatalf("printDryRunList error: %v", err)
		}
	})

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), out)
	}
	if lines[0] != "Would delete 2 emails:" {
		t.Fatalf("unexpected header: %q", lines[0])
	}
}

func TestPrintDryRunList_JSON(t *testing.T) {
	cmd := &cobra.Command{}
	ctx := context.WithValue(context.Background(), outputModeKey, outfmt.JSON)
	cmd.SetContext(ctx)
	app := newTestApp()

	out := captureStdout(t, func() {
		err := printDryRunList(app, cmd, "ignored", "wouldMove", []string{"id1"}, map[string]any{"mailbox": "Inbox"})
		if err != nil {
			t.Fatalf("printDryRunList error: %v", err)
		}
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	if payload["dryRun"] != true {
		t.Fatalf("expected dryRun true, got %v", payload["dryRun"])
	}
	if payload["mailbox"] != "Inbox" {
		t.Fatalf("expected mailbox Inbox, got %v", payload["mailbox"])
	}
	items, ok := payload["wouldMove"].([]any)
	if !ok || len(items) != 1 || items[0] != "id1" {
		t.Fatalf("unexpected wouldMove payload: %#v", payload["wouldMove"])
	}
}
