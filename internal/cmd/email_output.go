package cmd

import (
	"fmt"

	"github.com/salmonumbrella/fastmail-cli/internal/format"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
)

// EmailOutput is a flattened representation of Email for agent-friendly JSON output.
// It includes computed fields like fromEmail and isUnread that are easier to parse.
type EmailOutput struct {
	ID            string              `json:"id"`
	Subject       string              `json:"subject"`
	From          []jmap.EmailAddress `json:"from"`
	FromEmail     string              `json:"fromEmail,omitempty"`
	FromName      string              `json:"fromName,omitempty"`
	To            []jmap.EmailAddress `json:"to"`
	ToEmail       string              `json:"toEmail,omitempty"`
	CC            []jmap.EmailAddress `json:"cc"`
	ReceivedAt    string              `json:"receivedAt"`
	Preview       string              `json:"preview,omitempty"`
	HasAttachment bool                `json:"hasAttachment"`
	IsUnread      bool                `json:"isUnread"`
	ThreadID      string              `json:"threadId,omitempty"`
	Keywords      map[string]bool     `json:"keywords,omitempty"`
	MessageCount  int                 `json:"messageCount,omitempty"` // Count of messages in thread
}

// emailToOutput converts an Email to a flattened EmailOutput for JSON serialization.
func emailToOutput(e jmap.Email) EmailOutput {
	out := EmailOutput{
		ID:            e.ID,
		Subject:       e.Subject,
		From:          nilToEmpty(e.From),
		To:            nilToEmpty(e.To),
		CC:            nilToEmpty(e.CC),
		ReceivedAt:    e.ReceivedAt,
		Preview:       e.Preview,
		HasAttachment: e.HasAttachment,
		ThreadID:      e.ThreadID,
		Keywords:      e.Keywords,
	}
	// Flatten from address
	if len(e.From) > 0 {
		out.FromEmail = e.From[0].Email
		out.FromName = e.From[0].Name
	}
	// Flatten to address
	if len(e.To) > 0 {
		out.ToEmail = e.To[0].Email
	}
	// Compute isUnread from keywords (unread = $seen not present or false)
	out.IsUnread = e.Keywords == nil || !e.Keywords["$seen"]
	return out
}

// emailsToOutput converts a slice of emails to flattened output format.
func emailsToOutput(emails []jmap.Email) []EmailOutput {
	out := make([]EmailOutput, len(emails))
	for i, e := range emails {
		out[i] = emailToOutput(e)
	}
	return out
}

// emailsToOutputWithCounts converts emails to output format with thread message counts.
func emailsToOutputWithCounts(emails []jmap.Email, threadCounts map[string]int) []EmailOutput {
	out := make([]EmailOutput, len(emails))
	for i, email := range emails {
		out[i] = emailToOutput(email)
		if count, ok := threadCounts[email.ThreadID]; ok {
			out[i].MessageCount = count
		}
	}
	return out
}

func printEmailList(emails []jmap.Email, threadCounts map[string]int) {
	tw := outfmt.NewTabWriter()
	fmt.Fprintln(tw, "ID\tSUBJECT\tFROM\tDATE\tUNREAD\tTHREAD")
	for _, email := range emails {
		from := format.FormatEmailAddressList(email.From)
		date := format.FormatEmailDate(email.ReceivedAt)
		unread := ""
		if email.Keywords != nil && !email.Keywords["$seen"] {
			unread = "*"
		}
		thread := formatThreadCount(threadCounts[email.ThreadID])
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			email.ID,
			outfmt.SanitizeTab(format.Truncate(email.Subject, 50)),
			outfmt.SanitizeTab(format.Truncate(from, 30)),
			date,
			unread,
			thread,
		)
	}
	tw.Flush()
}

// nilToEmpty returns the slice as-is if non-nil, or an empty slice if nil.
// This ensures JSON serialization produces [] instead of null.
func nilToEmpty(s []jmap.EmailAddress) []jmap.EmailAddress {
	if s == nil {
		return []jmap.EmailAddress{}
	}
	return s
}

func printEmailDetails(email *jmap.Email) {
	fmt.Printf("ID:        %s\n", email.ID)
	fmt.Printf("Subject:   %s\n", email.Subject)
	fmt.Printf("From:      %s\n", format.FormatEmailAddressList(email.From))
	fmt.Printf("To:        %s\n", format.FormatEmailAddressList(email.To))
	if len(email.CC) > 0 {
		fmt.Printf("CC:        %s\n", format.FormatEmailAddressList(email.CC))
	}
	fmt.Printf("Date:      %s\n", email.ReceivedAt)
	fmt.Printf("Thread ID: %s\n", email.ThreadID)
	fmt.Printf("Attachments: %d\n", len(email.Attachments))
	fmt.Println()

	if len(email.TextBody) > 0 && len(email.BodyValues) > 0 {
		for _, part := range email.TextBody {
			if body, ok := email.BodyValues[part.PartID]; ok {
				fmt.Println(body.Value)
			}
		}
	} else if email.Preview != "" {
		fmt.Println(email.Preview)
	}
}
