package jmap

import "errors"

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
)
