package cmd

import (
	"testing"

	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
)

func TestStateAction(t *testing.T) {
	cases := []struct {
		name  string
		state jmap.MaskedEmailState
		want  string
	}{
		{"enabled", jmap.MaskedEmailEnabled, "enabled"},
		{"disabled", jmap.MaskedEmailDisabled, "disabled"},
		{"deleted", jmap.MaskedEmailDeleted, "deleted"},
		{"pending", jmap.MaskedEmailPending, "updated"},
		{"unknown", jmap.MaskedEmailState("other"), "updated"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := stateAction(tt.state); got != tt.want {
				t.Fatalf("stateAction(%q) = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

func TestStateActionVerb(t *testing.T) {
	cases := []struct {
		name  string
		state jmap.MaskedEmailState
		want  string
	}{
		{"enabled", jmap.MaskedEmailEnabled, "enable"},
		{"disabled", jmap.MaskedEmailDisabled, "disable"},
		{"deleted", jmap.MaskedEmailDeleted, "delete"},
		{"pending", jmap.MaskedEmailPending, "update"},
		{"unknown", jmap.MaskedEmailState("other"), "update"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := stateActionVerb(tt.state); got != tt.want {
				t.Fatalf("stateActionVerb(%q) = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

func TestSelectBestAlias(t *testing.T) {
	aliases := []jmap.MaskedEmail{
		{Email: "a@example.com", State: jmap.MaskedEmailDisabled},
		{Email: "b@example.com", State: jmap.MaskedEmailPending},
		{Email: "c@example.com", State: jmap.MaskedEmailEnabled},
	}

	best := selectBestAlias(aliases)
	if best == nil || best.Email != "c@example.com" {
		t.Fatalf("expected enabled alias, got %#v", best)
	}
}

func TestSelectBestAlias_PendingOverDisabled(t *testing.T) {
	aliases := []jmap.MaskedEmail{
		{Email: "a@example.com", State: jmap.MaskedEmailDisabled},
		{Email: "b@example.com", State: jmap.MaskedEmailPending},
		{Email: "c@example.com", State: jmap.MaskedEmailDeleted},
	}

	best := selectBestAlias(aliases)
	if best == nil || best.Email != "b@example.com" {
		t.Fatalf("expected pending alias, got %#v", best)
	}
}

func TestSelectBestAlias_Empty(t *testing.T) {
	if got := selectBestAlias(nil); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}
