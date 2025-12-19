package jmap

import (
	"context"
	"io"
)

// MockEmailService implements EmailService for testing.
// Each method can be overridden by setting the corresponding Func field.
// If a Func is not set, the method returns nil/empty values.
type MockEmailService struct {
	GetEmailsFunc                func(ctx context.Context, mailboxID string, limit int) ([]Email, error)
	SearchEmailsFunc             func(ctx context.Context, query string, limit int) ([]Email, error)
	GetEmailByIDFunc             func(ctx context.Context, id string) (*Email, error)
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
	SearchEmailsWithSnippetsFunc func(ctx context.Context, query string, limit int) ([]Email, []SearchSnippet, error)
	ImportEmailFunc              func(ctx context.Context, opts ImportEmailOpts) (string, error)
}

func (m *MockEmailService) GetEmails(ctx context.Context, mailboxID string, limit int) ([]Email, error) {
	if m.GetEmailsFunc != nil {
		return m.GetEmailsFunc(ctx, mailboxID, limit)
	}
	return nil, nil
}

func (m *MockEmailService) SearchEmails(ctx context.Context, query string, limit int) ([]Email, error) {
	if m.SearchEmailsFunc != nil {
		return m.SearchEmailsFunc(ctx, query, limit)
	}
	return nil, nil
}

func (m *MockEmailService) GetEmailByID(ctx context.Context, id string) (*Email, error) {
	if m.GetEmailByIDFunc != nil {
		return m.GetEmailByIDFunc(ctx, id)
	}
	return nil, nil
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

func (m *MockEmailService) SearchEmailsWithSnippets(ctx context.Context, query string, limit int) ([]Email, []SearchSnippet, error) {
	if m.SearchEmailsWithSnippetsFunc != nil {
		return m.SearchEmailsWithSnippetsFunc(ctx, query, limit)
	}
	return nil, nil, nil
}

func (m *MockEmailService) ImportEmail(ctx context.Context, opts ImportEmailOpts) (string, error) {
	if m.ImportEmailFunc != nil {
		return m.ImportEmailFunc(ctx, opts)
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
