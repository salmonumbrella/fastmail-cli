package cmd

import (
	"fmt"
	"strings"
	"testing"
)

func TestRootHelp_ShowsStaticHelpText(t *testing.T) {
	stdout := captureStdout(t, func() {
		captureStderr(t, func() {
			_ = Execute([]string{"--help"})
		})
	})

	// Static help text should appear
	if !strings.Contains(stdout, "fastmail - CLI for Fastmail") {
		t.Fatalf("expected static help text header, got: %q", stdout[:min(200, len(stdout))])
	}
	if !strings.Contains(stdout, "Aliases (resource -> short):") {
		t.Fatalf("expected aliases section in help text")
	}
	if !strings.Contains(stdout, "--li") {
		t.Fatalf("expected --li flag documented in help text")
	}
}

func TestSubcommandHelp_UsesCobra(t *testing.T) {
	stdout := captureStdout(t, func() {
		captureStderr(t, func() {
			_ = Execute([]string{"email", "--help"})
		})
	})

	// Subcommand help should NOT show the static text
	if strings.Contains(stdout, "fastmail - CLI for Fastmail") {
		t.Fatalf("subcommand help should not show static root help text")
	}
	// Should show Cobra-generated help
	if !strings.Contains(stdout, "Available Commands") || !strings.Contains(stdout, "email") {
		t.Fatalf("expected Cobra-generated help for subcommand, got: %q", stdout[:min(200, len(stdout))])
	}
}

func TestHelpText_EmbeddedNonEmpty(t *testing.T) {
	if len(helpText) < 100 {
		t.Fatalf("helpText should be embedded and non-trivial, got %d bytes", len(helpText))
	}
}

func TestHelpExitCodesMatchConstants(t *testing.T) {
	// Verify documented exit codes match actual constants
	codes := []int{
		ExitSuccess,
		ExitGeneral,
		ExitUsage,
		ExitAuth,
		ExitNotFound,
		ExitRateLimited,
		ExitTemporary,
		ExitCanceled,
	}

	for _, code := range codes {
		pattern := fmt.Sprintf("%d", code)
		if !strings.Contains(helpText, pattern) {
			t.Errorf("help.txt missing exit code %d", code)
		}
	}
}
