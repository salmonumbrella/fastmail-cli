package caldav

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateICS_WithAttendees(t *testing.T) {
	// Create a test event with attendees
	start := time.Date(2024, 3, 15, 14, 0, 0, 0, time.UTC)
	end := time.Date(2024, 3, 15, 15, 0, 0, 0, time.UTC)

	event := &Event{
		UID:         "test-event-123@fastmail.com",
		Summary:     "Team Meeting",
		Description: "Quarterly planning session",
		Location:    "Conference Room A",
		Start:       start,
		End:         end,
		AllDay:      false,
		Organizer:   "organizer@example.com",
		Status:      "CONFIRMED",
		Attendees: []Attendee{
			{
				Email:  "attendee1@example.com",
				Name:   "Alice Smith",
				RSVP:   true,
				Status: "NEEDS-ACTION",
			},
			{
				Email:  "attendee2@example.com",
				Name:   "Bob Jones",
				RSVP:   true,
				Status: "ACCEPTED",
			},
		},
	}

	ics := event.ToICS()

	// Verify structure
	if !strings.Contains(ics, "BEGIN:VCALENDAR") {
		t.Error("Missing BEGIN:VCALENDAR")
	}
	if !strings.Contains(ics, "END:VCALENDAR") {
		t.Error("Missing END:VCALENDAR")
	}
	if !strings.Contains(ics, "BEGIN:VEVENT") {
		t.Error("Missing BEGIN:VEVENT")
	}
	if !strings.Contains(ics, "END:VEVENT") {
		t.Error("Missing END:VEVENT")
	}

	// Verify required fields
	if !strings.Contains(ics, "VERSION:2.0") {
		t.Error("Missing VERSION:2.0")
	}
	if !strings.Contains(ics, "PRODID:-//Fastmail CLI//NONSGML Event//EN") {
		t.Error("Missing PRODID")
	}
	if !strings.Contains(ics, "CALSCALE:GREGORIAN") {
		t.Error("Missing CALSCALE:GREGORIAN")
	}
	if !strings.Contains(ics, "METHOD:REQUEST") {
		t.Error("Missing METHOD:REQUEST")
	}

	// Verify event fields
	if !strings.Contains(ics, "UID:test-event-123@fastmail.com") {
		t.Error("Missing or incorrect UID")
	}
	if !strings.Contains(ics, "SUMMARY:Team Meeting") {
		t.Error("Missing or incorrect SUMMARY")
	}
	if !strings.Contains(ics, "DESCRIPTION:Quarterly planning session") {
		t.Error("Missing or incorrect DESCRIPTION")
	}
	if !strings.Contains(ics, "LOCATION:Conference Room A") {
		t.Error("Missing or incorrect LOCATION")
	}
	if !strings.Contains(ics, "STATUS:CONFIRMED") {
		t.Error("Missing or incorrect STATUS")
	}
	if !strings.Contains(ics, "DTSTART:20240315T140000Z") {
		t.Error("Missing or incorrect DTSTART")
	}
	if !strings.Contains(ics, "DTEND:20240315T150000Z") {
		t.Error("Missing or incorrect DTEND")
	}

	// Verify organizer and attendees (removing line breaks and continuation spaces for comparison)
	// Lines may be folded, so we need to unfold them for comparison
	unfoldedICS := strings.ReplaceAll(ics, "\r\n ", "")

	if !strings.Contains(unfoldedICS, "ORGANIZER;CN=organizer@example.com:mailto:organizer@example.com") {
		t.Error("Missing or incorrect ORGANIZER")
	}
	if !strings.Contains(unfoldedICS, "ATTENDEE;CUTYPE=INDIVIDUAL;ROLE=REQ-PARTICIPANT;PARTSTAT=NEEDS-ACTION;RSVP=TRUE;CN=Alice Smith:mailto:attendee1@example.com") {
		t.Error("Missing or incorrect first ATTENDEE")
	}
	if !strings.Contains(unfoldedICS, "ATTENDEE;CUTYPE=INDIVIDUAL;ROLE=REQ-PARTICIPANT;PARTSTAT=ACCEPTED;RSVP=TRUE;CN=Bob Jones:mailto:attendee2@example.com") {
		t.Error("Missing or incorrect second ATTENDEE")
	}

	// Verify CRLF line endings
	if !strings.Contains(ics, "\r\n") {
		t.Error("Missing CRLF line endings")
	}
	if strings.Contains(strings.ReplaceAll(ics, "\r\n", ""), "\n") {
		t.Error("Contains LF without CR")
	}
}

