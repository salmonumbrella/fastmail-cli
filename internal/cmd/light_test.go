package cmd

import (
	"testing"
	"time"

	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/spf13/cobra"
)

func TestEmailToLight(t *testing.T) {
	email := jmap.Email{
		ID:       "M123",
		Subject:  "Test Subject",
		ThreadID: "T456",
		From: []jmap.EmailAddress{
			{Name: "Alice", Email: "alice@example.com"},
		},
		To: []jmap.EmailAddress{
			{Name: "Bob", Email: "bob@example.com"},
		},
		ReceivedAt:    "2025-01-15T10:00:00Z",
		Preview:       "This is a preview",
		HasAttachment: true,
		Keywords:      map[string]bool{"$seen": true},
	}

	light := emailToLight(email)

	if light.ID != "M123" {
		t.Errorf("ID = %q, want %q", light.ID, "M123")
	}
	if light.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", light.Subject, "Test Subject")
	}
	if light.FromEmail != "alice@example.com" {
		t.Errorf("FromEmail = %q, want %q", light.FromEmail, "alice@example.com")
	}
	if light.Date != "2025-01-15T10:00:00Z" {
		t.Errorf("Date = %q, want %q", light.Date, "2025-01-15T10:00:00Z")
	}
	if light.IsUnread {
		t.Error("IsUnread should be false when $seen is true")
	}
	if light.ThreadID != "T456" {
		t.Errorf("ThreadID = %q, want %q", light.ThreadID, "T456")
	}
}

func TestEmailToLight_Unread(t *testing.T) {
	email := jmap.Email{
		ID:       "M789",
		Keywords: nil, // nil keywords means unread
	}
	light := emailToLight(email)
	if !light.IsUnread {
		t.Error("IsUnread should be true when Keywords is nil")
	}
}

func TestEmailsToLightWithCounts(t *testing.T) {
	emails := []jmap.Email{
		{ID: "M1", ThreadID: "T1"},
		{ID: "M2", ThreadID: "T2"},
	}
	counts := map[string]int{"T1": 3, "T2": 1}

	lights := emailsToLightWithCounts(emails, counts)

	if len(lights) != 2 {
		t.Fatalf("len = %d, want 2", len(lights))
	}
	if lights[0].MsgCount != 3 {
		t.Errorf("lights[0].MsgCount = %d, want 3", lights[0].MsgCount)
	}
	if lights[1].MsgCount != 1 {
		t.Errorf("lights[1].MsgCount = %d, want 1", lights[1].MsgCount)
	}
}

func TestContactToLight(t *testing.T) {
	contact := jmap.Contact{
		ID:   "C1",
		Name: "John Doe",
		Emails: []jmap.ContactEmail{
			{Type: "work", Value: "john@work.com"},
			{Type: "home", Value: "john@home.com"},
		},
		Phones: []jmap.ContactPhone{
			{Type: "mobile", Value: "+1234567890"},
		},
		Company: "Acme Inc",
	}

	light := contactToLight(contact)

	if light.ID != "C1" {
		t.Errorf("ID = %q, want %q", light.ID, "C1")
	}
	if light.Name != "John Doe" {
		t.Errorf("Name = %q, want %q", light.Name, "John Doe")
	}
	if light.Email != "john@work.com" {
		t.Errorf("Email = %q, want %q (first email)", light.Email, "john@work.com")
	}
	if light.Phone != "+1234567890" {
		t.Errorf("Phone = %q, want %q", light.Phone, "+1234567890")
	}
	if light.Company != "Acme Inc" {
		t.Errorf("Company = %q, want %q", light.Company, "Acme Inc")
	}
}

func TestEventToLight(t *testing.T) {
	event := jmap.CalendarEvent{
		ID:     "E1",
		Title:  "Team Meeting",
		Start:  time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
		End:    time.Date(2025, 6, 15, 15, 0, 0, 0, time.UTC),
		Status: "confirmed",
	}

	light := eventToLight(event)

	if light.ID != "E1" {
		t.Errorf("ID = %q, want %q", light.ID, "E1")
	}
	if light.Title != "Team Meeting" {
		t.Errorf("Title = %q, want %q", light.Title, "Team Meeting")
	}
	if light.Status != "confirmed" {
		t.Errorf("Status = %q, want %q", light.Status, "confirmed")
	}
}

func TestAddLightFlag(t *testing.T) {
	app := NewApp()
	root := NewRootCmd(app)

	// findCmd looks up a subcommand by path (e.g. "email", "list").
	findCmd := func(parts ...string) *cobra.Command {
		cur := root
		for _, p := range parts {
			var found *cobra.Command
			for _, c := range cur.Commands() {
				if c.Name() == p {
					found = c
					break
				}
			}
			if found == nil {
				return nil
			}
			cur = found
		}
		return cur
	}

	// All command paths that must carry --light / --li.
	paths := [][]string{
		// Top-level shortcuts
		{"list"},
		{"get"},
		{"search"},
		{"thread"},
		// email subcommands
		{"email", "list"},
		{"email", "search"},
		{"email", "get"},
		{"email", "thread"},
		// contacts subcommands
		{"contacts", "list"},
		{"contacts", "get"},
		{"contacts", "search"},
		// calendar subcommands
		{"calendar", "events"},
		{"calendar", "event-get"},
		// draft subcommands
		{"draft", "list"},
		{"draft", "get"},
	}

	for _, path := range paths {
		name := ""
		for i, p := range path {
			if i > 0 {
				name += " "
			}
			name += p
		}

		cmd := findCmd(path...)
		if cmd == nil {
			t.Errorf("command %q not found", name)
			continue
		}

		lightFlag := cmd.Flags().Lookup("light")
		if lightFlag == nil {
			t.Errorf("command %q missing --light flag", name)
			continue
		}
		liFlag := cmd.Flags().Lookup("li")
		if liFlag == nil {
			t.Errorf("command %q missing --li flag", name)
			continue
		}
		if !liFlag.Hidden {
			t.Errorf("command %q --li flag should be hidden", name)
		}
	}
}
