package jmap

import (
	"context"
	"fmt"
)

// VacationResponse represents auto-reply/vacation responder settings.
type VacationResponse struct {
	ID        string `json:"id"`
	IsEnabled bool   `json:"isEnabled"`
	FromDate  string `json:"fromDate,omitempty"` // RFC3339 or null
	ToDate    string `json:"toDate,omitempty"`   // RFC3339 or null
	Subject   string `json:"subject,omitempty"`
	TextBody  string `json:"textBody,omitempty"`
	HTMLBody  string `json:"htmlBody,omitempty"`
}

// GetVacationResponse retrieves the current vacation/auto-reply settings.
func (c *Client) GetVacationResponse(ctx context.Context) (*VacationResponse, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:vacationresponse"},
		MethodCalls: []MethodCall{
			{"VacationResponse/get", map[string]any{
				"accountId": session.AccountID,
			}, "getVacation"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	list, ok := result["list"].([]any)
	if !ok || len(list) == 0 {
		return nil, fmt.Errorf("no vacation response found")
	}

	vrData, ok := list[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected vacation response format")
	}

	return &VacationResponse{
		ID:        getString(vrData, "id"),
		IsEnabled: getBool(vrData, "isEnabled"),
		FromDate:  getString(vrData, "fromDate"),
		ToDate:    getString(vrData, "toDate"),
		Subject:   getString(vrData, "subject"),
		TextBody:  getString(vrData, "textBody"),
		HTMLBody:  getString(vrData, "htmlBody"),
	}, nil
}

// SetVacationResponseOpts contains options for updating vacation settings.
type SetVacationResponseOpts struct {
	IsEnabled bool
	FromDate  string // RFC3339 format or empty to clear
	ToDate    string // RFC3339 format or empty to clear
	Subject   string
	TextBody  string
	HTMLBody  string
}

// SetVacationResponse updates the vacation/auto-reply settings.
func (c *Client) SetVacationResponse(ctx context.Context, opts SetVacationResponseOpts) error {
	session, err := c.GetSession(ctx)
	if err != nil {
		return err
	}

	// First get the current vacation response to get its ID
	current, err := c.GetVacationResponse(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current vacation response: %w", err)
	}

	update := map[string]any{
		"isEnabled": opts.IsEnabled,
	}

	// Handle date fields - use null to clear
	if opts.FromDate != "" {
		update["fromDate"] = opts.FromDate
	} else {
		update["fromDate"] = nil
	}

	if opts.ToDate != "" {
		update["toDate"] = opts.ToDate
	} else {
		update["toDate"] = nil
	}

	if opts.Subject != "" {
		update["subject"] = opts.Subject
	}

	if opts.TextBody != "" {
		update["textBody"] = opts.TextBody
	}

	if opts.HTMLBody != "" {
		update["htmlBody"] = opts.HTMLBody
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:vacationresponse"},
		MethodCalls: []MethodCall{
			{"VacationResponse/set", map[string]any{
				"accountId": session.AccountID,
				"update": map[string]any{
					current.ID: update,
				},
			}, "setVacation"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected response format")
	}

	if notUpdated, ok := result["notUpdated"].(map[string]any); ok {
		if errInfo, exists := notUpdated[current.ID]; exists {
			return fmt.Errorf("failed to update vacation response: %v", errInfo)
		}
	}

	return nil
}

// DisableVacationResponse is a convenience method to turn off the vacation responder.
func (c *Client) DisableVacationResponse(ctx context.Context) error {
	return c.SetVacationResponse(ctx, SetVacationResponseOpts{IsEnabled: false})
}
