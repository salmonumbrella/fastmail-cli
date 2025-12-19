package jmap

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Contact represents a JMAP contact (RFC 9610)
type Contact struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Emails      []ContactEmail   `json:"emails,omitempty"`
	Phones      []ContactPhone   `json:"phones,omitempty"`
	Addresses   []ContactAddress `json:"addresses,omitempty"`
	Company     string           `json:"company,omitempty"`
	JobTitle    string           `json:"jobTitle,omitempty"`
	Notes       string           `json:"notes,omitempty"`
	Birthday    string           `json:"birthday,omitempty"`
	Anniversary string           `json:"anniversary,omitempty"`
	Updated     time.Time        `json:"updated"`
}

// ContactEmail represents an email address for a contact
type ContactEmail struct {
	Type  string `json:"type"`  // home, work, other
	Value string `json:"value"`
}

// ContactPhone represents a phone number for a contact
type ContactPhone struct {
	Type  string `json:"type"`  // home, work, mobile, other
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

	if len(resp.MethodResponses) == 0 {
		return nil, fmt.Errorf("empty response from server")
	}

	// Check for error response
	methodName, ok := resp.MethodResponses[0][0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}
	if methodName == "error" {
		return nil, fmt.Errorf("API error: %v", resp.MethodResponses[0][1])
	}

	// Parse the result
	resultJSON, err := json.Marshal(resp.MethodResponses[0][1])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var result struct {
		List []AddressBook `json:"list"`
	}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
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

	if len(resp.MethodResponses) < 2 {
		return nil, fmt.Errorf("incomplete response from server")
	}

	// Parse the ContactCard/get response
	methodName, ok := resp.MethodResponses[1][0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}
	if methodName == "error" {
		return nil, fmt.Errorf("API error: %v", resp.MethodResponses[1][1])
	}

	resultJSON, err := json.Marshal(resp.MethodResponses[1][1])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var result struct {
		List []Contact `json:"list"`
	}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
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

	if len(resp.MethodResponses) == 0 {
		return nil, fmt.Errorf("empty response from server")
	}

	// Check for error response
	methodName, ok := resp.MethodResponses[0][0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}
	if methodName == "error" {
		return nil, fmt.Errorf("API error: %v", resp.MethodResponses[0][1])
	}

	resultJSON, err := json.Marshal(resp.MethodResponses[0][1])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var result struct {
		List []Contact `json:"list"`
	}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.List) == 0 {
		return nil, ErrContactNotFound
	}

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

	if len(resp.MethodResponses) == 0 {
		return nil, fmt.Errorf("empty response from server")
	}

	// Check for error response
	methodName, ok := resp.MethodResponses[0][0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}
	if methodName == "error" {
		return nil, fmt.Errorf("API error: %v", resp.MethodResponses[0][1])
	}

	resultJSON, err := json.Marshal(resp.MethodResponses[0][1])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var result struct {
		Created map[string]Contact `json:"created"`
	}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	created, ok := result.Created["new-contact"]
	if !ok {
		return nil, fmt.Errorf("contact creation failed")
	}

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

	if len(resp.MethodResponses) == 0 {
		return nil, fmt.Errorf("empty response from server")
	}

	// Check for error response
	methodName, ok := resp.MethodResponses[0][0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}
	if methodName == "error" {
		return nil, fmt.Errorf("API error: %v", resp.MethodResponses[0][1])
	}

	resultJSON, err := json.Marshal(resp.MethodResponses[0][1])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var result struct {
		Updated map[string]Contact `json:"updated"`
	}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	updated, ok := result.Updated[id]
	if !ok {
		return nil, fmt.Errorf("contact update failed")
	}

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

	if len(resp.MethodResponses) == 0 {
		return fmt.Errorf("empty response from server")
	}

	// Check for error response
	methodName, ok := resp.MethodResponses[0][0].(string)
	if !ok {
		return fmt.Errorf("invalid response format")
	}
	if methodName == "error" {
		return fmt.Errorf("API error: %v", resp.MethodResponses[0][1])
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

	if len(resp.MethodResponses) < 2 {
		return nil, fmt.Errorf("incomplete response from server")
	}

	// Parse the ContactCard/get response
	methodName, ok := resp.MethodResponses[1][0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}
	if methodName == "error" {
		return nil, fmt.Errorf("API error: %v", resp.MethodResponses[1][1])
	}

	resultJSON, err := json.Marshal(resp.MethodResponses[1][1])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var result struct {
		List []Contact `json:"list"`
	}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.List, nil
}
