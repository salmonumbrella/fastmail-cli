package cmd

import (
	"testing"
	"time"
)

func TestParseEmailSearchFilter(t *testing.T) {
	now := time.Date(2026, 1, 28, 15, 4, 5, 0, time.UTC)

	tests := []struct {
		name       string
		query      string
		wantText   string
		wantAfter  string
		wantBefore string
	}{
		{
			name:       "after only",
			query:      "after:yesterday",
			wantText:   "",
			wantAfter:  "2026-01-27T00:00:00Z",
			wantBefore: "",
		},
		{
			name:       "before only",
			query:      "before:today",
			wantText:   "",
			wantAfter:  "",
			wantBefore: "2026-01-28T00:00:00Z",
		},
		{
			name:       "both after and before",
			query:      "after:2026-01-01 before:2026-01-31",
			wantText:   "",
			wantAfter:  "2026-01-01T00:00:00Z",
			wantBefore: "2026-01-31T00:00:00Z",
		},
		{
			name:       "text with after date",
			query:      "subject:meeting after:yesterday",
			wantText:   "subject:meeting",
			wantAfter:  "2026-01-27T00:00:00Z",
			wantBefore: "",
		},
		{
			name:       "text with before date",
			query:      "invoice before:2026-01-15",
			wantText:   "invoice",
			wantAfter:  "",
			wantBefore: "2026-01-15T00:00:00Z",
		},
		{
			name:       "text with both dates",
			query:      "urgent after:yesterday before:today",
			wantText:   "urgent",
			wantAfter:  "2026-01-27T00:00:00Z",
			wantBefore: "2026-01-28T00:00:00Z",
		},
		{
			name:       "date in middle of text",
			query:      "from:alice after:yesterday important",
			wantText:   "from:alice important",
			wantAfter:  "2026-01-27T00:00:00Z",
			wantBefore: "",
		},
		{
			name:       "no date tokens",
			query:      "from:alice@example.com subject:hello",
			wantText:   "from:alice@example.com subject:hello",
			wantAfter:  "",
			wantBefore: "",
		},
		{
			name:       "empty query",
			query:      "",
			wantText:   "",
			wantAfter:  "",
			wantBefore: "",
		},
		{
			name:       "rfc3339 timestamp",
			query:      "after:2026-01-15T10:30:00Z",
			wantText:   "",
			wantAfter:  "2026-01-15T10:30:00Z",
			wantBefore: "",
		},
		{
			name:       "quoted relative date",
			query:      "after:\"2h ago\"",
			wantText:   "",
			wantAfter:  "2026-01-28T13:04:05Z",
			wantBefore: "",
		},
		{
			name:       "on: stays in text (not supported)",
			query:      "on:2026-01-15",
			wantText:   "on:2026-01-15",
			wantAfter:  "",
			wantBefore: "",
		},
		{
			name:       "multiple spaces collapsed",
			query:      "hello  after:yesterday  world",
			wantText:   "hello world",
			wantAfter:  "2026-01-27T00:00:00Z",
			wantBefore: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEmailSearchFilter(tt.query, now)
			if err != nil {
				t.Fatalf("parseEmailSearchFilter error = %v", err)
			}
			if got.Text != tt.wantText {
				t.Errorf("Text = %q, want %q", got.Text, tt.wantText)
			}
			if got.After != tt.wantAfter {
				t.Errorf("After = %q, want %q", got.After, tt.wantAfter)
			}
			if got.Before != tt.wantBefore {
				t.Errorf("Before = %q, want %q", got.Before, tt.wantBefore)
			}
		})
	}
}

func TestParseEmailSearchFilter_Invalid(t *testing.T) {
	now := time.Date(2026, 1, 28, 15, 4, 5, 0, time.UTC)
	if _, err := parseEmailSearchFilter("after:not-a-date", now); err == nil {
		t.Fatalf("expected error for invalid date token")
	}
}
