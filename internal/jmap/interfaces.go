package jmap

import (
	"context"
	"io"
	"time"
)

// EmailService defines the interface for email operations in the JMAP client.
// This interface enables unit testing without network calls by allowing mock implementations.
type EmailService interface {
	// GetEmails retrieves emails from a mailbox with optional filtering and limit
	GetEmails(ctx context.Context, mailboxID string, limit int) ([]Email, error)

	// SearchEmails searches for emails matching a filter
	SearchEmails(ctx context.Context, filter *EmailSearchFilter, limit int) ([]Email, error)

	// GetDrafts retrieves draft emails
	GetDrafts(ctx context.Context, limit int) ([]Email, error)

	// GetEmailByID retrieves a specific email by ID with full details
	GetEmailByID(ctx context.Context, id string) (*Email, error)

	// UpdateDraft updates an existing draft
	UpdateDraft(ctx context.Context, draftID string, opts SendEmailOpts) error

	// SendDraft sends an existing draft
	SendDraft(ctx context.Context, draftID string) (string, error)

	// SendEmail sends an email with the provided options
	SendEmail(ctx context.Context, opts SendEmailOpts) (string, error)

	// DeleteEmail moves an email to trash
	DeleteEmail(ctx context.Context, id string) error

	// MoveEmail moves an email to a target mailbox
	MoveEmail(ctx context.Context, id, targetMailboxID string) error

	// MarkEmailRead marks an email as read or unread
	MarkEmailRead(ctx context.Context, id string, read bool) error

	// GetThread retrieves all emails in a thread
	GetThread(ctx context.Context, threadID string) ([]Email, error)

	// GetEmailAttachments retrieves attachments for an email
	GetEmailAttachments(ctx context.Context, id string) ([]Attachment, error)

	// GetMailboxes retrieves all mailboxes for the account
	GetMailboxes(ctx context.Context) ([]Mailbox, error)

	// DownloadBlob downloads a blob (attachment) by ID and returns a ReadCloser
	DownloadBlob(ctx context.Context, blobID string) (io.ReadCloser, error)

	// UploadBlob uploads binary data and returns the blob ID
	UploadBlob(ctx context.Context, reader io.Reader, contentType string) (*UploadBlobResult, error)

	// GetIdentities retrieves sending identities for the account
	GetIdentities(ctx context.Context) ([]Identity, error)

	// GetMailboxByName finds a mailbox by name (case-insensitive)
	GetMailboxByName(ctx context.Context, name string) (*Mailbox, error)

	// ResolveMailboxID takes either a mailbox ID or name and returns the ID
	ResolveMailboxID(ctx context.Context, idOrName string) (string, error)

	// CreateMailbox creates a new mailbox (folder)
	CreateMailbox(ctx context.Context, opts CreateMailboxOpts) (*Mailbox, error)

	// DeleteMailbox deletes a mailbox by ID
	DeleteMailbox(ctx context.Context, id string) error

	// RenameMailbox renames a mailbox
	RenameMailbox(ctx context.Context, id, newName string) error

	// SearchEmailsWithSnippets searches with highlighted context
	SearchEmailsWithSnippets(ctx context.Context, filter *EmailSearchFilter, limit int) ([]Email, []SearchSnippet, error)

	// ImportEmail imports a raw RFC 5322 message
	ImportEmail(ctx context.Context, opts ImportEmailOpts) (string, error)

	// SaveDraft saves a new draft email
	SaveDraft(ctx context.Context, opts SendEmailOpts) (string, error)

	// CreateReplyDraft creates a draft as a reply to an existing email
	CreateReplyDraft(ctx context.Context, replyToID string, opts SendEmailOpts) (string, error)
}

// MaskedEmailService defines the interface for masked email (alias) operations.
// This interface enables unit testing without network calls by allowing mock implementations.
type MaskedEmailService interface {
	// GetMaskedEmails retrieves all masked email aliases
	GetMaskedEmails(ctx context.Context) ([]MaskedEmail, error)

	// GetMaskedEmailByEmail retrieves a specific masked email by its address
	GetMaskedEmailByEmail(ctx context.Context, email string) (*MaskedEmail, error)

	// GetMaskedEmailsForDomain retrieves masked emails for a specific domain
	GetMaskedEmailsForDomain(ctx context.Context, domain string) ([]MaskedEmail, error)

	// CreateMaskedEmail creates a new masked email alias
	CreateMaskedEmail(ctx context.Context, domain, description string) (*MaskedEmail, error)

	// UpdateMaskedEmailState updates the state of a masked email (enabled/disabled/deleted)
	UpdateMaskedEmailState(ctx context.Context, id string, state MaskedEmailState) error

	// UpdateMaskedEmailDescription updates the description of a masked email
	UpdateMaskedEmailDescription(ctx context.Context, id, description string) error
}

// VacationService defines the interface for vacation response operations.
type VacationService interface {
	// GetVacationResponse retrieves the current vacation/auto-reply settings
	GetVacationResponse(ctx context.Context) (*VacationResponse, error)

	// SetVacationResponse updates the vacation/auto-reply settings
	SetVacationResponse(ctx context.Context, opts SetVacationResponseOpts) error

	// DisableVacationResponse turns off the vacation responder
	DisableVacationResponse(ctx context.Context) error
}

// ContactsService defines the interface for contacts operations.
type ContactsService interface {
	// GetContacts retrieves contacts from an address book with optional limit
	GetContacts(ctx context.Context, addressBookID string, limit int) ([]Contact, error)

	// GetContactByID retrieves a specific contact by ID
	GetContactByID(ctx context.Context, id string) (*Contact, error)

	// CreateContact creates a new contact
	CreateContact(ctx context.Context, contact *Contact) (*Contact, error)

	// UpdateContact updates an existing contact
	UpdateContact(ctx context.Context, id string, updates map[string]interface{}) (*Contact, error)

	// DeleteContact deletes a contact by ID
	DeleteContact(ctx context.Context, id string) error

	// SearchContacts searches for contacts matching a query string
	SearchContacts(ctx context.Context, query string, limit int) ([]Contact, error)

	// GetAddressBooks retrieves all address books for the account
	GetAddressBooks(ctx context.Context) ([]AddressBook, error)
}

// CalendarService defines the interface for calendar operations.
type CalendarService interface {
	// GetCalendars retrieves all calendars for the account
	GetCalendars(ctx context.Context) ([]Calendar, error)

	// GetEvents retrieves calendar events within a date range
	GetEvents(ctx context.Context, calendarID string, from, to time.Time, limit int) ([]CalendarEvent, error)

	// GetEventByID retrieves a specific calendar event by ID
	GetEventByID(ctx context.Context, id string) (*CalendarEvent, error)

	// CreateEvent creates a new calendar event
	CreateEvent(ctx context.Context, event *CalendarEvent) (*CalendarEvent, error)

	// UpdateEvent updates an existing calendar event
	UpdateEvent(ctx context.Context, id string, updates map[string]interface{}) (*CalendarEvent, error)

	// DeleteEvent deletes a calendar event by ID
	DeleteEvent(ctx context.Context, id string) error
}

// QuotaService defines the interface for quota operations.
type QuotaService interface {
	// GetQuotas retrieves all quotas for the account
	GetQuotas(ctx context.Context) ([]Quota, error)
}

// SieveService defines Sieve filter management operations.
type SieveService interface {
	GetSieveBlocks(ctx context.Context) (*SieveBlocks, error)
	SetSieveBlocks(ctx context.Context, opts SetSieveBlocksOpts) error
}
