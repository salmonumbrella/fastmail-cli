package cmd

import (
	"github.com/salmonumbrella/fastmail-cli/internal/format"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/spf13/cobra"
)

// EmailLight is a minimal email representation for --light mode.
type EmailLight struct {
	ID        string `json:"id"`
	Subject   string `json:"subject"`
	FromEmail string `json:"fromEmail"`
	Date      string `json:"receivedAt"`
	IsUnread  bool   `json:"isUnread"`
	ThreadID  string `json:"threadId"`
	MsgCount  int    `json:"messageCount,omitempty"`
}

// emailToLight converts an Email to a light representation.
func emailToLight(e jmap.Email) EmailLight {
	fromEmail := ""
	if len(e.From) > 0 {
		fromEmail = e.From[0].Email
	}
	return EmailLight{
		ID:        e.ID,
		Subject:   format.Truncate(e.Subject, 80),
		FromEmail: fromEmail,
		Date:      e.ReceivedAt,
		IsUnread:  e.Keywords == nil || !e.Keywords["$seen"],
		ThreadID:  e.ThreadID,
	}
}

// emailsToLight converts a slice of emails to light output.
func emailsToLight(emails []jmap.Email) []EmailLight {
	out := make([]EmailLight, len(emails))
	for i, e := range emails {
		out[i] = emailToLight(e)
	}
	return out
}

// emailsToLightWithCounts converts emails to light output with thread counts.
func emailsToLightWithCounts(emails []jmap.Email, threadCounts map[string]int) []EmailLight {
	out := make([]EmailLight, len(emails))
	for i, e := range emails {
		out[i] = emailToLight(e)
		if count, ok := threadCounts[e.ThreadID]; ok {
			out[i].MsgCount = count
		}
	}
	return out
}

// ContactLight is a minimal contact representation for --light mode.
type ContactLight struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email,omitempty"`
	Phone   string `json:"phone,omitempty"`
	Company string `json:"company,omitempty"`
}

// contactToLight converts a Contact to a light representation.
func contactToLight(c jmap.Contact) ContactLight {
	email := ""
	if len(c.Emails) > 0 {
		email = c.Emails[0].Value
	}
	phone := ""
	if len(c.Phones) > 0 {
		phone = c.Phones[0].Value
	}
	return ContactLight{
		ID:      c.ID,
		Name:    format.Truncate(c.Name, 80),
		Email:   email,
		Phone:   phone,
		Company: c.Company,
	}
}

// contactsToLight converts a slice of contacts to light output.
func contactsToLight(contacts []jmap.Contact) []ContactLight {
	out := make([]ContactLight, len(contacts))
	for i, c := range contacts {
		out[i] = contactToLight(c)
	}
	return out
}

// CalendarEventLight is a minimal event representation for --light mode.
type CalendarEventLight struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Start  string `json:"start"`
	End    string `json:"end"`
	Status string `json:"status"`
}

// eventToLight converts a CalendarEvent to a light representation.
func eventToLight(e jmap.CalendarEvent) CalendarEventLight {
	return CalendarEventLight{
		ID:     e.ID,
		Title:  format.Truncate(e.Title, 80),
		Start:  formatEventTime(e.Start, e.IsAllDay),
		End:    formatEventTime(e.End, e.IsAllDay),
		Status: e.Status,
	}
}

// eventsToLight converts a slice of events to light output.
func eventsToLight(events []jmap.CalendarEvent) []CalendarEventLight {
	out := make([]CalendarEventLight, len(events))
	for i, e := range events {
		out[i] = eventToLight(e)
	}
	return out
}

// addLightFlag adds --light and --li flags to a command.
// Both point to the same boolean variable.
func addLightFlag(cmd *cobra.Command, dst *bool) {
	cmd.Flags().BoolVar(dst, "light", false, "minimal payload (saves tokens)")
	cmd.Flags().BoolVar(dst, "li", false, "minimal payload (saves tokens)")
	_ = cmd.Flags().MarkHidden("li")
}
