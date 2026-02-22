package jmap

import (
	"context"
	"io"
	"time"
)

// MockEmailService implements EmailService for testing.
// Each method can be overridden by setting the corresponding Func field.
// If a Func is not set, the method returns nil/empty values.
type MockEmailService struct {
	GetEmailsFunc                func(ctx context.Context, mailboxID string, limit int) ([]Email, error)
	SearchEmailsFunc             func(ctx context.Context, filter *EmailSearchFilter, limit int) ([]Email, error)
	GetDraftsFunc                func(ctx context.Context, limit int) ([]Email, error)
	GetEmailByIDFunc             func(ctx context.Context, id string) (*Email, error)
	UpdateDraftFunc              func(ctx context.Context, draftID string, opts SendEmailOpts) error
	SendDraftFunc                func(ctx context.Context, draftID string) (string, error)
	SendEmailFunc                func(ctx context.Context, opts SendEmailOpts) (string, error)
	DeleteEmailFunc              func(ctx context.Context, id string) error
	MoveEmailFunc                func(ctx context.Context, id, targetMailboxID string) error
	MarkEmailReadFunc            func(ctx context.Context, id string, read bool) error
	GetThreadFunc                func(ctx context.Context, threadID string) ([]Email, error)
	GetEmailAttachmentsFunc      func(ctx context.Context, id string) ([]Attachment, error)
	GetMailboxesFunc             func(ctx context.Context) ([]Mailbox, error)
	DownloadBlobFunc             func(ctx context.Context, blobID string) (io.ReadCloser, error)
	UploadBlobFunc               func(ctx context.Context, reader io.Reader, contentType string) (*UploadBlobResult, error)
	GetIdentitiesFunc            func(ctx context.Context) ([]Identity, error)
	GetMailboxByNameFunc         func(ctx context.Context, name string) (*Mailbox, error)
	ResolveMailboxIDFunc         func(ctx context.Context, idOrName string) (string, error)
	CreateMailboxFunc            func(ctx context.Context, opts CreateMailboxOpts) (*Mailbox, error)
	DeleteMailboxFunc            func(ctx context.Context, id string) error
	RenameMailboxFunc            func(ctx context.Context, id, newName string) error
	SearchEmailsWithSnippetsFunc func(ctx context.Context, filter *EmailSearchFilter, limit int) ([]Email, []SearchSnippet, error)
	ImportEmailFunc              func(ctx context.Context, opts ImportEmailOpts) (string, error)
	SaveDraftFunc                func(ctx context.Context, opts SendEmailOpts) (string, error)
	CreateReplyDraftFunc         func(ctx context.Context, replyToID string, opts SendEmailOpts) (string, error)
}

func (m *MockEmailService) GetEmails(ctx context.Context, mailboxID string, limit int) ([]Email, error) {
	if m.GetEmailsFunc != nil {
		return m.GetEmailsFunc(ctx, mailboxID, limit)
	}
	return nil, nil
}

func (m *MockEmailService) SearchEmails(ctx context.Context, filter *EmailSearchFilter, limit int) ([]Email, error) {
	if m.SearchEmailsFunc != nil {
		return m.SearchEmailsFunc(ctx, filter, limit)
	}
	return nil, nil
}

func (m *MockEmailService) GetDrafts(ctx context.Context, limit int) ([]Email, error) {
	if m.GetDraftsFunc != nil {
		return m.GetDraftsFunc(ctx, limit)
	}
	return nil, nil
}

