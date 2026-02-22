package caldav

import (
	"fmt"
	"strings"
	"time"
)

// Attendee represents a calendar event attendee
type Attendee struct {
	Email  string // Email address of the attendee
	Name   string // Display name of the attendee
	RSVP   bool   // Whether RSVP is requested
	Status string // NEEDS-ACTION, ACCEPTED, DECLINED, TENTATIVE
}

// Event represents a calendar event
type Event struct {
	UID         string     // Unique identifier for the event
	Summary     string     // Event title
	Description string     // Event description
	Location    string     // Event location
	Start       time.Time  // Start time
	End         time.Time  // End time
	AllDay      bool       // Whether this is an all-day event
	Organizer   string     // Organizer email address
	Attendees   []Attendee // List of attendees
	Status      string     // CONFIRMED, TENTATIVE, CANCELLED
}

// foldLine folds a line at 75 octets per RFC 5545 section 3.1.
// Continuation lines start with a single space.
// This implementation handles UTF-8 correctly by folding at byte boundaries
// without splitting multi-byte characters.
func foldLine(line string) string {
	const maxLen = 75
	bytes := []byte(line)
	if len(bytes) <= maxLen {
		return line + "\r\n"
	}

	var result strings.Builder
	i := 0

	// First line: up to 75 bytes, respecting UTF-8 boundaries
	end := findSafeByteEnd(bytes, 0, maxLen)
	result.Write(bytes[:end])
	result.WriteString("\r\n")
	i = end

	// Continuation lines: 74 bytes each (after leading space), respecting UTF-8
	for i < len(bytes) {
		result.WriteString(" ")                   // Leading space for continuation
		end = findSafeByteEnd(bytes, i, maxLen-1) // 74 bytes of content
		result.Write(bytes[i:end])
		result.WriteString("\r\n")
		i = end
	}

	return result.String()
}

// findSafeByteEnd finds a safe byte index to split at, respecting UTF-8 boundaries.
// It returns the largest index <= start+maxBytes that doesn't split a UTF-8 character.
func findSafeByteEnd(data []byte, start, maxBytes int) int {
	end := start + maxBytes
	if end >= len(data) {
		return len(data)
	}

	// Walk back to find the start of the current UTF-8 character
	// UTF-8 continuation bytes have the form 10xxxxxx (0x80-0xBF)
	for end > start && data[end]&0xC0 == 0x80 {
		end--
	}

	return end
}

// ToICS generates an iCalendar format string from the event
func (e *Event) ToICS() string {
	var sb strings.Builder

	// VCALENDAR wrapper
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//Fastmail CLI//NONSGML Event//EN\r\n")
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:REQUEST\r\n")

	// VEVENT
	sb.WriteString("BEGIN:VEVENT\r\n")
	sb.WriteString(foldLine(fmt.Sprintf("UID:%s", e.UID)))
	fmt.Fprintf(&sb, "DTSTAMP:%s\r\n", formatICSTime(time.Now().UTC()))

	// Handle all-day events differently
	if e.AllDay {
		fmt.Fprintf(&sb, "DTSTART;VALUE=DATE:%s\r\n", e.Start.Format("20060102"))
		fmt.Fprintf(&sb, "DTEND;VALUE=DATE:%s\r\n", e.End.Format("20060102"))
	} else {
		fmt.Fprintf(&sb, "DTSTART:%s\r\n", formatICSTime(e.Start.UTC()))
		fmt.Fprintf(&sb, "DTEND:%s\r\n", formatICSTime(e.End.UTC()))
	}

	sb.WriteString(foldLine(fmt.Sprintf("SUMMARY:%s", escapeICS(e.Summary))))

	// Optional fields
	if e.Description != "" {
		sb.WriteString(foldLine(fmt.Sprintf("DESCRIPTION:%s", escapeICS(e.Description))))
	}

	if e.Location != "" {
		sb.WriteString(foldLine(fmt.Sprintf("LOCATION:%s", escapeICS(e.Location))))
	}

	if e.Status != "" {
		fmt.Fprintf(&sb, "STATUS:%s\r\n", e.Status)
	}

	// Organizer
	if e.Organizer != "" {
		sb.WriteString(foldLine(fmt.Sprintf("ORGANIZER;CN=%s:mailto:%s",
			escapeICS(e.Organizer), e.Organizer)))
	}

	// Attendees
	for _, attendee := range e.Attendees {
		var rsvpStr string
		if attendee.RSVP {
			rsvpStr = "TRUE"
		} else {
			rsvpStr = "FALSE"
		}

		status := attendee.Status
		if status == "" {
			status = "NEEDS-ACTION"
		}

		name := attendee.Name
		if name == "" {
			name = attendee.Email
		}

		sb.WriteString(foldLine(fmt.Sprintf("ATTENDEE;CUTYPE=INDIVIDUAL;ROLE=REQ-PARTICIPANT;PARTSTAT=%s;RSVP=%s;CN=%s:mailto:%s",
			status, rsvpStr, escapeICS(name), attendee.Email)))
	}

	sb.WriteString("END:VEVENT\r\n")
	sb.WriteString("END:VCALENDAR\r\n")

	return sb.String()
}

// formatICSTime formats a time.Time as an iCalendar datetime string (UTC)
// Format: YYYYMMDDTHHmmssZ
func formatICSTime(t time.Time) string {
	return t.Format("20060102T150405Z")
}

// escapeICS escapes special characters in iCalendar text values
// Escapes: backslash, semicolon, comma, and newlines
func escapeICS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
