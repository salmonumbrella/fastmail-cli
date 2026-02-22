package jmap

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestParseAddresses(t *testing.T) {
	tests := []struct {
		name     string
		input    []any
		expected []EmailAddress
	}{
		{
			name: "single address with name",
			input: []any{
				map[string]any{
					"name":  "John Doe",
					"email": "john@example.com",
				},
			},
			expected: []EmailAddress{
				{Name: "John Doe", Email: "john@example.com"},
			},
		},
		{
			name: "single address without name",
			input: []any{
				map[string]any{
					"email": "jane@example.com",
				},
			},
			expected: []EmailAddress{
				{Name: "", Email: "jane@example.com"},
			},
		},
		{
			name: "multiple addresses",
			input: []any{
				map[string]any{
					"name":  "Alice Smith",
					"email": "alice@example.com",
				},
				map[string]any{
					"email": "bob@example.com",
				},
			},
			expected: []EmailAddress{
				{Name: "Alice Smith", Email: "alice@example.com"},
				{Name: "", Email: "bob@example.com"},
			},
		},
		{
			name:     "empty list",
			input:    []any{},
			expected: []EmailAddress{},
		},
		{
			name: "invalid item in list",
			input: []any{
				map[string]any{
					"name":  "Valid User",
					"email": "valid@example.com",
				},
				"invalid string",
				map[string]any{
					"email": "another@example.com",
				},
			},
			expected: []EmailAddress{
				{Name: "Valid User", Email: "valid@example.com"},
				{Name: "", Email: "another@example.com"},
			},
		},
		{
			name: "missing email field",
			input: []any{
				map[string]any{
					"name": "No Email",
				},
			},
			expected: []EmailAddress{
				{Name: "No Email", Email: ""},
			},
		},
		{
			name: "wrong field types",
			input: []any{
				map[string]any{
					"name":  123,
					"email": true,
				},
			},
			expected: []EmailAddress{
				{Name: "", Email: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAddresses(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseAddresses() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseBodyParts(t *testing.T) {
	tests := []struct {
		name     string
		input    []any
		expected []BodyPart
	}{
		{
			name: "single body part",
			input: []any{
				map[string]any{
					"partId": "text-part",
					"type":   "text/plain",
				},
			},
			expected: []BodyPart{
				{PartID: "text-part", Type: "text/plain"},
			},
		},
		{
			name: "multiple body parts",
			input: []any{
				map[string]any{
					"partId": "text-part",
					"type":   "text/plain",
				},
				map[string]any{
					"partId": "html-part",
					"type":   "text/html",
				},
			},
			expected: []BodyPart{
				{PartID: "text-part", Type: "text/plain"},
				{PartID: "html-part", Type: "text/html"},
			},
		},
		{
			name:     "empty list",
			input:    []any{},
			expected: []BodyPart{},
		},
		{
			name: "missing fields",
			input: []any{
				map[string]any{
					"partId": "incomplete",
				},
			},
			expected: []BodyPart{
				{PartID: "incomplete", Type: ""},
			},
		},
		{
			name: "invalid item in list",
			input: []any{
				map[string]any{
					"partId": "valid",
					"type":   "text/plain",
				},
				"invalid",
				42,
			},
			expected: []BodyPart{
				{PartID: "valid", Type: "text/plain"},
			},
		},
		{
			name: "wrong field types",
			input: []any{
				map[string]any{
					"partId": 123,
					"type":   false,
				},
			},
			expected: []BodyPart{
				{PartID: "", Type: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBodyParts(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseBodyParts() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected *Email
	}{
		{
			name: "complete email",
			input: map[string]any{
				"id":            "email123",
				"threadId":      "thread456",
				"subject":       "Test Email",
				"receivedAt":    "2025-01-15T10:30:00Z",
				"preview":       "This is a preview",
				"hasAttachment": true,
				"from": []any{
					map[string]any{
						"name":  "Sender Name",
						"email": "sender@example.com",
					},
				},
				"to": []any{
					map[string]any{
						"name":  "Recipient Name",
						"email": "recipient@example.com",
					},
				},
				"cc": []any{
					map[string]any{
						"email": "cc@example.com",
					},
				},
				"bcc": []any{
					map[string]any{
						"email": "bcc@example.com",
					},
				},
				"keywords": map[string]any{
					"$seen":    true,
					"$flagged": false,
				},
				"mailboxIds": map[string]any{
					"inbox-123": true,
				},
				"bodyValues": map[string]any{
					"text": map[string]any{
						"value": "Email body content",
					},
				},
				"textBody": []any{
					map[string]any{
						"partId": "text-part",
						"type":   "text/plain",
					},
				},
				"htmlBody": []any{
					map[string]any{
						"partId": "html-part",
						"type":   "text/html",
					},
				},
				"attachments": []any{
					map[string]any{
						"partId": "att-1",
						"blobId": "blob-123",
						"name":   "document.pdf",
						"type":   "application/pdf",
						"size":   float64(12345),
					},
				},
			},
			expected: &Email{
				ID:            "email123",
				ThreadID:      "thread456",
				Subject:       "Test Email",
				ReceivedAt:    "2025-01-15T10:30:00Z",
				Preview:       "This is a preview",
				HasAttachment: true,
				From: []EmailAddress{
					{Name: "Sender Name", Email: "sender@example.com"},
				},
				To: []EmailAddress{
					{Name: "Recipient Name", Email: "recipient@example.com"},
				},
				CC: []EmailAddress{
					{Name: "", Email: "cc@example.com"},
				},
				BCC: []EmailAddress{
					{Name: "", Email: "bcc@example.com"},
				},
				ReplyTo:    []EmailAddress{},
				MessageID:  []string{},
				InReplyTo:  []string{},
				References: []string{},
				Keywords: map[string]bool{
					"$seen":    true,
					"$flagged": false,
				},
				MailboxIDs: map[string]bool{
					"inbox-123": true,
				},
				BodyValues: map[string]BodyValue{
					"text": {Value: "Email body content"},
				},
				TextBody: []BodyPart{
					{PartID: "text-part", Type: "text/plain"},
				},
				HTMLBody: []BodyPart{
					{PartID: "html-part", Type: "text/html"},
				},
				Attachments: []Attachment{
					{
						PartID: "att-1",
						BlobID: "blob-123",
						Name:   "document.pdf",
						Type:   "application/pdf",
						Size:   12345,
					},
				},
			},
		},
		{
			name: "minimal email",
			input: map[string]any{
				"id":         "minimal-email",
				"subject":    "Minimal",
				"receivedAt": "2025-01-15T12:00:00Z",
			},
			expected: &Email{
				ID:            "minimal-email",
				ThreadID:      "",
				Subject:       "Minimal",
				ReceivedAt:    "2025-01-15T12:00:00Z",
				Preview:       "",
				HasAttachment: false,
				From:          []EmailAddress{},
				To:            []EmailAddress{},
				CC:            []EmailAddress{},
				BCC:           []EmailAddress{},
				ReplyTo:       []EmailAddress{},
				TextBody:      []BodyPart{},
				HTMLBody:      []BodyPart{},
				Attachments:   []Attachment{},
				MessageID:     []string{},
				InReplyTo:     []string{},
				References:    []string{},
			},
		},
		{
			name: "email with invalid keywords",
			input: map[string]any{
				"id":         "email-with-bad-keywords",
				"subject":    "Test",
				"receivedAt": "2025-01-15T12:00:00Z",
				"keywords": map[string]any{
					"$seen":   true,
					"invalid": "not a bool",
				},
			},
			expected: &Email{
				ID:         "email-with-bad-keywords",
				Subject:    "Test",
				ReceivedAt: "2025-01-15T12:00:00Z",
				Keywords: map[string]bool{
					"$seen": true,
				},
				From:        []EmailAddress{},
				To:          []EmailAddress{},
				CC:          []EmailAddress{},
				BCC:         []EmailAddress{},
				ReplyTo:     []EmailAddress{},
				TextBody:    []BodyPart{},
				HTMLBody:    []BodyPart{},
				Attachments: []Attachment{},
				MessageID:   []string{},
				InReplyTo:   []string{},
				References:  []string{},
			},
		},
		{
			name: "email with invalid mailboxIds",
			input: map[string]any{
				"id":         "email-with-bad-mailboxes",
				"subject":    "Test",
				"receivedAt": "2025-01-15T12:00:00Z",
				"mailboxIds": map[string]any{
					"mailbox-1": true,
					"mailbox-2": "not a bool",
				},
			},
			expected: &Email{
				ID:         "email-with-bad-mailboxes",
				Subject:    "Test",
				ReceivedAt: "2025-01-15T12:00:00Z",
				MailboxIDs: map[string]bool{
					"mailbox-1": true,
				},
				From:        []EmailAddress{},
				To:          []EmailAddress{},
				CC:          []EmailAddress{},
				BCC:         []EmailAddress{},
				ReplyTo:     []EmailAddress{},
				TextBody:    []BodyPart{},
				HTMLBody:    []BodyPart{},
				Attachments: []Attachment{},
				MessageID:   []string{},
				InReplyTo:   []string{},
				References:  []string{},
			},
		},
		{
			name: "email with invalid bodyValues",
			input: map[string]any{
				"id":         "email-with-bad-body",
				"subject":    "Test",
				"receivedAt": "2025-01-15T12:00:00Z",
				"bodyValues": map[string]any{
					"text":    map[string]any{"value": "Valid body"},
					"invalid": "not a map",
				},
			},
			expected: &Email{
				ID:         "email-with-bad-body",
				Subject:    "Test",
				ReceivedAt: "2025-01-15T12:00:00Z",
				BodyValues: map[string]BodyValue{
					"text": {Value: "Valid body"},
				},
				From:        []EmailAddress{},
				To:          []EmailAddress{},
				CC:          []EmailAddress{},
				BCC:         []EmailAddress{},
				ReplyTo:     []EmailAddress{},
				TextBody:    []BodyPart{},
				HTMLBody:    []BodyPart{},
				Attachments: []Attachment{},
				MessageID:   []string{},
				InReplyTo:   []string{},
				References:  []string{},
			},
		},
		{
			name: "email with multiple attachments",
			input: map[string]any{
				"id":         "email-with-attachments",
				"subject":    "Files attached",
				"receivedAt": "2025-01-15T12:00:00Z",
				"attachments": []any{
					map[string]any{
						"partId": "att-1",
						"blobId": "blob-1",
						"name":   "doc1.pdf",
						"type":   "application/pdf",
						"size":   float64(1024),
					},
					map[string]any{
						"partId": "att-2",
						"blobId": "blob-2",
						"name":   "image.png",
						"type":   "image/png",
						"size":   float64(2048),
					},
				},
			},
			expected: &Email{
				ID:         "email-with-attachments",
				Subject:    "Files attached",
				ReceivedAt: "2025-01-15T12:00:00Z",
				From:       []EmailAddress{},
				To:         []EmailAddress{},
				CC:         []EmailAddress{},
				BCC:        []EmailAddress{},
				ReplyTo:    []EmailAddress{},
				TextBody:   []BodyPart{},
				HTMLBody:   []BodyPart{},
				MessageID:  []string{},
				InReplyTo:  []string{},
				References: []string{},
				Attachments: []Attachment{
					{
						PartID: "att-1",
						BlobID: "blob-1",
						Name:   "doc1.pdf",
						Type:   "application/pdf",
						Size:   1024,
					},
					{
						PartID: "att-2",
						BlobID: "blob-2",
						Name:   "image.png",
						Type:   "image/png",
						Size:   2048,
					},
				},
			},
		},
		{
			name:  "empty email data",
			input: map[string]any{},
			expected: &Email{
				ID:            "",
				ThreadID:      "",
				Subject:       "",
				ReceivedAt:    "",
				Preview:       "",
				HasAttachment: false,
				From:          []EmailAddress{},
				To:            []EmailAddress{},
				CC:            []EmailAddress{},
				BCC:           []EmailAddress{},
				ReplyTo:       []EmailAddress{},
				TextBody:      []BodyPart{},
				HTMLBody:      []BodyPart{},
				Attachments:   []Attachment{},
				MessageID:     []string{},
				InReplyTo:     []string{},
				References:    []string{},
			},
		},
		{
			name: "email with wrong field types",
			input: map[string]any{
				"id":            123,
				"subject":       true,
				"receivedAt":    []string{"wrong"},
				"hasAttachment": "yes",
				"from":          "not an array",
				"keywords":      []any{"not", "a", "map"},
			},
			expected: &Email{
				ID:            "",
				ThreadID:      "",
				Subject:       "",
				ReceivedAt:    "",
				Preview:       "",
				HasAttachment: false,
				From:          []EmailAddress{},
				To:            []EmailAddress{},
				CC:            []EmailAddress{},
				BCC:           []EmailAddress{},
				ReplyTo:       []EmailAddress{},
				TextBody:      []BodyPart{},
				HTMLBody:      []BodyPart{},
				Attachments:   []Attachment{},
				MessageID:     []string{},
				InReplyTo:     []string{},
				References:    []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEmail(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseEmail() mismatch\ngot:  %+v\nwant: %+v", got, tt.expected)
			}
		})
	}
}

func TestParseEmailList(t *testing.T) {
	tests := []struct {
		name        string
		input       MethodResponse
		expected    []Email
		expectError bool
	}{
		{
			name: "valid email list",
			input: MethodResponse{
				"Email/get",
				map[string]any{
					"list": []any{
						map[string]any{
							"id":         "email1",
							"subject":    "First Email",
							"receivedAt": "2025-01-15T10:00:00Z",
						},
						map[string]any{
							"id":         "email2",
							"subject":    "Second Email",
							"receivedAt": "2025-01-15T11:00:00Z",
						},
					},
				},
				"emails",
			},
			expected: []Email{
				{
					ID: "email1", Subject: "First Email", ReceivedAt: "2025-01-15T10:00:00Z",
					From: []EmailAddress{}, To: []EmailAddress{}, CC: []EmailAddress{},
					BCC: []EmailAddress{}, ReplyTo: []EmailAddress{},
					TextBody: []BodyPart{}, HTMLBody: []BodyPart{}, Attachments: []Attachment{},
					MessageID: []string{}, InReplyTo: []string{}, References: []string{},
				},
				{
					ID: "email2", Subject: "Second Email", ReceivedAt: "2025-01-15T11:00:00Z",
					From: []EmailAddress{}, To: []EmailAddress{}, CC: []EmailAddress{},
					BCC: []EmailAddress{}, ReplyTo: []EmailAddress{},
					TextBody: []BodyPart{}, HTMLBody: []BodyPart{}, Attachments: []Attachment{},
					MessageID: []string{}, InReplyTo: []string{}, References: []string{},
				},
			},
			expectError: false,
		},
		{
			name: "empty list",
			input: MethodResponse{
				"Email/get",
				map[string]any{
					"list": []any{},
				},
				"emails",
			},
			expected:    []Email{},
			expectError: false,
		},
		{
			name: "list with invalid item",
			input: MethodResponse{
				"Email/get",
				map[string]any{
					"list": []any{
						map[string]any{
							"id":         "email1",
							"subject":    "Valid Email",
							"receivedAt": "2025-01-15T10:00:00Z",
						},
						"invalid item",
						map[string]any{
							"id":         "email2",
							"subject":    "Another Valid Email",
							"receivedAt": "2025-01-15T11:00:00Z",
						},
					},
				},
				"emails",
			},
			expected: []Email{
				{
					ID: "email1", Subject: "Valid Email", ReceivedAt: "2025-01-15T10:00:00Z",
					From: []EmailAddress{}, To: []EmailAddress{}, CC: []EmailAddress{},
					BCC: []EmailAddress{}, ReplyTo: []EmailAddress{},
					TextBody: []BodyPart{}, HTMLBody: []BodyPart{}, Attachments: []Attachment{},
					MessageID: []string{}, InReplyTo: []string{}, References: []string{},
				},
				{
					ID: "email2", Subject: "Another Valid Email", ReceivedAt: "2025-01-15T11:00:00Z",
					From: []EmailAddress{}, To: []EmailAddress{}, CC: []EmailAddress{},
					BCC: []EmailAddress{}, ReplyTo: []EmailAddress{},
					TextBody: []BodyPart{}, HTMLBody: []BodyPart{}, Attachments: []Attachment{},
					MessageID: []string{}, InReplyTo: []string{}, References: []string{},
				},
			},
			expectError: false,
		},
		{
			name: "invalid response format - not a map",
			input: MethodResponse{
				"Email/get",
				"not a map",
				"emails",
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "invalid response format - missing list",
			input: MethodResponse{
				"Email/get",
				map[string]any{
					"notList": []any{},
				},
				"emails",
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "invalid response format - list is not array",
			input: MethodResponse{
				"Email/get",
				map[string]any{
					"list": "not an array",
				},
				"emails",
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "response with metadata",
			input: MethodResponse{
				"Email/get",
				map[string]any{
					"accountId": "acc123",
					"state":     "state456",
					"list": []any{
						map[string]any{
							"id":         "email1",
							"subject":    "Email with metadata",
							"receivedAt": "2025-01-15T10:00:00Z",
						},
					},
				},
				"emails",
			},
			expected: []Email{
				{
					ID: "email1", Subject: "Email with metadata", ReceivedAt: "2025-01-15T10:00:00Z",
					From: []EmailAddress{}, To: []EmailAddress{}, CC: []EmailAddress{},
					BCC: []EmailAddress{}, ReplyTo: []EmailAddress{},
					TextBody: []BodyPart{}, HTMLBody: []BodyPart{}, Attachments: []Attachment{},
					MessageID: []string{}, InReplyTo: []string{}, References: []string{},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEmailList(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("parseEmailList() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("parseEmailList() unexpected error: %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseEmailList() mismatch\ngot:  %+v\nwant: %+v", got, tt.expected)
			}
		})
	}
}

func TestSendEmailOptsWithAttachments(t *testing.T) {
	opts := SendEmailOpts{
		To:       []string{"recipient@example.com"},
		Subject:  "Test with attachments",
		TextBody: "See attached files",
		Attachments: []AttachmentOpts{
			{BlobID: "blob-123", Name: "document.pdf", Type: "application/pdf"},
			{BlobID: "blob-456", Name: "image.png", Type: "image/png"},
		},
	}

	if len(opts.Attachments) != 2 {
		t.Errorf("expected 2 attachments, got %d", len(opts.Attachments))
	}
	if opts.Attachments[0].BlobID != "blob-123" {
		t.Errorf("expected blob-123, got %s", opts.Attachments[0].BlobID)
	}
}

func TestParseSearchSnippets(t *testing.T) {
	tests := []struct {
		name        string
		response    MethodResponse
		want        []SearchSnippet
		expectError bool
	}{
		{
			name: "valid snippets with highlights",
			response: MethodResponse{
				"SearchSnippet/get",
				map[string]any{
					"list": []any{
						map[string]any{
							"emailId": "email1",
							"subject": "<em>Invoice</em> for December",
							"preview": "Please find your <em>invoice</em> attached...",
						},
						map[string]any{
							"emailId": "email2",
							"subject": "Meeting notes",
							"preview": "Discussed the <em>invoice</em> processing workflow",
						},
					},
				},
				"snippets",
			},
			want: []SearchSnippet{
				{
					EmailID: "email1",
					Subject: "<em>Invoice</em> for December",
					Preview: "Please find your <em>invoice</em> attached...",
				},
				{
					EmailID: "email2",
					Subject: "Meeting notes",
					Preview: "Discussed the <em>invoice</em> processing workflow",
				},
			},
			expectError: false,
		},
		{
			name: "empty list",
			response: MethodResponse{
				"SearchSnippet/get",
				map[string]any{
					"list": []any{},
				},
				"snippets",
			},
			want:        []SearchSnippet{},
			expectError: false,
		},
		{
			name: "snippet with only emailId",
			response: MethodResponse{
				"SearchSnippet/get",
				map[string]any{
					"list": []any{
						map[string]any{
							"emailId": "email1",
						},
					},
				},
				"snippets",
			},
			want: []SearchSnippet{
				{
					EmailID: "email1",
					Subject: "",
					Preview: "",
				},
			},
			expectError: false,
		},
		{
			name: "snippet with subject only",
			response: MethodResponse{
				"SearchSnippet/get",
				map[string]any{
					"list": []any{
						map[string]any{
							"emailId": "email1",
							"subject": "Highlighted <em>subject</em>",
						},
					},
				},
				"snippets",
			},
			want: []SearchSnippet{
				{
					EmailID: "email1",
					Subject: "Highlighted <em>subject</em>",
					Preview: "",
				},
			},
			expectError: false,
		},
		{
			name: "snippet with preview only",
			response: MethodResponse{
				"SearchSnippet/get",
				map[string]any{
					"list": []any{
						map[string]any{
							"emailId": "email1",
							"preview": "Body <em>match</em> here",
						},
					},
				},
				"snippets",
			},
			want: []SearchSnippet{
				{
					EmailID: "email1",
					Subject: "",
					Preview: "Body <em>match</em> here",
				},
			},
			expectError: false,
		},
		{
			name: "list with invalid item - not a map",
			response: MethodResponse{
				"SearchSnippet/get",
				map[string]any{
					"list": []any{
						map[string]any{
							"emailId": "email1",
							"subject": "Valid snippet",
						},
						"invalid string item",
						map[string]any{
							"emailId": "email2",
							"subject": "Another valid snippet",
						},
					},
				},
				"snippets",
			},
			want: []SearchSnippet{
				{
					EmailID: "email1",
					Subject: "Valid snippet",
					Preview: "",
				},
				{
					EmailID: "email2",
					Subject: "Another valid snippet",
					Preview: "",
				},
			},
			expectError: false,
		},
		{
			name: "invalid response format - not a map",
			response: MethodResponse{
				"SearchSnippet/get",
				"invalid string response",
				"snippets",
			},
			want:        nil,
			expectError: true,
		},
		{
			name: "invalid response format - missing list",
			response: MethodResponse{
				"SearchSnippet/get",
				map[string]any{
					"accountId": "acc123",
					"state":     "state456",
				},
				"snippets",
			},
			want:        nil,
			expectError: true,
		},
		{
			name: "invalid response format - list is not array",
			response: MethodResponse{
				"SearchSnippet/get",
				map[string]any{
					"list": "not an array",
				},
				"snippets",
			},
			want:        nil,
			expectError: true,
		},
		{
			name: "invalid response format - list is number",
			response: MethodResponse{
				"SearchSnippet/get",
				map[string]any{
					"list": 123,
				},
				"snippets",
			},
			want:        nil,
			expectError: true,
		},
		{
			name: "snippet with wrong field types",
			response: MethodResponse{
				"SearchSnippet/get",
				map[string]any{
					"list": []any{
						map[string]any{
							"emailId": 12345,
							"subject": true,
							"preview": []string{"wrong", "type"},
						},
					},
				},
				"snippets",
			},
			want: []SearchSnippet{
				{
					EmailID: "",
					Subject: "",
					Preview: "",
				},
			},
			expectError: false,
		},
		{
			name: "multiple snippets mixed quality",
			response: MethodResponse{
				"SearchSnippet/get",
				map[string]any{
					"list": []any{
						map[string]any{
							"emailId": "email1",
							"subject": "Complete <em>snippet</em>",
							"preview": "Full preview with <em>highlight</em>",
						},
						map[string]any{
							"emailId": "email2",
							"subject": "Subject only",
						},
						map[string]any{
							"emailId": "email3",
							"preview": "Preview only",
						},
						map[string]any{
							"emailId": "email4",
						},
					},
				},
				"snippets",
			},
			want: []SearchSnippet{
				{
					EmailID: "email1",
					Subject: "Complete <em>snippet</em>",
					Preview: "Full preview with <em>highlight</em>",
				},
				{
					EmailID: "email2",
					Subject: "Subject only",
					Preview: "",
				},
				{
					EmailID: "email3",
					Subject: "",
					Preview: "Preview only",
				},
				{
					EmailID: "email4",
					Subject: "",
					Preview: "",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSearchSnippets(tt.response)
			if tt.expectError {
				if err == nil {
					t.Errorf("parseSearchSnippets() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("parseSearchSnippets() unexpected error: %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseSearchSnippets() mismatch\ngot:  %+v\nwant: %+v", got, tt.want)
			}
		})
	}
}

func Test_formatAddressList(t *testing.T) {
	tests := []struct {
		name     string
		addrs    []EmailAddress
		expected string
	}{
		{
			name: "single address with name",
			addrs: []EmailAddress{
				{Name: "John Doe", Email: "john@example.com"},
			},
			expected: "John Doe <john@example.com>",
		},
		{
			name: "single address without name",
			addrs: []EmailAddress{
				{Name: "", Email: "jane@example.com"},
			},
			expected: "jane@example.com",
		},
		{
			name: "multiple addresses",
			addrs: []EmailAddress{
				{Name: "Alice Smith", Email: "alice@example.com"},
				{Name: "", Email: "bob@example.com"},
				{Name: "Charlie", Email: "charlie@example.com"},
			},
			expected: "Alice Smith <alice@example.com>, bob@example.com, Charlie <charlie@example.com>",
		},
		{
			name:     "empty list",
			addrs:    []EmailAddress{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAddressList(tt.addrs)
			if got != tt.expected {
				t.Errorf("formatAddressList() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func Test_buildForwardBody(t *testing.T) {
	tests := []struct {
		name             string
		original         *Email
		prependBody      string
		wantTextContains []string
		wantHTMLContains []string
		wantHTMLEmpty    bool
	}{
		{
			name: "basic forward without prepended body",
			original: &Email{
				Subject:    "Test Subject",
				ReceivedAt: "2025-01-15T10:30:00Z",
				From:       []EmailAddress{{Name: "Sender", Email: "sender@example.com"}},
				To:         []EmailAddress{{Name: "Recipient", Email: "recipient@example.com"}},
				TextBody:   []BodyPart{{PartID: "text", Type: "text/plain"}},
				BodyValues: map[string]BodyValue{"text": {Value: "Original message content"}},
			},
			prependBody: "",
			wantTextContains: []string{
				"---------- Forwarded message ---------",
				"From: Sender <sender@example.com>",
				"Date: Wed, 15 Jan 2025 10:30:00 +0000",
				"Subject: Test Subject",
				"To: Recipient <recipient@example.com>",
				"Original message content",
			},
			wantHTMLEmpty: true,
		},
		{
			name: "forward with prepended body",
			original: &Email{
				Subject:    "Test Subject",
				ReceivedAt: "2025-01-15T10:30:00Z",
				From:       []EmailAddress{{Email: "sender@example.com"}},
				To:         []EmailAddress{{Email: "recipient@example.com"}},
				TextBody:   []BodyPart{{PartID: "text", Type: "text/plain"}},
				BodyValues: map[string]BodyValue{"text": {Value: "Original content"}},
			},
			prependBody: "FYI - see below",
			wantTextContains: []string{
				"FYI - see below",
				"---------- Forwarded message ---------",
				"From: sender@example.com",
				"Original content",
			},
			wantHTMLEmpty: true,
		},
		{
			name: "forward with prepended body HTML escaping",
			original: &Email{
				Subject:    "Test Subject",
				ReceivedAt: "2025-01-15T10:30:00Z",
				From:       []EmailAddress{{Email: "sender@example.com"}},
				To:         []EmailAddress{{Email: "recipient@example.com"}},
				TextBody:   []BodyPart{{PartID: "text", Type: "text/plain"}},
				HTMLBody:   []BodyPart{{PartID: "html", Type: "text/html"}},
				BodyValues: map[string]BodyValue{
					"text": {Value: "Original content"},
					"html": {Value: "<p>Original HTML</p>"},
				},
			},
			prependBody: "<script>alert('xss')</script>",
			wantTextContains: []string{
				"<script>alert('xss')</script>",
				"---------- Forwarded message ---------",
			},
			wantHTMLContains: []string{
				"&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
				"<p>Original HTML</p>",
			},
		},
		{
			name: "forward with HTML original",
			original: &Email{
				Subject:    "HTML Email",
				ReceivedAt: "2025-01-15T10:30:00Z",
				From:       []EmailAddress{{Name: "Sender", Email: "sender@example.com"}},
				To:         []EmailAddress{{Email: "recipient@example.com"}},
				TextBody:   []BodyPart{{PartID: "text", Type: "text/plain"}},
				HTMLBody:   []BodyPart{{PartID: "html", Type: "text/html"}},
				BodyValues: map[string]BodyValue{
					"text": {Value: "Plain text version"},
					"html": {Value: "<p>HTML content here</p>"},
				},
			},
			prependBody: "",
			wantTextContains: []string{
				"---------- Forwarded message ---------",
				"Plain text version",
			},
			wantHTMLContains: []string{
				"border-left: 2px solid #ccc",
				"<p>HTML content here</p>",
				"From: Sender <sender@example.com>",
			},
		},
		{
			name: "forward with CC recipients",
			original: &Email{
				Subject:    "CC Test",
				ReceivedAt: "2025-01-15T10:30:00Z",
				From:       []EmailAddress{{Email: "sender@example.com"}},
				To:         []EmailAddress{{Email: "to@example.com"}},
				CC:         []EmailAddress{{Name: "CC Person", Email: "cc@example.com"}, {Email: "cc2@example.com"}},
				TextBody:   []BodyPart{{PartID: "text", Type: "text/plain"}},
				BodyValues: map[string]BodyValue{"text": {Value: "Message"}},
			},
			prependBody: "",
			wantTextContains: []string{
				"Cc: CC Person <cc@example.com>, cc2@example.com",
			},
			wantHTMLEmpty: true,
		},
		{
			name: "forward with empty body values",
			original: &Email{
				Subject:    "Empty Body",
				ReceivedAt: "2025-01-15T10:30:00Z",
				From:       []EmailAddress{{Email: "sender@example.com"}},
				To:         []EmailAddress{{Email: "recipient@example.com"}},
				TextBody:   []BodyPart{},
				BodyValues: map[string]BodyValue{},
			},
			prependBody: "",
			wantTextContains: []string{
				"---------- Forwarded message ---------",
				"From: sender@example.com",
			},
			wantHTMLEmpty: true,
		},
		{
			name: "forward with invalid date format falls back to raw value",
			original: &Email{
				Subject:    "Invalid Date Test",
				ReceivedAt: "not-a-valid-date",
				From:       []EmailAddress{{Email: "sender@example.com"}},
				To:         []EmailAddress{{Email: "recipient@example.com"}},
				TextBody:   []BodyPart{{PartID: "text", Type: "text/plain"}},
				BodyValues: map[string]BodyValue{"text": {Value: "Message content"}},
			},
			prependBody: "",
			wantTextContains: []string{
				"Date: not-a-valid-date",
			},
			wantHTMLEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, gotHTML := buildForwardBody(tt.original, tt.prependBody)

			// Check text body contains expected strings
			for _, want := range tt.wantTextContains {
				if !strings.Contains(gotText, want) {
					t.Errorf("buildForwardBody() textBody missing %q\ngot: %s", want, gotText)
				}
			}

			// Check HTML body
			if tt.wantHTMLEmpty {
				if gotHTML != "" {
					t.Errorf("buildForwardBody() expected empty htmlBody, got: %s", gotHTML)
				}
			} else {
				for _, want := range tt.wantHTMLContains {
					if !strings.Contains(gotHTML, want) {
						t.Errorf("buildForwardBody() htmlBody missing %q\ngot: %s", want, gotHTML)
					}
				}
			}
		})
	}
}

func TestImportEmail(t *testing.T) {
	tests := []struct {
		name     string
		opts     ImportEmailOpts
		response string
		wantID   string
		wantErr  bool
	}{
		{
			name: "successful import",
			opts: ImportEmailOpts{
				BlobID:     "blob123",
				MailboxIDs: map[string]bool{"inbox": true},
			},
			response: `{
				"methodResponses": [
					["Email/import", {
						"accountId": "acc123",
						"created": {
							"import1": {
								"id": "email456"
							}
						}
					}, "importEmail"]
				]
			}`,
			wantID:  "email456",
			wantErr: false,
		},
		{
			name: "missing blobId",
			opts: ImportEmailOpts{
				MailboxIDs: map[string]bool{"inbox": true},
			},
			wantErr: true,
		},
		{
			name: "missing mailboxIds",
			opts: ImportEmailOpts{
				BlobID: "blob123",
			},
			wantErr: true,
		},
		{
			name: "empty mailboxIds",
			opts: ImportEmailOpts{
				BlobID:     "blob123",
				MailboxIDs: map[string]bool{},
			},
			wantErr: true,
		},
		{
			name: "import with keywords",
			opts: ImportEmailOpts{
				BlobID:     "blob123",
				MailboxIDs: map[string]bool{"inbox": true},
				Keywords:   map[string]bool{"$seen": true},
			},
			response: `{
				"methodResponses": [
					["Email/import", {
						"accountId": "acc123",
						"created": {
							"import1": {
								"id": "email789"
							}
						}
					}, "importEmail"]
				]
			}`,
			wantID:  "email789",
			wantErr: false,
		},
		{
			name: "import with receivedAt",
			opts: ImportEmailOpts{
				BlobID:     "blob123",
				MailboxIDs: map[string]bool{"inbox": true},
				ReceivedAt: "2024-01-15T10:30:00Z",
			},
			response: `{
				"methodResponses": [
					["Email/import", {
						"accountId": "acc123",
						"created": {
							"import1": {
								"id": "email999"
							}
						}
					}, "importEmail"]
				]
			}`,
			wantID:  "email999",
			wantErr: false,
		},
		{
			name: "import with all optional fields",
			opts: ImportEmailOpts{
				BlobID:     "blob123",
				MailboxIDs: map[string]bool{"inbox": true, "archive": true},
				Keywords:   map[string]bool{"$seen": true, "$flagged": true},
				ReceivedAt: "2024-01-15T10:30:00Z",
			},
			response: `{
				"methodResponses": [
					["Email/import", {
						"accountId": "acc123",
						"created": {
							"import1": {
								"id": "email111"
							}
						}
					}, "importEmail"]
				]
			}`,
			wantID:  "email111",
			wantErr: false,
		},
		{
			name: "server error - not created",
			opts: ImportEmailOpts{
				BlobID:     "blob123",
				MailboxIDs: map[string]bool{"inbox": true},
			},
			response: `{
				"methodResponses": [
					["Email/import", {
						"accountId": "acc123",
						"notCreated": {
							"import1": {
								"type": "invalidEmail",
								"description": "Email is invalid"
							}
						}
					}, "importEmail"]
				]
			}`,
			wantErr: true,
		},
		{
			name: "server error - blob not found",
			opts: ImportEmailOpts{
				BlobID:     "invalid-blob",
				MailboxIDs: map[string]bool{"inbox": true},
			},
			response: `{
				"methodResponses": [
					["Email/import", {
						"accountId": "acc123",
						"notCreated": {
							"import1": {
								"type": "blobNotFound",
								"description": "Blob not found"
							}
						}
					}, "importEmail"]
				]
			}`,
			wantErr: true,
		},
		{
			name: "server error - invalid mailbox",
			opts: ImportEmailOpts{
				BlobID:     "blob123",
				MailboxIDs: map[string]bool{"nonexistent": true},
			},
			response: `{
				"methodResponses": [
					["Email/import", {
						"accountId": "acc123",
						"notCreated": {
							"import1": {
								"type": "invalidMailboxes",
								"description": "Mailbox not found"
							}
						}
					}, "importEmail"]
				]
			}`,
			wantErr: true,
		},
		{
			name: "invalid response format - not a map",
			opts: ImportEmailOpts{
				BlobID:     "blob123",
				MailboxIDs: map[string]bool{"inbox": true},
			},
			response: `{
				"methodResponses": [
					["Email/import", "not a map", "importEmail"]
				]
			}`,
			wantErr: true,
		},
		{
			name: "invalid response format - missing created",
			opts: ImportEmailOpts{
				BlobID:     "blob123",
				MailboxIDs: map[string]bool{"inbox": true},
			},
			response: `{
				"methodResponses": [
					["Email/import", {
						"accountId": "acc123"
					}, "importEmail"]
				]
			}`,
			wantErr: true,
		},
		{
			name: "invalid response format - created missing import1",
			opts: ImportEmailOpts{
				BlobID:     "blob123",
				MailboxIDs: map[string]bool{"inbox": true},
			},
			response: `{
				"methodResponses": [
					["Email/import", {
						"accountId": "acc123",
						"created": {}
					}, "importEmail"]
				]
			}`,
			wantErr: true,
		},
		{
			name: "invalid response format - import1 missing id",
			opts: ImportEmailOpts{
				BlobID:     "blob123",
				MailboxIDs: map[string]bool{"inbox": true},
			},
			response: `{
				"methodResponses": [
					["Email/import", {
						"accountId": "acc123",
						"created": {
							"import1": {}
						}
					}, "importEmail"]
				]
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip creating server for validation errors
			if tt.wantErr && tt.response == "" {
				client := NewClientWithBaseURL("test-token", "http://dummy")
				_, err := client.ImportEmail(context.Background(), tt.opts)
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			defer apiServer.Close()

			sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{
					"apiUrl": "` + apiServer.URL + `",
					"uploadUrl": "` + apiServer.URL + `/{accountId}/",
					"downloadUrl": "` + apiServer.URL + `",
					"accounts": {"acc123": {}},
					"primaryAccounts": {"urn:ietf:params:jmap:mail": "acc123"}
				}`))
			}))
			defer sessionServer.Close()

			client := NewClientWithBaseURL("test-token", sessionServer.URL)

			got, err := client.ImportEmail(context.Background(), tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.wantID {
				t.Errorf("got email ID %s, want %s", got, tt.wantID)
			}
		})
	}
}

func TestEmailSearchFilter_ToJMAPFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter *EmailSearchFilter
		want   map[string]any
	}{
		{
			name:   "empty filter",
			filter: &EmailSearchFilter{},
			want:   map[string]any{},
		},
		{
			name:   "text only",
			filter: &EmailSearchFilter{Text: "hello world"},
			want:   map[string]any{"text": "hello world"},
		},
		{
			name:   "after only",
			filter: &EmailSearchFilter{After: "2026-01-15T00:00:00Z"},
			want:   map[string]any{"after": "2026-01-15T00:00:00Z"},
		},
		{
			name:   "before only",
			filter: &EmailSearchFilter{Before: "2026-01-31T00:00:00Z"},
			want:   map[string]any{"before": "2026-01-31T00:00:00Z"},
		},
		{
			name: "all fields",
			filter: &EmailSearchFilter{
				Text:   "subject:meeting",
				After:  "2026-01-15T00:00:00Z",
				Before: "2026-01-31T00:00:00Z",
			},
			want: map[string]any{
				"text":   "subject:meeting",
				"after":  "2026-01-15T00:00:00Z",
				"before": "2026-01-31T00:00:00Z",
			},
		},
		{
			name: "after and before without text",
			filter: &EmailSearchFilter{
				After:  "2026-01-15T00:00:00Z",
				Before: "2026-01-31T00:00:00Z",
			},
			want: map[string]any{
				"after":  "2026-01-15T00:00:00Z",
				"before": "2026-01-31T00:00:00Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.ToJMAPFilter()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToJMAPFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSendDraft(t *testing.T) {
	tests := []struct {
		name                       string
		draftID                    string
		apiResponses               []string // Multiple API responses for different calls
		wantID                     string
		wantErr                    bool
		wantErrContain             string
		verifyOnSuccessUpdateEmail bool // If true, verify onSuccessUpdateEmail is in the request
		expectedSentMailboxID      string
	}{
		{
			name:    "successful send with onSuccessUpdateEmail",
			draftID: "draft123",
			apiResponses: []string{
				// First call: Email/get to verify it's a draft
				`{
					"methodResponses": [
						["Email/get", {
							"accountId": "acc123",
							"list": [{
								"id": "draft123",
								"subject": "Test Draft",
								"keywords": {"$draft": true}
							}]
						}, "getEmail"]
					]
				}`,
				// Second call: Identity/get for default identity
				`{
					"methodResponses": [
						["Identity/get", {
							"accountId": "acc123",
							"list": [{"id": "identity1", "email": "test@example.com", "mayDelete": true}]
						}, "getIdentities"]
					]
				}`,
				// Third call: Mailbox/get for sent mailbox
				`{
					"methodResponses": [
						["Mailbox/get", {
							"accountId": "acc123",
							"list": [
								{"id": "inbox1", "name": "Inbox", "role": "inbox"},
								{"id": "sent1", "name": "Sent", "role": "sent"},
								{"id": "drafts1", "name": "Drafts", "role": "drafts"}
							]
						}, "getMailboxes"]
					]
				}`,
				// Fourth call: EmailSubmission/set (with onSuccessUpdateEmail)
				`{
					"methodResponses": [
						["EmailSubmission/set", {
							"accountId": "acc123",
							"created": {
								"send": {"id": "submission123"}
							}
						}, "send"]
					]
				}`,
			},
			wantID:                     "submission123",
			wantErr:                    false,
			verifyOnSuccessUpdateEmail: true,
			expectedSentMailboxID:      "sent1",
		},
		{
			name:    "submission fails - email should stay unchanged",
			draftID: "draft456",
			apiResponses: []string{
				// First call: Email/get to verify it's a draft
				`{
					"methodResponses": [
						["Email/get", {
							"accountId": "acc123",
							"list": [{
								"id": "draft456",
								"subject": "Test Draft",
								"keywords": {"$draft": true}
							}]
						}, "getEmail"]
					]
				}`,
				// Second call: Identity/get for default identity
				`{
					"methodResponses": [
						["Identity/get", {
							"accountId": "acc123",
							"list": [{"id": "identity1", "email": "test@example.com", "mayDelete": true}]
						}, "getIdentities"]
					]
				}`,
				// Third call: Mailbox/get for sent mailbox
				`{
					"methodResponses": [
						["Mailbox/get", {
							"accountId": "acc123",
							"list": [
								{"id": "inbox1", "name": "Inbox", "role": "inbox"},
								{"id": "sent1", "name": "Sent", "role": "sent"}
							]
						}, "getMailboxes"]
					]
				}`,
				// Fourth call: EmailSubmission/set fails with "forbidden from" error
				`{
					"methodResponses": [
						["EmailSubmission/set", {
							"accountId": "acc123",
							"notCreated": {
								"send": {
									"type": "forbiddenFrom",
									"description": "The from address is not allowed for this identity"
								}
							}
						}, "send"]
					]
				}`,
			},
			wantErr:        true,
			wantErrContain: "failed to send draft",
		},
		{
			name:    "email is not a draft",
			draftID: "notadraft123",
			apiResponses: []string{
				// First call: Email/get returns email without $draft keyword
				`{
					"methodResponses": [
						["Email/get", {
							"accountId": "acc123",
							"list": [{
								"id": "notadraft123",
								"subject": "Already Sent",
								"keywords": {"$seen": true}
							}]
						}, "getEmail"]
					]
				}`,
			},
			wantErr:        true,
			wantErrContain: "is not a draft",
		},
		{
			name:    "no sent mailbox found",
			draftID: "draft789",
			apiResponses: []string{
				// First call: Email/get to verify it's a draft
				`{
					"methodResponses": [
						["Email/get", {
							"accountId": "acc123",
							"list": [{
								"id": "draft789",
								"subject": "Test Draft",
								"keywords": {"$draft": true}
							}]
						}, "getEmail"]
					]
				}`,
				// Second call: Identity/get for default identity
				`{
					"methodResponses": [
						["Identity/get", {
							"accountId": "acc123",
							"list": [{"id": "identity1", "email": "test@example.com", "mayDelete": true}]
						}, "getIdentities"]
					]
				}`,
				// Third call: Mailbox/get - no sent mailbox
				`{
					"methodResponses": [
						["Mailbox/get", {
							"accountId": "acc123",
							"list": [
								{"id": "inbox1", "name": "Inbox", "role": "inbox"},
								{"id": "drafts1", "name": "Drafts", "role": "drafts"}
							]
						}, "getMailboxes"]
					]
				}`,
			},
			wantErr:        true,
			wantErrContain: "sent mailbox",
		},
		{
			name:    "submission fails with invalid recipients",
			draftID: "draft-bad-recipients",
			apiResponses: []string{
				// First call: Email/get to verify it's a draft
				`{
					"methodResponses": [
						["Email/get", {
							"accountId": "acc123",
							"list": [{
								"id": "draft-bad-recipients",
								"subject": "Test Draft",
								"keywords": {"$draft": true}
							}]
						}, "getEmail"]
					]
				}`,
				// Second call: Identity/get for default identity
				`{
					"methodResponses": [
						["Identity/get", {
							"accountId": "acc123",
							"list": [{"id": "identity1", "email": "test@example.com", "mayDelete": true}]
						}, "getIdentities"]
					]
				}`,
				// Third call: Mailbox/get for sent mailbox
				`{
					"methodResponses": [
						["Mailbox/get", {
							"accountId": "acc123",
							"list": [
								{"id": "sent1", "name": "Sent", "role": "sent"}
							]
						}, "getMailboxes"]
					]
				}`,
				// Fourth call: EmailSubmission/set fails with invalid recipients
				`{
					"methodResponses": [
						["EmailSubmission/set", {
							"accountId": "acc123",
							"notCreated": {
								"send": {
									"type": "invalidRecipients",
									"description": "No valid recipients found"
								}
							}
						}, "send"]
					]
				}`,
			},
			wantErr:        true,
			wantErrContain: "failed to send draft",
		},
		{
			name:    "unexpected response format - no created or notCreated",
			draftID: "draft-unexpected",
			apiResponses: []string{
				// First call: Email/get to verify it's a draft
				`{
					"methodResponses": [
						["Email/get", {
							"accountId": "acc123",
							"list": [{
								"id": "draft-unexpected",
								"subject": "Test Draft",
								"keywords": {"$draft": true}
							}]
						}, "getEmail"]
					]
				}`,
				// Second call: Identity/get for default identity
				`{
					"methodResponses": [
						["Identity/get", {
							"accountId": "acc123",
							"list": [{"id": "identity1", "email": "test@example.com", "mayDelete": true}]
						}, "getIdentities"]
					]
				}`,
				// Third call: Mailbox/get for sent mailbox
				`{
					"methodResponses": [
						["Mailbox/get", {
							"accountId": "acc123",
							"list": [
								{"id": "sent1", "name": "Sent", "role": "sent"}
							]
						}, "getMailboxes"]
					]
				}`,
				// Fourth call: EmailSubmission/set - response missing both created and notCreated
				`{
					"methodResponses": [
						["EmailSubmission/set", {
							"accountId": "acc123"
						}, "send"]
					]
				}`,
			},
			wantErr:        true,
			wantErrContain: "unexpected submission response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callIndex := 0
			var capturedSubmissionRequest map[string]any

			apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				// Capture request body for the EmailSubmission/set call (4th call, index 3)
				if tt.verifyOnSuccessUpdateEmail && callIndex == 3 {
					body, _ := io.ReadAll(r.Body)
					_ = json.Unmarshal(body, &capturedSubmissionRequest)
				}

				if callIndex < len(tt.apiResponses) {
					_, _ = w.Write([]byte(tt.apiResponses[callIndex]))
					callIndex++
				} else {
					// Return empty response if we run out of expected responses
					_, _ = w.Write([]byte(`{"methodResponses": []}`))
				}
			}))
			defer apiServer.Close()

			sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{
					"apiUrl": "` + apiServer.URL + `",
					"uploadUrl": "` + apiServer.URL + `/{accountId}/",
					"downloadUrl": "` + apiServer.URL + `",
					"accounts": {"acc123": {}},
					"primaryAccounts": {"urn:ietf:params:jmap:mail": "acc123"}
				}`))
			}))
			defer sessionServer.Close()

			client := NewClientWithBaseURL("test-token", sessionServer.URL)

			got, err := client.SendDraft(context.Background(), tt.draftID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.wantErrContain != "" && !strings.Contains(err.Error(), tt.wantErrContain) {
					t.Errorf("expected error containing %q, got %q", tt.wantErrContain, err.Error())
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.wantID {
				t.Errorf("got submission ID %s, want %s", got, tt.wantID)
			}

			// Verify onSuccessUpdateEmail was included in the request
			if tt.verifyOnSuccessUpdateEmail {
				methodCalls, ok := capturedSubmissionRequest["methodCalls"].([]any)
				if !ok || len(methodCalls) == 0 {
					t.Errorf("expected methodCalls in request, got %v", capturedSubmissionRequest)
					return
				}
				firstCall, ok := methodCalls[0].([]any)
				if !ok || len(firstCall) < 2 {
					t.Errorf("expected method call array, got %v", methodCalls[0])
					return
				}
				if firstCall[0] != "EmailSubmission/set" {
					t.Errorf("expected EmailSubmission/set, got %v", firstCall[0])
					return
				}
				args, ok := firstCall[1].(map[string]any)
				if !ok {
					t.Errorf("expected method args map, got %v", firstCall[1])
					return
				}

				// Verify onSuccessUpdateEmail exists with #send key
				onSuccess, ok := args["onSuccessUpdateEmail"].(map[string]any)
				if !ok {
					t.Errorf("onSuccessUpdateEmail missing or not a map: %v", args["onSuccessUpdateEmail"])
					return
				}
				sendUpdate, ok := onSuccess["#send"].(map[string]any)
				if !ok {
					t.Errorf("onSuccessUpdateEmail missing #send key: %v", onSuccess)
					return
				}

				// Verify keywords/$draft is set to nil (null in JSON)
				if _, exists := sendUpdate["keywords/$draft"]; !exists {
					t.Errorf("onSuccessUpdateEmail missing keywords/$draft: %v", sendUpdate)
				}

				// Verify mailboxIds contains the sent mailbox
				mailboxIds, ok := sendUpdate["mailboxIds"].(map[string]any)
				if !ok {
					t.Errorf("onSuccessUpdateEmail missing mailboxIds: %v", sendUpdate)
					return
				}
				if _, exists := mailboxIds[tt.expectedSentMailboxID]; !exists {
					t.Errorf("onSuccessUpdateEmail mailboxIds missing sent mailbox %s: %v", tt.expectedSentMailboxID, mailboxIds)
				}
			}
		})
	}
}
