package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func TestPrintMaskedDryRunSingle_JSON(t *testing.T) {
	cmd := &cobra.Command{}
	ctx := context.WithValue(context.Background(), outputModeKey, outfmt.JSON)
	cmd.SetContext(ctx)
	app := newTestApp()

	out := captureStdout(t, func() {
		err := printMaskedDryRunSingle(app, cmd, "a@example.com", jmap.MaskedEmailDisabled, jmap.MaskedEmailEnabled)
		if err != nil {
			t.Fatalf("printMaskedDryRunSingle error: %v", err)
		}
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	if payload["dry_run"] != true {
		t.Fatalf("expected dry_run true, got %v", payload["dry_run"])
	}
	if payload["email"] != "a@example.com" {
		t.Fatalf("expected email, got %v", payload["email"])
	}
	if payload["current_state"] != string(jmap.MaskedEmailDisabled) {
		t.Fatalf("expected current_state disabled, got %v", payload["current_state"])
	}
	if payload["new_state"] != string(jmap.MaskedEmailEnabled) {
		t.Fatalf("expected new_state enabled, got %v", payload["new_state"])
	}
}

func TestPrintMaskedDryRunSingle_Text(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	app := newTestApp()

	out := captureStdout(t, func() {
		err := printMaskedDryRunSingle(app, cmd, "a@example.com", jmap.MaskedEmailDisabled, jmap.MaskedEmailEnabled)
		if err != nil {
			t.Fatalf("printMaskedDryRunSingle error: %v", err)
		}
	})

	if !strings.Contains(out, "[dry-run] Would enable: a@example.com (currently disabled)") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestPrintMaskedDryRunBulk_JSON(t *testing.T) {
	cmd := &cobra.Command{}
	ctx := context.WithValue(context.Background(), outputModeKey, outfmt.JSON)
	cmd.SetContext(ctx)
	app := newTestApp()

	toUpdate := []jmap.MaskedEmail{
		{Email: "a@example.com", State: jmap.MaskedEmailDisabled},
		{Email: "b@example.com", State: jmap.MaskedEmailDisabled},
	}

	out := captureStdout(t, func() {
		err := printMaskedDryRunBulk(app, cmd, "example.com", jmap.MaskedEmailEnabled, toUpdate)
		if err != nil {
			t.Fatalf("printMaskedDryRunBulk error: %v", err)
		}
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	if payload["dry_run"] != true {
		t.Fatalf("expected dry_run true, got %v", payload["dry_run"])
	}
	if payload["domain"] != "example.com" {
		t.Fatalf("expected domain, got %v", payload["domain"])
	}
	if payload["count"] != float64(2) {
		t.Fatalf("expected count 2, got %v", payload["count"])
	}
	aliases, ok := payload["aliases"].([]any)
	if !ok || len(aliases) != 2 {
		t.Fatalf("unexpected aliases payload: %#v", payload["aliases"])
	}
}

func TestPrintMaskedDryRunBulk_Text(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	app := newTestApp()

	toUpdate := []jmap.MaskedEmail{
		{Email: "a@example.com", State: jmap.MaskedEmailDisabled},
		{Email: "b@example.com", State: jmap.MaskedEmailDisabled},
	}

	out := captureStdout(t, func() {
		err := printMaskedDryRunBulk(app, cmd, "example.com", jmap.MaskedEmailEnabled, toUpdate)
		if err != nil {
			t.Fatalf("printMaskedDryRunBulk error: %v", err)
		}
	})

	if !strings.Contains(out, "[dry-run] Would enable 2 aliases for example.com:") {
		t.Fatalf("missing header: %q", out)
	}
	if !strings.Contains(out, "  a@example.com (currently disabled)") || !strings.Contains(out, "  b@example.com (currently disabled)") {
		t.Fatalf("missing items: %q", out)
	}
}

func TestBuildMaskedDryRunAlias(t *testing.T) {
	alias := buildMaskedDryRunAlias("a@example.com", jmap.MaskedEmailDisabled, jmap.MaskedEmailEnabled)
	if alias["email"] != "a@example.com" {
		t.Fatalf("unexpected email: %v", alias["email"])
	}
	if alias["current_state"] != jmap.MaskedEmailDisabled {
		t.Fatalf("unexpected current_state: %v", alias["current_state"])
	}
	if alias["new_state"] != jmap.MaskedEmailEnabled {
		t.Fatalf("unexpected new_state: %v", alias["new_state"])
	}
}
