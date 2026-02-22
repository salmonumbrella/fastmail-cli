package cmd

import (
	"testing"

	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
)

func TestFormatThreadCount(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  string
	}{
		{"zero", 0, "-"},
		{"single message", 1, "-"},
		{"two messages", 2, "[2 msgs]"},
		{"three messages", 3, "[3 msgs]"},
		{"ten messages", 10, "[10 msgs]"},
		{"large thread", 100, "[100 msgs]"},
		{"negative (edge case)", -1, "-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatThreadCount(tt.count)
			if got != tt.want {
				t.Errorf("formatThreadCount(%d) = %q, want %q", tt.count, got, tt.want)
			}
		})
	}
}

func TestEmailsToOutputWithCounts(t *testing.T) {
	emails := []jmap.Email{
		{ID: "1", ThreadID: "thread-a", From: []jmap.EmailAddress{{Email: "a@example.com"}}},
		{ID: "2", ThreadID: "thread-b", From: []jmap.EmailAddress{{Email: "b@example.com"}}},
		{ID: "3", ThreadID: "thread-c", From: []jmap.EmailAddress{{Email: "c@example.com"}}},
	}

	threadCounts := map[string]int{
		"thread-a": 1,
		"thread-b": 5,
		// thread-c not in map
	}

	out := emailsToOutputWithCounts(emails, threadCounts)

	if len(out) != 3 {
		t.Fatalf("emailsToOutputWithCounts() returned %d items, want 3", len(out))
	}

	// Single message thread
	if out[0].MessageCount != 1 {
		t.Errorf("out[0].MessageCount = %d, want 1", out[0].MessageCount)
	}

	// Multi-message thread
	if out[1].MessageCount != 5 {
		t.Errorf("out[1].MessageCount = %d, want 5", out[1].MessageCount)
	}

	// Thread not in counts map should have zero value
	if out[2].MessageCount != 0 {
		t.Errorf("out[2].MessageCount = %d, want 0", out[2].MessageCount)
	}
}
