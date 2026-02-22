package cmd

import (
	"testing"

	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
)

func TestSanitizeTab(t *testing.T) {
	if got := outfmt.SanitizeTab("a\tb\tc"); got != "a b c" {
		t.Fatalf("outfmt.SanitizeTab() = %q, want %q", got, "a b c")
	}
	if got := outfmt.SanitizeTab("no tabs"); got != "no tabs" {
		t.Fatalf("outfmt.SanitizeTab() = %q, want %q", got, "no tabs")
	}
}