func TestGenerateICS_AllDay(t *testing.T) {
	// Create an all-day event
	start := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 3, 16, 0, 0, 0, 0, time.UTC)

	event := &Event{
		UID:     "allday-event-456@fastmail.com",
		Summary: "Company Holiday",
		Start:   start,
		End:     end,
		AllDay:  true,
		Status:  "CONFIRMED",
	}

	ics := event.ToICS()

	// Verify all-day date format (no time component, VALUE=DATE)
	if !strings.Contains(ics, "DTSTART;VALUE=DATE:20240315") {
		t.Error("Missing or incorrect all-day DTSTART format")
	}
	if !strings.Contains(ics, "DTEND;VALUE=DATE:20240316") {
		t.Error("Missing or incorrect all-day DTEND format")
	}

	// Should not contain time component
	if strings.Contains(ics, "DTSTART:20240315T") {
		t.Error("All-day event should not have time component in DTSTART")
	}
}

func TestGenerateICS_MinimalEvent(t *testing.T) {
	// Create minimal event with only required fields
	start := time.Date(2024, 3, 15, 14, 0, 0, 0, time.UTC)
	end := time.Date(2024, 3, 15, 15, 0, 0, 0, time.UTC)

	event := &Event{
		UID:     "minimal-event@fastmail.com",
		Summary: "Quick Meeting",
		Start:   start,
		End:     end,
	}

	ics := event.ToICS()

	// Verify required fields are present
	if !strings.Contains(ics, "UID:minimal-event@fastmail.com") {
		t.Error("Missing UID")
	}
	if !strings.Contains(ics, "SUMMARY:Quick Meeting") {
		t.Error("Missing SUMMARY")
	}

	// Verify optional fields are absent
	if strings.Contains(ics, "DESCRIPTION:") {
		t.Error("DESCRIPTION should not be present")
	}
	if strings.Contains(ics, "LOCATION:") {
		t.Error("LOCATION should not be present")
	}
	if strings.Contains(ics, "ORGANIZER:") {
		t.Error("ORGANIZER should not be present")
	}
	if strings.Contains(ics, "ATTENDEE:") {
		t.Error("ATTENDEE should not be present")
	}
}

func TestGenerateICS_AttendeeDefaults(t *testing.T) {
	// Create event with attendee that has minimal info
	start := time.Date(2024, 3, 15, 14, 0, 0, 0, time.UTC)
	end := time.Date(2024, 3, 15, 15, 0, 0, 0, time.UTC)

	event := &Event{
		UID:     "test-defaults@fastmail.com",
		Summary: "Test Defaults",
		Start:   start,
		End:     end,
		Attendees: []Attendee{
			{
				Email: "test@example.com",
				// Name, RSVP, and Status left empty
			},
		},
	}

	ics := event.ToICS()

	// Unfold lines for checking (lines may be folded)
	unfoldedICS := strings.ReplaceAll(ics, "\r\n ", "")

	// Verify defaults are applied
	if !strings.Contains(unfoldedICS, "PARTSTAT=NEEDS-ACTION") {
		t.Error("Should default to NEEDS-ACTION status")
	}
	if !strings.Contains(unfoldedICS, "RSVP=FALSE") {
		t.Error("Should default to RSVP=FALSE")
	}
	if !strings.Contains(unfoldedICS, "CN=test@example.com") {
		t.Error("Should default CN to email when name is empty")
	}
}