func (m *MockEmailService) GetEmailByID(ctx context.Context, id string) (*Email, error) {
	if m.GetEmailByIDFunc != nil {
		return m.GetEmailByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockEmailService) UpdateDraft(ctx context.Context, draftID string, opts SendEmailOpts) error {
	if m.UpdateDraftFunc != nil {
		return m.UpdateDraftFunc(ctx, draftID, opts)
	}
	return nil
}

func (m *MockEmailService) SendDraft(ctx context.Context, draftID string) (string, error) {
	if m.SendDraftFunc != nil {
		return m.SendDraftFunc(ctx, draftID)
	}
	return "", nil
}

func (m *MockEmailService) SendEmail(ctx context.Context, opts SendEmailOpts) (string, error) {
	if m.SendEmailFunc != nil {
		return m.SendEmailFunc(ctx, opts)
	}
	return "", nil
}

func (m *MockEmailService) DeleteEmail(ctx context.Context, id string) error {
	if m.DeleteEmailFunc != nil {
		return m.DeleteEmailFunc(ctx, id)
	}
	return nil
}

func (m *MockEmailService) MoveEmail(ctx context.Context, id, targetMailboxID string) error {
	if m.MoveEmailFunc != nil {
		return m.MoveEmailFunc(ctx, id, targetMailboxID)
	}
	return nil
}

func (m *MockEmailService) MarkEmailRead(ctx context.Context, id string, read bool) error {
	if m.MarkEmailReadFunc != nil {
		return m.MarkEmailReadFunc(ctx, id, read)
	}
	return nil
}

func (m *MockEmailService) GetThread(ctx context.Context, threadID string) ([]Email, error) {
	if m.GetThreadFunc != nil {
		return m.GetThreadFunc(ctx, threadID)
	}
	return nil, nil
}

func (m *MockEmailService) GetEmailAttachments(ctx context.Context, id string) ([]Attachment, error) {
	if m.GetEmailAttachmentsFunc != nil {
		return m.GetEmailAttachmentsFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockEmailService) GetMailboxes(ctx context.Context) ([]Mailbox, error) {
	if m.GetMailboxesFunc != nil {
		return m.GetMailboxesFunc(ctx)
	}
	return nil, nil
}

func (m *MockEmailService) DownloadBlob(ctx context.Context, blobID string) (io.ReadCloser, error) {
	if m.DownloadBlobFunc != nil {
		return m.DownloadBlobFunc(ctx, blobID)
	}
	return nil, nil
}

func (m *MockEmailService) UploadBlob(ctx context.Context, reader io.Reader, contentType string) (*UploadBlobResult, error) {
	if m.UploadBlobFunc != nil {
		return m.UploadBlobFunc(ctx, reader, contentType)
	}
	return nil, nil
}

func (m *MockEmailService) GetIdentities(ctx context.Context) ([]Identity, error) {
	if m.GetIdentitiesFunc != nil {
		return m.GetIdentitiesFunc(ctx)
	}
	return nil, nil
}

func (m *MockEmailService) GetMailboxByName(ctx context.Context, name string) (*Mailbox, error) {
	if m.GetMailboxByNameFunc != nil {
		return m.GetMailboxByNameFunc(ctx, name)
	}
	return nil, nil
}

func (m *MockEmailService) ResolveMailboxID(ctx context.Context, idOrName string) (string, error) {
	if m.ResolveMailboxIDFunc != nil {
		return m.ResolveMailboxIDFunc(ctx, idOrName)
	}
	return "", nil
}

func (m *MockEmailService) CreateMailbox(ctx context.Context, opts CreateMailboxOpts) (*Mailbox, error) {
	if m.CreateMailboxFunc != nil {
		return m.CreateMailboxFunc(ctx, opts)
	}
	return nil, nil
}

func (m *MockEmailService) DeleteMailbox(ctx context.Context, id string) error {
	if m.DeleteMailboxFunc != nil {
		return m.DeleteMailboxFunc(ctx, id)
	}
	return nil
}

func (m *MockEmailService) RenameMailbox(ctx context.Context, id, newName string) error {
	if m.RenameMailboxFunc != nil {
		return m.RenameMailboxFunc(ctx, id, newName)
	}
	return nil
}

func (m *MockEmailService) SearchEmailsWithSnippets(ctx context.Context, filter *EmailSearchFilter, limit int) ([]Email, []SearchSnippet, error) {
	if m.SearchEmailsWithSnippetsFunc != nil {
		return m.SearchEmailsWithSnippetsFunc(ctx, filter, limit)
	}
	return nil, nil, nil
}

func (m *MockEmailService) ImportEmail(ctx context.Context, opts ImportEmailOpts) (string, error) {
	if m.ImportEmailFunc != nil {
		return m.ImportEmailFunc(ctx, opts)
	}
	return "", nil
}

func (m *MockEmailService) SaveDraft(ctx context.Context, opts SendEmailOpts) (string, error) {
	if m.SaveDraftFunc != nil {
		return m.SaveDraftFunc(ctx, opts)
	}
	return "", nil
}

func (m *MockEmailService) CreateReplyDraft(ctx context.Context, replyToID string, opts SendEmailOpts) (string, error) {
	if m.CreateReplyDraftFunc != nil {
		return m.CreateReplyDraftFunc(ctx, replyToID, opts)
	}
	return "", nil
}

// MockMaskedEmailService implements MaskedEmailService for testing.
// Each method can be overridden by setting the corresponding Func field.
// If a Func is not set, the method returns nil/empty values.
type MockMaskedEmailService struct {
	GetMaskedEmailsFunc              func(ctx context.Context) ([]MaskedEmail, error)
	GetMaskedEmailByEmailFunc        func(ctx context.Context, email string) (*MaskedEmail, error)
	GetMaskedEmailsForDomainFunc     func(ctx context.Context, domain string) ([]MaskedEmail, error)
	CreateMaskedEmailFunc            func(ctx context.Context, domain, description string) (*MaskedEmail, error)
	UpdateMaskedEmailStateFunc       func(ctx context.Context, id string, state MaskedEmailState) error
	UpdateMaskedEmailDescriptionFunc func(ctx context.Context, id, description string) error
}

func (m *MockMaskedEmailService) GetMaskedEmails(ctx context.Context) ([]MaskedEmail, error) {
	if m.GetMaskedEmailsFunc != nil {
		return m.GetMaskedEmailsFunc(ctx)
	}
	return nil, nil
}

func (m *MockMaskedEmailService) GetMaskedEmailByEmail(ctx context.Context, email string) (*MaskedEmail, error) {
	if m.GetMaskedEmailByEmailFunc != nil {
		return m.GetMaskedEmailByEmailFunc(ctx, email)
	}
	return nil, nil
}

func (m *MockMaskedEmailService) GetMaskedEmailsForDomain(ctx context.Context, domain string) ([]MaskedEmail, error) {
	if m.GetMaskedEmailsForDomainFunc != nil {
		return m.GetMaskedEmailsForDomainFunc(ctx, domain)
	}
	return nil, nil
}

func (m *MockMaskedEmailService) CreateMaskedEmail(ctx context.Context, domain, description string) (*MaskedEmail, error) {
	if m.CreateMaskedEmailFunc != nil {
		return m.CreateMaskedEmailFunc(ctx, domain, description)
	}
	return nil, nil
}

func (m *MockMaskedEmailService) UpdateMaskedEmailState(ctx context.Context, id string, state MaskedEmailState) error {
	if m.UpdateMaskedEmailStateFunc != nil {
		return m.UpdateMaskedEmailStateFunc(ctx, id, state)
	}
	return nil
}

func (m *MockMaskedEmailService) UpdateMaskedEmailDescription(ctx context.Context, id, description string) error {
	if m.UpdateMaskedEmailDescriptionFunc != nil {
		return m.UpdateMaskedEmailDescriptionFunc(ctx, id, description)
	}
	return nil
}

// MockVacationService implements VacationService for testing.
// Each method can be overridden by setting the corresponding Func field.
// If a Func is not set, the method returns nil/empty values.
type MockVacationService struct {
	GetVacationResponseFunc     func(ctx context.Context) (*VacationResponse, error)
	SetVacationResponseFunc     func(ctx context.Context, opts SetVacationResponseOpts) error
	DisableVacationResponseFunc func(ctx context.Context) error
}

func (m *MockVacationService) GetVacationResponse(ctx context.Context) (*VacationResponse, error) {
	if m.GetVacationResponseFunc != nil {
		return m.GetVacationResponseFunc(ctx)
	}
	return nil, nil
}

func (m *MockVacationService) SetVacationResponse(ctx context.Context, opts SetVacationResponseOpts) error {
	if m.SetVacationResponseFunc != nil {
		return m.SetVacationResponseFunc(ctx, opts)
	}
	return nil
}

func (m *MockVacationService) DisableVacationResponse(ctx context.Context) error {
	if m.DisableVacationResponseFunc != nil {
		return m.DisableVacationResponseFunc(ctx)
	}
	return nil
}

// MockContactsService implements ContactsService for testing.
// Each method can be overridden by setting the corresponding Func field.
// If a Func is not set, the method returns nil/empty values.
type MockContactsService struct {
	GetContactsFunc     func(ctx context.Context, addressBookID string, limit int) ([]Contact, error)
	GetContactByIDFunc  func(ctx context.Context, id string) (*Contact, error)
	CreateContactFunc   func(ctx context.Context, contact *Contact) (*Contact, error)
	UpdateContactFunc   func(ctx context.Context, id string, updates map[string]interface{}) (*Contact, error)
	DeleteContactFunc   func(ctx context.Context, id string) error
	SearchContactsFunc  func(ctx context.Context, query string, limit int) ([]Contact, error)
	GetAddressBooksFunc func(ctx context.Context) ([]AddressBook, error)
}

func (m *MockContactsService) GetContacts(ctx context.Context, addressBookID string, limit int) ([]Contact, error) {
	if m.GetContactsFunc != nil {
		return m.GetContactsFunc(ctx, addressBookID, limit)
	}
	return nil, nil
}

func (m *MockContactsService) GetContactByID(ctx context.Context, id string) (*Contact, error) {
	if m.GetContactByIDFunc != nil {
		return m.GetContactByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockContactsService) CreateContact(ctx context.Context, contact *Contact) (*Contact, error) {
	if m.CreateContactFunc != nil {
		return m.CreateContactFunc(ctx, contact)
	}
	return nil, nil
}

func (m *MockContactsService) UpdateContact(ctx context.Context, id string, updates map[string]interface{}) (*Contact, error) {
	if m.UpdateContactFunc != nil {
		return m.UpdateContactFunc(ctx, id, updates)
	}
	return nil, nil
}

func (m *MockContactsService) DeleteContact(ctx context.Context, id string) error {
	if m.DeleteContactFunc != nil {
		return m.DeleteContactFunc(ctx, id)
	}
	return nil
}

func (m *MockContactsService) SearchContacts(ctx context.Context, query string, limit int) ([]Contact, error) {
	if m.SearchContactsFunc != nil {
		return m.SearchContactsFunc(ctx, query, limit)
	}
	return nil, nil
}

func (m *MockContactsService) GetAddressBooks(ctx context.Context) ([]AddressBook, error) {
	if m.GetAddressBooksFunc != nil {
		return m.GetAddressBooksFunc(ctx)
	}
	return nil, nil
}

// MockCalendarService implements CalendarService for testing.
// Each method can be overridden by setting the corresponding Func field.
// If a Func is not set, the method returns nil/empty values.
type MockCalendarService struct {
	GetCalendarsFunc func(ctx context.Context) ([]Calendar, error)
	GetEventsFunc    func(ctx context.Context, calendarID string, from, to time.Time, limit int) ([]CalendarEvent, error)
	GetEventByIDFunc func(ctx context.Context, id string) (*CalendarEvent, error)
	CreateEventFunc  func(ctx context.Context, event *CalendarEvent) (*CalendarEvent, error)
	UpdateEventFunc  func(ctx context.Context, id string, updates map[string]interface{}) (*CalendarEvent, error)
	DeleteEventFunc  func(ctx context.Context, id string) error
}

func (m *MockCalendarService) GetCalendars(ctx context.Context) ([]Calendar, error) {
	if m.GetCalendarsFunc != nil {
		return m.GetCalendarsFunc(ctx)
	}
	return nil, nil
}

func (m *MockCalendarService) GetEvents(ctx context.Context, calendarID string, from, to time.Time, limit int) ([]CalendarEvent, error) {
	if m.GetEventsFunc != nil {
		return m.GetEventsFunc(ctx, calendarID, from, to, limit)
	}
	return nil, nil
}

func (m *MockCalendarService) GetEventByID(ctx context.Context, id string) (*CalendarEvent, error) {
	if m.GetEventByIDFunc != nil {
		return m.GetEventByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockCalendarService) CreateEvent(ctx context.Context, event *CalendarEvent) (*CalendarEvent, error) {
	if m.CreateEventFunc != nil {
		return m.CreateEventFunc(ctx, event)
	}
	return nil, nil
}

func (m *MockCalendarService) UpdateEvent(ctx context.Context, id string, updates map[string]interface{}) (*CalendarEvent, error) {
	if m.UpdateEventFunc != nil {
		return m.UpdateEventFunc(ctx, id, updates)
	}
	return nil, nil
}

func (m *MockCalendarService) DeleteEvent(ctx context.Context, id string) error {
	if m.DeleteEventFunc != nil {
		return m.DeleteEventFunc(ctx, id)
	}
	return nil
}

// MockQuotaService implements QuotaService for testing.
// Each method can be overridden by setting the corresponding Func field.
// If a Func is not set, the method returns nil/empty values.
type MockQuotaService struct {
	GetQuotasFunc func(ctx context.Context) ([]Quota, error)
}

func (m *MockQuotaService) GetQuotas(ctx context.Context) ([]Quota, error) {
	if m.GetQuotasFunc != nil {
		return m.GetQuotasFunc(ctx)
	}
	return nil, nil
}
