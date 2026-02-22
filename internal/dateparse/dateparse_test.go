package dateparse

import (
	"testing"
	"time"
)

func TestParseDateTime_RelativeKeywords(t *testing.T) {
	loc := time.FixedZone("Test", -5*60*60)
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, loc)

	tests := []struct {
		name string
		in   string
		want time.Time
	}{
		{
			name: "today",
			in:   "today",
			want: time.Date(2025, 1, 15, 0, 0, 0, 0, loc),
		},
		{
			name: "yesterday",
			in:   "yesterday",
			want: time.Date(2025, 1, 14, 0, 0, 0, 0, loc),
		},
		{
			name: "tomorrow",
			in:   "tomorrow",
			want: time.Date(2025, 1, 16, 0, 0, 0, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDateTime(tt.in, now)
			if err != nil {
				t.Fatalf("ParseDateTime(%q) error = %v", tt.in, err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("ParseDateTime(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseDateTime_RelativeDuration(t *testing.T) {
	loc := time.FixedZone("Test", -5*60*60)
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, loc)

	tests := []struct {
		name string
		in   string
		want time.Time
	}{
		{
			name: "hours ago",
			in:   "2h ago",
			want: now.Add(-2 * time.Hour),
		},
		{
			name: "hours without ago (future)",
			in:   "2h",
			want: now.Add(2 * time.Hour),
		},
		{
			name: "days ago",
			in:   "2d ago",
			want: now.Add(-48 * time.Hour),
		},
		{
			name: "weeks ago",
			in:   "1w",
			want: now.Add(7 * 24 * time.Hour),
		},
		{
			name: "months ago",
			in:   "1mo ago",
			want: now.Add(-30 * 24 * time.Hour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDateTime(tt.in, now)
			if err != nil {
				t.Fatalf("ParseDateTime(%q) error = %v", tt.in, err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("ParseDateTime(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseDateTime_Weekday(t *testing.T) {
	loc := time.FixedZone("Test", -5*60*60)
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, loc) // Wednesday

	got, err := ParseDateTime("monday", now)
	if err != nil {
		t.Fatalf("ParseDateTime(\"monday\") error = %v", err)
	}

	want := time.Date(2025, 1, 20, 0, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("ParseDateTime(\"monday\") = %v, want %v", got, want)
	}

	got, err = ParseDateTime("next friday", now)
	if err != nil {
		t.Fatalf("ParseDateTime(\"next friday\") error = %v", err)
	}

	want = time.Date(2025, 1, 17, 0, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("ParseDateTime(\"next friday\") = %v, want %v", got, want)
	}
}

func TestParseDateTime_Absolute(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	got, err := ParseDateTime("2024-01-15T12:00:00Z", now)
	if err != nil {
		t.Fatalf("ParseDateTime RFC3339 error = %v", err)
	}

	want := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("ParseDateTime RFC3339 = %v, want %v", got, want)
	}

	got, err = ParseDateTime("2024-01-15", now)
	if err != nil {
		t.Fatalf("ParseDateTime date error = %v", err)
	}

	want = time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("ParseDateTime date = %v, want %v", got, want)
	}
}

func TestParseDateTime_Invalid(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	if _, err := ParseDateTime("not-a-date", now); err == nil {
		t.Fatalf("expected error for invalid date")
	}
	if _, err := ParseDateTime("0h ago", now); err == nil {
		t.Fatalf("expected error for invalid relative date")
	}
}
