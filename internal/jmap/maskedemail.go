package jmap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// MaskedEmailState represents the possible states of a masked email
type MaskedEmailState string

const (
	MaskedEmailPending  MaskedEmailState = "pending"
	MaskedEmailEnabled  MaskedEmailState = "enabled"
	MaskedEmailDisabled MaskedEmailState = "disabled"
	MaskedEmailDeleted  MaskedEmailState = "deleted"
)

// MaskedEmail represents a Fastmail masked email alias
type MaskedEmail struct {
	ID            string           `json:"id"`
	Email         string           `json:"email"`
	State         MaskedEmailState `json:"state"`
	ForDomain     string           `json:"forDomain"`
	Description   string           `json:"description"`
	CreatedAt     time.Time        `json:"createdAt,omitempty"`
	LastMessageAt *time.Time       `json:"lastMessageAt,omitempty"`
}

// maskedEmailCreate defines the payload for creating a masked email
type maskedEmailCreate struct {
	ForDomain   string `json:"forDomain"`
	Description string `json:"description,omitempty"`
}

// maskedEmailUpdate defines the payload for updating a masked email
type maskedEmailUpdate struct {
	State       *MaskedEmailState `json:"state,omitempty"`
	Description *string           `json:"description,omitempty"`
}

const maskedEmailNamespace = "https://www.fastmail.com/dev/maskedemail"

// GetMaskedEmails retrieves all masked email aliases for the account
func (c *Client) GetMaskedEmails(ctx context.Context) ([]MaskedEmail, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", maskedEmailNamespace},
		MethodCalls: []MethodCall{
			{
				"MaskedEmail/get",
				map[string]any{
					"accountId":  session.AccountID,
					"properties": []string{"id", "email", "forDomain", "state", "description", "createdAt", "lastMessageAt"},
				},
				"0",
			},
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
		List []MaskedEmail `json:"list"`
	}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.List, nil
}

// GetMaskedEmailByEmail retrieves a specific masked email by its address
func (c *Client) GetMaskedEmailByEmail(ctx context.Context, email string) (*MaskedEmail, error) {
	aliases, err := c.GetMaskedEmails(ctx)
	if err != nil {
		return nil, err
	}

	for _, alias := range aliases {
		if alias.Email == email {
			return &alias, nil
		}
	}

	return nil, fmt.Errorf("masked email not found: %s", email)
}

// GetMaskedEmailsForDomain retrieves masked emails for a specific domain
func (c *Client) GetMaskedEmailsForDomain(ctx context.Context, domain string) ([]MaskedEmail, error) {
	normalizedDomain, err := NormalizeDomain(domain)
	if err != nil {
		return nil, err
	}

	aliases, err := c.GetMaskedEmails(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []MaskedEmail
	for _, alias := range aliases {
		if alias.State == MaskedEmailDeleted {
			continue
		}
		if domainsMatch(alias.ForDomain, normalizedDomain) {
			filtered = append(filtered, alias)
		}
	}

	return filtered, nil
}

// CreateMaskedEmail creates a new masked email alias for a domain
func (c *Client) CreateMaskedEmail(ctx context.Context, domain, description string) (*MaskedEmail, error) {
	normalizedDomain, err := NormalizeDomain(domain)
	if err != nil {
		return nil, err
	}

	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", maskedEmailNamespace},
		MethodCalls: []MethodCall{
			{
				"MaskedEmail/set",
				map[string]any{
					"accountId": session.AccountID,
					"create": map[string]maskedEmailCreate{
						"new": {
							ForDomain:   normalizedDomain,
							Description: description,
						},
					},
				},
				"0",
			},
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
		Created map[string]MaskedEmail `json:"created"`
	}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	created, ok := result.Created["new"]
	if !ok {
		return nil, fmt.Errorf("failed to create masked email")
	}

	return &created, nil
}

// UpdateMaskedEmailState updates the state of a masked email
func (c *Client) UpdateMaskedEmailState(ctx context.Context, id string, state MaskedEmailState) error {
	session, err := c.GetSession(ctx)
	if err != nil {
		return err
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", maskedEmailNamespace},
		MethodCalls: []MethodCall{
			{
				"MaskedEmail/set",
				map[string]any{
					"accountId": session.AccountID,
					"update": map[string]maskedEmailUpdate{
						id: {
							State: &state,
						},
					},
				},
				"0",
			},
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

	// Parse the result
	resultJSON, err := json.Marshal(resp.MethodResponses[0][1])
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	var result struct {
		Updated map[string]any `json:"updated"`
	}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if _, ok := result.Updated[id]; !ok {
		return fmt.Errorf("failed to update masked email")
	}

	return nil
}

// UpdateMaskedEmailDescription updates the description of a masked email
func (c *Client) UpdateMaskedEmailDescription(ctx context.Context, id, description string) error {
	session, err := c.GetSession(ctx)
	if err != nil {
		return err
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", maskedEmailNamespace},
		MethodCalls: []MethodCall{
			{
				"MaskedEmail/set",
				map[string]any{
					"accountId": session.AccountID,
					"update": map[string]maskedEmailUpdate{
						id: {
							Description: &description,
						},
					},
				},
				"0",
			},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return err
	}

	if len(resp.MethodResponses) == 0 {
		return fmt.Errorf("empty response from server")
	}

	// Parse the result
	resultJSON, err := json.Marshal(resp.MethodResponses[0][1])
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	var result struct {
		Updated map[string]any `json:"updated"`
	}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if _, ok := result.Updated[id]; !ok {
		return fmt.Errorf("failed to update masked email description")
	}

	return nil
}

// NormalizeDomain converts a user-supplied URL or domain into a canonical origin
// string consisting of "<scheme>://<host>".
func NormalizeDomain(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("domain cannot be empty")
	}

	// Add scheme if missing
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("failed to parse domain %q: %w", input, err)
	}

	host := parsed.Hostname()
	if host == "" {
		return "", fmt.Errorf("invalid domain %q: missing host", input)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme == "" {
		scheme = "https"
	}

	host = strings.TrimSuffix(strings.ToLower(host), ".")

	return fmt.Sprintf("%s://%s", scheme, host), nil
}

// domainsMatch compares two domain strings by normalizing them
func domainsMatch(a, b string) bool {
	na, errA := NormalizeDomain(a)
	nb, errB := NormalizeDomain(b)
	if errA == nil && errB == nil {
		return na == nb
	}

	// Fallback: compare trimmed strings case-insensitively
	trimA := strings.TrimRight(strings.ToLower(strings.TrimSpace(a)), "/")
	trimB := strings.TrimRight(strings.ToLower(strings.TrimSpace(b)), "/")
	return trimA == trimB
}

// LooksLikeEmail returns true if the input looks like an email address
func LooksLikeEmail(input string) bool {
	return strings.Count(input, "@") == 1 && !strings.ContainsAny(input, " \t")
}
