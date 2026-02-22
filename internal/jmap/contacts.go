package jmap

import (
	"context"
	"fmt"
	"time"
)

// Contact represents a JMAP contact (RFC 9610)
type Contact struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Emails      []ContactEmail   `json:"emails"`
	Phones      []ContactPhone   `json:"phones"`
	Addresses   []ContactAddress `json:"addresses"`
	Company     string           `json:"company,omitempty"`
	JobTitle    string           `json:"jobTitle,omitempty"`
	Notes       string           `json:"notes,omitempty"`
	Birthday    string           `json:"birthday,omitempty"`
	Anniversary string           `json:"anniversary,omitempty"`
	Updated     time.Time        `json:"updated"`
}

// ContactEmail represents an email address for a contact
type ContactEmail struct {
	Type  string `json:"type"` // home, work, other
	Value string `json:"value"`
}

// ContactPhone represents a phone number for a contact
type ContactPhone struct {
	Type  string `json:"type"` // home, work, mobile, other
	Value string `json:"value"`
}

// ContactAddress represents a physical address for a contact
type ContactAddress struct {
	Type       string `json:"type"` // home, work, other
	Street     string `json:"street,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
	Country    string `json:"country,omitempty"`
}

// AddressBook represents a JMAP address book
type AddressBook struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	IsDefault    bool   `json:"isDefault"`
	IsSubscribed bool   `json:"isSubscribed"`
}

// EnsureSlices initializes nil slice fields to empty slices so JSON
// serialization produces [] instead of null.
func (c *Contact) EnsureSlices() {
	if c.Emails == nil {
		c.Emails = []ContactEmail{}
	}
	if c.Phones == nil {
		c.Phones = []ContactPhone{}
	}
	if c.Addresses == nil {
		c.Addresses = []ContactAddress{}
	}
}

const contactsCapability = "urn:ietf:params:jmap:contacts"

// GetAddressBooks retrieves all address books for the account
func (c *Client) GetAddressBooks(ctx context.Context) ([]AddressBook, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// Check if contacts capability is available
	if _, ok := session.Capabilities[contactsCapability]; !ok {
		return nil, ErrContactsNotEnabled
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", contactsCapability},
		MethodCalls: []MethodCall{
			{"AddressBook/get", map[string]any{
				"accountId": session.AccountID,
			}, "0"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	result, err := decodeMethodResponse[struct {
		List []AddressBook `json:"list"`
	}](resp, 0)
	if err != nil {
		return nil, err
	}

	return result.List, nil
}

// GetContacts retrieves contacts from an address book with optional limit
func (c *Client) GetContacts(ctx context.Context, addressBookID string, limit int) ([]Contact, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// Check if contacts capability is available
	if _, ok := session.Capabilities[contactsCapability]; !ok {
		return nil, ErrContactsNotEnabled
	}

	if limit <= 0 {
		limit = 100
	}

	// Build filter
	filter := map[string]any{}
	if addressBookID != "" {
		filter["inAddressBook"] = addressBookID
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", contactsCapability},
		MethodCalls: []MethodCall{
			{"ContactCard/query", map[string]any{
				"accountId": session.AccountID,
				"filter":    filter,
				"limit":     limit,
			}, "0"},
			{"ContactCard/get", map[string]any{
				"accountId": session.AccountID,
				"#ids": map[string]any{
					"resultOf": "0",
					"name":     "ContactCard/query",
					"path":     "/ids",
				},
			}, "1"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	result, err := decodeMethodResponse[struct {
		List []Contact `json:"list"`
	}](resp, 1)
	if err != nil {
		return nil, err
	}

	for i := range result.List {
		result.List[i].EnsureSlices()
	}

	return result.List, nil
}

// GetContactByID retrieves a specific contact by ID
func (c *Client) GetContactByID(ctx context.Context, id string) (*Contact, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// Check if contacts capability is available
	if _, ok := session.Capabilities[contactsCapability]; !ok {
		return nil, ErrContactsNotEnabled
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", contactsCapability},
		MethodCalls: []MethodCall{
			{"ContactCard/get", map[string]any{
				"accountId": session.AccountID,
				"ids":       []string{id},
			}, "0"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	result, err := decodeMethodResponse[struct {
		List []Contact `json:"list"`
	}](resp, 0)
	if err != nil {
		return nil, err
	}

	if len(result.List) == 0 {
		return nil, ErrContactNotFound
	}

	result.List[0].EnsureSlices()
	return &result.List[0], nil
}

// CreateContact creates a new contact
func (c *Client) CreateContact(ctx context.Context, contact *Contact) (*Contact, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// Check if contacts capability is available
	if _, ok := session.Capabilities[contactsCapability]; !ok {
		return nil, ErrContactsNotEnabled
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", contactsCapability},
		MethodCalls: []MethodCall{
			{"ContactCard/set", map[string]any{
				"accountId": session.AccountID,
				"create": map[string]any{
					"new-contact": contact,
				},
			}, "0"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	result, err := decodeMethodResponse[struct {
		Created map[string]Contact `json:"created"`
	}](resp, 0)
	if err != nil {
		return nil, err
	}

	created, ok := result.Created["new-contact"]
	if !ok {
		return nil, fmt.Errorf("contact creation failed")
	}

	created.EnsureSlices()
	return &created, nil
}

// UpdateContact updates an existing contact
func (c *Client) UpdateContact(ctx context.Context, id string, updates map[string]interface{}) (*Contact, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// Check if contacts capability is available
	if _, ok := session.Capabilities[contactsCapability]; !ok {
		return nil, ErrContactsNotEnabled
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", contactsCapability},
		MethodCalls: []MethodCall{
			{"ContactCard/set", map[string]any{
				"accountId": session.AccountID,
				"update": map[string]any{
					id: updates,
				},
			}, "0"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	result, err := decodeMethodResponse[struct {
		Updated map[string]Contact `json:"updated"`
	}](resp, 0)
	if err != nil {
		return nil, err
	}

	updated, ok := result.Updated[id]
	if !ok {
		return nil, fmt.Errorf("contact update failed")
	}

	updated.EnsureSlices()
	return &updated, nil
}

// DeleteContact deletes a contact by ID
func (c *Client) DeleteContact(ctx context.Context, id string) error {
	session, err := c.GetSession(ctx)
	if err != nil {
		return err
	}

	// Check if contacts capability is available
	if _, ok := session.Capabilities[contactsCapability]; !ok {
		return ErrContactsNotEnabled
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", contactsCapability},
		MethodCalls: []MethodCall{
			{"ContactCard/set", map[string]any{
				"accountId": session.AccountID,
				"destroy":   []string{id},
			}, "0"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return err
	}

	if _, err := decodeMethodResponse[map[string]any](resp, 0); err != nil {
		return err
	}

	return nil
}

// SearchContacts searches for contacts matching a query string
func (c *Client) SearchContacts(ctx context.Context, query string, limit int) ([]Contact, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// Check if contacts capability is available
	if _, ok := session.Capabilities[contactsCapability]; !ok {
		return nil, ErrContactsNotEnabled
	}

	if limit <= 0 {
		limit = 50
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", contactsCapability},
		MethodCalls: []MethodCall{
			{"ContactCard/query", map[string]any{
				"accountId": session.AccountID,
				"filter": map[string]any{
					"text": query,
				},
				"limit": limit,
			}, "0"},
			{"ContactCard/get", map[string]any{
				"accountId": session.AccountID,
				"#ids": map[string]any{
					"resultOf": "0",
					"name":     "ContactCard/query",
					"path":     "/ids",
				},
			}, "1"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	result, err := decodeMethodResponse[struct {
		List []Contact `json:"list"`
	}](resp, 1)
	if err != nil {
		return nil, err
	}

	for i := range result.List {
		result.List[i].EnsureSlices()
	}

	return result.List, nil
}
