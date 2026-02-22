package cmd

import (
	"runtime"
	"strings"
	"testing"

	"github.com/salmonumbrella/fastmail-cli/internal/format"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/validation"
)

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		// Valid emails
		{"simple", "user@example.com", true},
		{"with dots", "user.name@example.com", true},
		{"with plus", "user+tag@example.com", true},
		{"subdomain", "user@mail.example.com", true},
		{"numbers", "user123@example123.com", true},
		{"hyphen domain", "user@my-site.com", true},

		// Invalid emails - basic structure
		{"empty", "", false},
		{"too short", "a@", false},
		{"no at", "userexample.com", false},
		{"no domain", "user@", false},
		{"no local part", "@example.com", false},
		{"double at", "user@@example.com", false},

		// SECURITY: Injection attempts
		{"angle brackets", "<script>@example.com", false},
		{"angle brackets end", "user@example.com>", false},
		{"null byte", "user\x00@example.com", false},
		{"newline", "user\n@example.com", false},
		{"carriage return", "user\r@example.com", false},
		{"tab", "user\t@example.com", false},

		// SECURITY: Length limits (RFC 5321 max 254)
		{"near max length", strings.Repeat("a", 60) + "@" + strings.Repeat("b", 60) + "." + strings.Repeat("c", 60) + ".com", true},                                                                  // 185 chars
		{"over max length", strings.Repeat("a", 64) + "@" + strings.Repeat("b", 60) + "." + strings.Repeat("c", 60) + "." + strings.Repeat("d", 60) + "." + strings.Repeat("e", 60) + ".com", false}, // 314 chars

		// Edge cases
		{"domain no tld", "user@localhost", true},     // valid per RFC, local servers
		{"ip domain style", "user@192.168.1.1", true}, // valid per RFC 5321 (IP literals)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validation.IsValidEmail(tt.email); got != tt.want {
				t.Errorf("IsValidEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Normal filenames
		{"simple", "file.txt", "file.txt"},
		{"with spaces", "my file.txt", "my file.txt"},

		// Path traversal attacks
		{"path traversal", "../../../etc/passwd", "passwd"},
		{"mixed path", "foo/../bar/baz.txt", "baz.txt"},

		// Hidden files
		{"hidden file", ".bashrc", "bashrc"},
		{"multiple dots", "...hidden", "hidden"},

		// SECURITY: Null bytes
		{"null byte", "file\x00.txt", "file.txt"},
		{"null in middle", "fi\x00le.txt", "file.txt"},

		// SECURITY: Control characters
		{"control chars", "file\x01\x02\x03.txt", "file.txt"},
		{"tab in name", "file\tname.txt", "filename.txt"},
		{"newline in name", "file\nname.txt", "filename.txt"},

		// SECURITY: Windows reserved names
		{"con", "CON", "_CON"},
		{"con lower", "con", "_con"},
		{"con.txt", "CON.txt", "_CON.txt"},
		{"prn", "PRN", "_PRN"},
		{"aux", "AUX", "_AUX"},
		{"nul", "NUL", "_NUL"},
		{"com1", "COM1", "_COM1"},
		{"com9", "COM9", "_COM9"},
		{"lpt1", "LPT1", "_LPT1"},
		{"lpt1.txt", "LPT1.txt", "_LPT1.txt"},
		{"not reserved", "CONX", "CONX"},      // not exact match
		{"not reserved2", "MYCOM1", "MYCOM1"}, // prefix doesn't match

		// SECURITY: Whitespace edge cases
		{"leading space", " .bashrc", "bashrc"},
		{"trailing space", "file.txt ", "file.txt"},
		{"space hidden", " hidden", "hidden"},

		// Empty/dangerous names
		{"empty", "", "attachment"},
		{"dot", ".", "attachment"},
		{"dotdot", "..", "attachment"},
		{"just dots", "...", "attachment"},

		// Length limits - handled in separate test
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := format.SanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeFilename_WindowsPath(t *testing.T) {
	// filepath.Base behavior is platform-specific for backslashes
	input := "C:\\Windows\\System32\\cmd.exe"
	result := format.SanitizeFilename(input)

	if runtime.GOOS == "windows" {
		// On Windows, backslashes are path separators
		if result != "cmd.exe" {
			t.Errorf("SanitizeFilename(%q) = %q, want %q on Windows", input, result, "cmd.exe")
		}
	} else {
		// On Unix, backslashes are valid filename characters
		// The whole string is treated as a filename (this is expected behavior)
		if result != input {
			t.Errorf("SanitizeFilename(%q) = %q, want %q on Unix", input, result, input)
		}
	}
}

func TestSanitizeFilename_LengthLimit(t *testing.T) {
	// Test that very long filenames are truncated to 255 bytes
	longName := ""
	for i := 0; i < 300; i++ {
		longName += "a"
	}
	longName = longName + ".txt"

	result := format.SanitizeFilename(longName)
	if len(result) > 255 {
		t.Errorf("SanitizeFilename() returned %d bytes, want <= 255", len(result))
	}

	// Should preserve .txt extension
	if result[len(result)-4:] != ".txt" {
		t.Errorf("SanitizeFilename() did not preserve extension, got %q", result[len(result)-4:])
	}
}

func TestEmailToOutput(t *testing.T) {
	tests := []struct {
		name       string
		email      jmap.Email
		wantFrom   string
		wantTo     string
		wantUnread bool
	}{
		{
			name: "normal read email",
			email: jmap.Email{
				ID:       "test-id",
				Subject:  "Test Subject",
				From:     []jmap.EmailAddress{{Email: "sender@example.com", Name: "Sender Name"}},
				To:       []jmap.EmailAddress{{Email: "recipient@example.com"}},
				Keywords: map[string]bool{"$seen": true},
			},
			wantFrom:   "sender@example.com",
			wantTo:     "recipient@example.com",
			wantUnread: false,
		},
		{
			name: "unread email - seen false",
			email: jmap.Email{
				ID:       "test-id",
				Keywords: map[string]bool{"$seen": false},
			},
			wantFrom:   "",
			wantTo:     "",
			wantUnread: true,
		},
		{
			name: "unread email - no seen keyword",
			email: jmap.Email{
				ID:       "test-id",
				Keywords: map[string]bool{"$flagged": true},
			},
			wantFrom:   "",
			wantTo:     "",
			wantUnread: true,
		},
		{
			name: "nil keywords means unread",
			email: jmap.Email{
				ID:       "test-id",
				Keywords: nil,
			},
			wantFrom:   "",
			wantTo:     "",
			wantUnread: true,
		},
		{
			name: "empty from and to arrays",
			email: jmap.Email{
				ID:       "test-id",
				From:     []jmap.EmailAddress{},
				To:       []jmap.EmailAddress{},
				Keywords: map[string]bool{"$seen": true},
			},
			wantFrom:   "",
			wantTo:     "",
			wantUnread: false,
		},
		{
			name: "multiple from addresses uses first",
			email: jmap.Email{
				ID: "test-id",
				From: []jmap.EmailAddress{
					{Email: "first@example.com", Name: "First"},
					{Email: "second@example.com", Name: "Second"},
				},
				Keywords: map[string]bool{"$seen": true},
			},
			wantFrom:   "first@example.com",
			wantTo:     "",
			wantUnread: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := emailToOutput(tt.email)
			if out.FromEmail != tt.wantFrom {
				t.Errorf("FromEmail = %q, want %q", out.FromEmail, tt.wantFrom)
			}
			if out.ToEmail != tt.wantTo {
				t.Errorf("ToEmail = %q, want %q", out.ToEmail, tt.wantTo)
			}
			if out.IsUnread != tt.wantUnread {
				t.Errorf("IsUnread = %v, want %v", out.IsUnread, tt.wantUnread)
			}
		})
	}
}

func TestEmailsToOutput(t *testing.T) {
	emails := []jmap.Email{
		{ID: "1", From: []jmap.EmailAddress{{Email: "a@example.com"}}},
		{ID: "2", From: []jmap.EmailAddress{{Email: "b@example.com"}}},
	}

	out := emailsToOutput(emails)

	if len(out) != 2 {
		t.Fatalf("emailsToOutput() returned %d items, want 2", len(out))
	}
	if out[0].FromEmail != "a@example.com" {
		t.Errorf("out[0].FromEmail = %q, want %q", out[0].FromEmail, "a@example.com")
	}
	if out[1].FromEmail != "b@example.com" {
		t.Errorf("out[1].FromEmail = %q, want %q", out[1].FromEmail, "b@example.com")
	}
}
