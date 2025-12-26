package jmap

import (
	"errors"
	"fmt"
	"time"
)

// Sentinel errors for JMAP operations
var (
	// ErrNoAccounts indicates no accounts were found in session
	ErrNoAccounts = errors.New("no accounts found in session")

	// ErrEmailNotFound indicates the requested email was not found
	ErrEmailNotFound = errors.New("email not found")

	// ErrContactNotFound indicates the requested contact was not found
	ErrContactNotFound = errors.New("contact not found")

	// ErrThreadNotFound indicates the requested thread was not found
	ErrThreadNotFound = errors.New("thread not found")

	// ErrMailboxNotFound indicates the requested mailbox was not found
	ErrMailboxNotFound = errors.New("mailbox not found")

	// ErrContactsNotEnabled indicates contacts API is not available
	ErrContactsNotEnabled = errors.New("contacts API not enabled for this account")

	// ErrCalendarsNotEnabled indicates calendars API is not available
	ErrCalendarsNotEnabled = errors.New("calendars API not enabled for this account")

	// ErrEventNotFound indicates the requested calendar event was not found
	ErrEventNotFound = errors.New("calendar event not found")

	// ErrNoIdentities indicates no sending identities were found
	ErrNoIdentities = errors.New("no sending identities found")

	// ErrInvalidFromAddress indicates the from address is not verified
	ErrInvalidFromAddress = errors.New("from address not verified for sending")

	// ErrNoDraftsMailbox indicates drafts mailbox was not found
	ErrNoDraftsMailbox = errors.New("drafts mailbox not found")

	// ErrNoSentMailbox indicates sent mailbox was not found
	ErrNoSentMailbox = errors.New("sent mailbox not found")

	// ErrNoTrashMailbox indicates trash mailbox was not found
	ErrNoTrashMailbox = errors.New("trash mailbox not found")

	// ErrNoBody indicates neither text nor HTML body was provided
	ErrNoBody = errors.New("either text or HTML body must be provided")

	// ErrQuotaNotEnabled indicates quota API is not available
	ErrQuotaNotEnabled = errors.New("quota API not enabled for this account")
)

// Typed errors for specific error conditions

// ValidationError represents an input validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// RateLimitError indicates the request was rate limited
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited, retry after %v", e.RetryAfter)
}

// CircuitBreakerError indicates the circuit breaker is open
type CircuitBreakerError struct{}

func (e *CircuitBreakerError) Error() string {
	return "circuit breaker open: service temporarily unavailable"
}

// AuthError represents an authentication error
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("authentication error: %s", e.Message)
}

// Helper functions for type checking errors

// IsValidationError checks if an error is a ValidationError
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

// IsRateLimitError checks if an error is a RateLimitError
func IsRateLimitError(err error) bool {
	var rle *RateLimitError
	return errors.As(err, &rle)
}

// IsCircuitBreakerError checks if an error is a CircuitBreakerError
func IsCircuitBreakerError(err error) bool {
	var cbe *CircuitBreakerError
	return errors.As(err, &cbe)
}

// IsAuthError checks if an error is an AuthError
func IsAuthError(err error) bool {
	var ae *AuthError
	return errors.As(err, &ae)
}

// JMAPError represents a JMAP protocol error response.
type JMAPError struct {
	Type        string // e.g., "invalidArguments", "serverFail"
	Description string
}

func (e *JMAPError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("JMAP error (%s): %s", e.Type, e.Description)
	}
	return fmt.Sprintf("JMAP error: %s", e.Type)
}

// NotFoundError represents a resource not found error.
type NotFoundError struct {
	Resource string // e.g., "email", "contact", "mailbox"
	ID       string
}

func (e *NotFoundError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// RequestContext wraps an error with JMAP method context.
type RequestContext struct {
	Method string // e.g., "Email/get", "Mailbox/query"
	Err    error
}

func (e *RequestContext) Error() string {
	return fmt.Sprintf("%s: %v", e.Method, e.Err)
}

func (e *RequestContext) Unwrap() error {
	return e.Err
}

// IsJMAPError checks if an error is a JMAPError.
func IsJMAPError(err error) bool {
	var je *JMAPError
	return errors.As(err, &je)
}

// IsNotFoundError checks if an error indicates a resource was not found.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var nfe *NotFoundError
	if errors.As(err, &nfe) {
		return true
	}
	// Also check sentinel errors
	return errors.Is(err, ErrEmailNotFound) ||
		errors.Is(err, ErrContactNotFound) ||
		errors.Is(err, ErrThreadNotFound) ||
		errors.Is(err, ErrMailboxNotFound) ||
		errors.Is(err, ErrEventNotFound)
}
