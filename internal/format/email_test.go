package format

import (
	"testing"

	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
)

func TestFormatEmailAddressList(t *testing.T) {
	cases := []struct {
		name string
		in   []jmap.EmailAddress
		want string
	}{
		{"empty", nil, ""},
		{"single", []jmap.EmailAddress{{Email: "a@example.com"}}, "a@example.com"},
		{"named", []jmap.EmailAddress{{Email: "a@example.com", Name: "Alice"}}, "Alice <a@example.com>"},
		{"multi", []jmap.EmailAddress{{Email: "a@example.com"}, {Email: "b@example.com", Name: "Bob"}}, "a@example.com, Bob <b@example.com>"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatEmailAddressList(tt.in); got != tt.want {
				t.Fatalf("FormatEmailAddressList() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatEmailDate(t *testing.T) {
	if got := FormatEmailDate("2024-01-15T14:30:00Z"); got != "2024-01-15 14:30" {
		t.Fatalf("FormatEmailDate() = %q, want %q", got, "2024-01-15 14:30")
	}

	invalid := "not-a-date"
	if got := FormatEmailDate(invalid); got != invalid {
		t.Fatalf("FormatEmailDate(invalid) = %q, want %q", got, invalid)
	}
}
