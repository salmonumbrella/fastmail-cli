package cmd

import (
	"testing"
	"time"
)

func TestCheckCredentialAge(t *testing.T) {
	tests := []struct {
		name    string
		created time.Time
		want    string
	}{
		{
			name:    "zero time returns empty string",
			created: time.Time{},
			want:    "",
		},
		{
			name:    "new credentials (10 days old)",
			created: time.Now().Add(-10 * 24 * time.Hour),
			want:    "",
		},
		{
			name:    "credentials at threshold (90 days) - should warn",
			created: time.Now().Add(-90 * 24 * time.Hour),
			want:    "Warning: credentials are 90 days old, consider rotating",
		},
		{
			name:    "credentials just under threshold (89 days)",
			created: time.Now().Add(-89 * 24 * time.Hour),
			want:    "",
		},
		{
			name:    "old credentials (91 days)",
			created: time.Now().Add(-91 * 24 * time.Hour),
			want:    "Warning: credentials are 91 days old, consider rotating",
		},
		{
			name:    "very old credentials (200 days)",
			created: time.Now().Add(-200 * 24 * time.Hour),
			want:    "Warning: credentials are 200 days old, consider rotating",
		},
		{
			name:    "ancient credentials (365 days)",
			created: time.Now().Add(-365 * 24 * time.Hour),
			want:    "Warning: credentials are 365 days old, consider rotating",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkCredentialAge(tt.created)
			if got != tt.want {
				t.Errorf("checkCredentialAge() = %q, want %q", got, tt.want)
			}
		})
	}
}