func TestEscapeICS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "backslash",
			input:    "path\\to\\file",
			expected: "path\\\\to\\\\file",
		},
		{
			name:     "semicolon",
			input:    "item1;item2;item3",
			expected: "item1\\;item2\\;item3",
		},
		{
			name:     "comma",
			input:    "Smith, John",
			expected: "Smith\\, John",
		},
		{
			name:     "newline",
			input:    "Line 1\nLine 2",
			expected: "Line 1\\nLine 2",
		},
		{
			name:     "carriage return",
			input:    "Line 1\r\nLine 2",
			expected: "Line 1\\nLine 2",
		},
		{
			name:     "multiple special chars",
			input:    "Test\\with;many,special\nchars",
			expected: "Test\\\\with\\;many\\,special\\nchars",
		},
		{
			name:     "no special chars",
			input:    "Normal text",
			expected: "Normal text",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeICS(tt.input)
			if result != tt.expected {
				t.Errorf("escapeICS(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatICSTime(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "standard datetime",
			input:    time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC),
			expected: "20240315T143045Z",
		},
		{
			name:     "midnight",
			input:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: "20240101T000000Z",
		},
		{
			name:     "end of day",
			input:    time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			expected: "20241231T235959Z",
		},
		{
			name:     "single digit month and day",
			input:    time.Date(2024, 1, 5, 9, 5, 3, 0, time.UTC),
			expected: "20240105T090503Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatICSTime(tt.input)
			if result != tt.expected {
				t.Errorf("formatICSTime(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatICSTime_Timezone(t *testing.T) {
	// Verify that non-UTC times are properly converted
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("Could not load timezone")
	}

	// 2:00 PM EST = 7:00 PM UTC (during standard time)
	input := time.Date(2024, 1, 15, 14, 0, 0, 0, location)
	result := formatICSTime(input.UTC())
	expected := "20240115T190000Z"

	if result != expected {
		t.Errorf("formatICSTime with timezone conversion = %q, want %q", result, expected)
	}
}

func TestFoldLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short line (no folding)",
			input:    "UID:test-event-123@fastmail.com",
			expected: "UID:test-event-123@fastmail.com\r\n",
		},
		{
			name:     "exactly 75 characters (no folding)",
			input:    strings.Repeat("X", 75),
			expected: strings.Repeat("X", 75) + "\r\n",
		},
		{
			name:  "76 characters (folding required)",
			input: strings.Repeat("X", 76),
			expected: strings.Repeat("X", 75) + "\r\n" +
				" X\r\n",
		},
		{
			name:  "long line requiring multiple folds",
			input: strings.Repeat("A", 160),
			expected: strings.Repeat("A", 75) + "\r\n" +
				" " + strings.Repeat("A", 74) + "\r\n" +
				" " + strings.Repeat("A", 11) + "\r\n",
		},
		{
			name:  "ATTENDEE line exceeding 75 chars",
			input: "ATTENDEE;CUTYPE=INDIVIDUAL;ROLE=REQ-PARTICIPANT;PARTSTAT=NEEDS-ACTION;RSVP=TRUE;CN=Alice Smith:mailto:attendee1@example.com",
			expected: "ATTENDEE;CUTYPE=INDIVIDUAL;ROLE=REQ-PARTICIPANT;PARTSTAT=NEEDS-ACTION;RSVP=\r\n" +
				" TRUE;CN=Alice Smith:mailto:attendee1@example.com\r\n",
		},
		{
			name:  "long DESCRIPTION",
			input: "DESCRIPTION:This is a very long description that exceeds seventy-five characters and must be folded according to RFC 5545",
			expected: "DESCRIPTION:This is a very long description that exceeds seventy-five chara\r\n" +
				" cters and must be folded according to RFC 5545\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := foldLine(tt.input)
			if result != tt.expected {
				t.Errorf("foldLine() result mismatch:\nGot:\n%q\nWant:\n%q", result, tt.expected)
			}

			// Verify all lines (except the last) are at most 75 characters + CRLF
			lines := strings.Split(strings.TrimSuffix(result, "\r\n"), "\r\n")
			for i, line := range lines {
				if i > 0 && !strings.HasPrefix(line, " ") {
					t.Errorf("Continuation line %d must start with space: %q", i, line)
				}
				if len(line) > 75 {
					t.Errorf("Line %d exceeds 75 octets: %d octets: %q", i, len(line), line)
				}
			}
		})
	}
}

// TestFoldLineDataIntegrity verifies that folding and unfolding preserves data
func TestFoldLineDataIntegrity(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "150 characters (2 folds)",
			input: strings.Repeat("B", 150),
		},
		{
			name:  "160 characters (3 folds)",
			input: strings.Repeat("A", 160),
		},
		{
			name:  "200 characters",
			input: strings.Repeat("C", 200),
		},
		{
			name:  "300 characters",
			input: strings.Repeat("D", 300),
		},
		{
			name:  "real ATTENDEE line",
			input: "ATTENDEE;CUTYPE=INDIVIDUAL;ROLE=REQ-PARTICIPANT;PARTSTAT=NEEDS-ACTION;RSVP=TRUE;CN=Alice Smith:mailto:attendee1@example.com",
		},
		{
			name:  "UTF-8 multibyte characters",
			input: "SUMMARY:会议安排 - 产品评审会 - 这是一个非常长的标题包含许多中文字符需要正确折叠而不破坏字符编码",
		},
		{
			name:  "UTF-8 emoji characters",
			input: "SUMMARY:Meeting 🎉🎊🎁🎂🎈 - This is a long title with emojis that must be folded without corrupting the emoji bytes",
		},
		{
			name:  "mixed ASCII and UTF-8",
			input: "DESCRIPTION:Mixed content 你好世界 with Chinese 日本語 and Japanese characters spread through a very long line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fold the line
			folded := foldLine(tt.input)

			// Unfold by removing CRLF + space sequences
			unfolded := strings.ReplaceAll(folded, "\r\n ", "")
			// Remove final CRLF
			unfolded = strings.TrimSuffix(unfolded, "\r\n")

			// Verify data integrity: unfold(fold(x)) == x
			if unfolded != tt.input {
				t.Errorf("Data corruption detected!\nOriginal length: %d\nUnfolded length: %d\nOriginal: %q\nUnfolded: %q",
					len(tt.input), len(unfolded), tt.input, unfolded)
			}
		})
	}
}
