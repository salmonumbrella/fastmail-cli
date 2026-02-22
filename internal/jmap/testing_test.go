package jmap

import (
	"context"
	"errors"
	"testing"
)

func TestMockEmailService(t *testing.T) {
	ctx := context.Background()

	t.Run("GetEmails with custom func", func(t *testing.T) {
		mock := &MockEmailService{
			GetEmailsFunc: func(ctx context.Context, mailboxID string, limit int) ([]Email, error) {
				if mailboxID == "test-mailbox" && limit == 10 {
					return []Email{
						{ID: "email1", Subject: "Test Email 1"},
						{ID: "email2", Subject: "Test Email 2"},
					}, nil
				}
				return nil, errors.New("unexpected parameters")
			},
		}

		emails, err := mock.GetEmails(ctx, "test-mailbox", 10)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(emails) != 2 {
			t.Errorf("expected 2 emails, got %d", len(emails))
		}
		if emails[0].Subject != "Test Email 1" {
			t.Errorf("expected 'Test Email 1', got %q", emails[0].Subject)
		}
	})

	t.Run("GetEmails without custom func returns nil", func(t *testing.T) {
		mock := &MockEmailService{}

		emails, err := mock.GetEmails(ctx, "test-mailbox", 10)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if emails != nil {
			t.Errorf("expected nil emails, got %v", emails)
		}
	})

	t.Run("DeleteEmail with custom func", func(t *testing.T) {
		mock := &MockEmailService{
			DeleteEmailFunc: func(ctx context.Context, id string) error {
				if id == "valid-id" {
					return nil
				}
				return errors.New("not found")
			},
		}

		err := mock.DeleteEmail(ctx, "valid-id")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		err = mock.DeleteEmail(ctx, "invalid-id")
		if err == nil {
			t.Error("expected error for invalid ID, got nil")
		}
	})
}

func TestMockMaskedEmailService(t *testing.T) {
	ctx := context.Background()

	t.Run("GetMaskedEmails with custom func", func(t *testing.T) {
		mock := &MockMaskedEmailService{
			GetMaskedEmailsFunc: func(ctx context.Context) ([]MaskedEmail, error) {
				return []MaskedEmail{
					{ID: "masked1", Email: "alias1@test.fastmail.com"},
					{ID: "masked2", Email: "alias2@test.fastmail.com"},
				}, nil
			},
		}

		masked, err := mock.GetMaskedEmails(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(masked) != 2 {
			t.Errorf("expected 2 masked emails, got %d", len(masked))
		}
	})

	t.Run("CreateMaskedEmail with custom func", func(t *testing.T) {
		mock := &MockMaskedEmailService{
			CreateMaskedEmailFunc: func(ctx context.Context, domain, description string) (*MaskedEmail, error) {
				return &MaskedEmail{
					ID:          "new-masked",
					Email:       "generated@" + domain,
					Description: description,
				}, nil
			},
		}

		result, err := mock.CreateMaskedEmail(ctx, "test.fastmail.com", "Test alias")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.Email != "generated@test.fastmail.com" {
			t.Errorf("expected 'generated@test.fastmail.com', got %q", result.Email)
		}
		if result.Description != "Test alias" {
			t.Errorf("expected 'Test alias', got %q", result.Description)
		}
	})
}

func TestMockVacationService(t *testing.T) {
	ctx := context.Background()

	t.Run("GetVacationResponse with custom func", func(t *testing.T) {
		mock := &MockVacationService{
			GetVacationResponseFunc: func(ctx context.Context) (*VacationResponse, error) {
				return &VacationResponse{
					IsEnabled: true,
					Subject:   "Out of Office",
					TextBody:  "I'm away",
					FromDate:  "2025-01-01",
					ToDate:    "2025-01-07",
				}, nil
			},
		}

		vacation, err := mock.GetVacationResponse(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !vacation.IsEnabled {
			t.Error("expected vacation to be enabled")
		}
		if vacation.Subject != "Out of Office" {
			t.Errorf("expected 'Out of Office', got %q", vacation.Subject)
		}
	})

	t.Run("DisableVacationResponse with custom func", func(t *testing.T) {
		disabled := false
		mock := &MockVacationService{
			DisableVacationResponseFunc: func(ctx context.Context) error {
				disabled = true
				return nil
			},
		}

		err := mock.DisableVacationResponse(ctx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if !disabled {
			t.Error("expected DisableVacationResponseFunc to be called")
		}
	})
}
