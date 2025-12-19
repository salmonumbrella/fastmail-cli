package jmap

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Calendar represents a JMAP calendar
type Calendar struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Color         string  `json:"color,omitempty"`
	IsVisible     bool    `json:"isVisible"`
	IsSubscribed  bool    `json:"isSubscribed"`
	DefaultAlerts []Alert `json:"defaultAlerts,omitempty"`
}

// CalendarEvent represents a JMAP calendar event
type CalendarEvent struct {
	ID           string           `json:"id"`
	CalendarID   string           `json:"calendarId"`
	Title        string           `json:"title"`
	Description  string           `json:"description,omitempty"`
	Location     string           `json:"location,omitempty"`
	Start        time.Time        `json:"start"`
	End          time.Time        `json:"end"`
	TimeZone     string           `json:"timeZone,omitempty"`
	IsAllDay     bool             `json:"isAllDay"`
	Status       string           `json:"status"` // confirmed, tentative, cancelled
	Recurrence   *RecurrenceRule  `json:"recurrenceRule,omitempty"`
	Alerts       []Alert          `json:"alerts,omitempty"`
	Participants []Participant    `json:"participants,omitempty"`
	Updated      time.Time        `json:"updated"`
}

// RecurrenceRule represents a recurring event rule
type RecurrenceRule struct {
	Frequency string `json:"frequency"` // daily, weekly, monthly, yearly
	Interval  int    `json:"interval,omitempty"`
	Until     string `json:"until,omitempty"`
	Count     int    `json:"count,omitempty"`
}

// Alert represents a calendar event alert/reminder
type Alert struct {
	Trigger string `json:"trigger"` // e.g., "-PT15M" (15 min before)
	Action  string `json:"action"`  // display, email
}

// Participant represents an event participant
type Participant struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Status string `json:"status"` // accepted, declined, tentative, needs-action
}

const calendarsCapability = "urn:ietf:params:jmap:calendars"

// GetCalendars retrieves all calendars for the account
func (c *Client) GetCalendars(ctx context.Context) ([]Calendar, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// Check if calendars capability is available
	if _, ok := session.Capabilities[calendarsCapability]; !ok {
		return nil, ErrCalendarsNotEnabled
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", calendarsCapability},
		MethodCalls: []MethodCall{
			{"Calendar/get", map[string]any{
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
		List []Calendar `json:"list"`
	}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.List, nil
}

// GetEvents retrieves calendar events within a date range
func (c *Client) GetEvents(ctx context.Context, calendarID string, from, to time.Time, limit int) ([]CalendarEvent, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// Check if calendars capability is available
	if _, ok := session.Capabilities[calendarsCapability]; !ok {
		return nil, ErrCalendarsNotEnabled
	}

	if limit <= 0 {
		limit = 100
	}

	// Build filter
	filter := map[string]any{}
	if calendarID != "" {
		filter["inCalendar"] = calendarID
	}
	if !from.IsZero() {
		filter["after"] = from.Format(time.RFC3339)
	}
	if !to.IsZero() {
		filter["before"] = to.Format(time.RFC3339)
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", calendarsCapability},
		MethodCalls: []MethodCall{
			{"CalendarEvent/query", map[string]any{
				"accountId": session.AccountID,
				"filter":    filter,
				"limit":     limit,
			}, "0"},
			{"CalendarEvent/get", map[string]any{
				"accountId": session.AccountID,
				"#ids": map[string]any{
					"resultOf": "0",
					"name":     "CalendarEvent/query",
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

	// Parse the CalendarEvent/get response
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
		List []CalendarEvent `json:"list"`
	}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.List, nil
}

// GetEventByID retrieves a specific calendar event by ID
func (c *Client) GetEventByID(ctx context.Context, id string) (*CalendarEvent, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// Check if calendars capability is available
	if _, ok := session.Capabilities[calendarsCapability]; !ok {
		return nil, ErrCalendarsNotEnabled
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", calendarsCapability},
		MethodCalls: []MethodCall{
			{"CalendarEvent/get", map[string]any{
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
		List []CalendarEvent `json:"list"`
	}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.List) == 0 {
		return nil, ErrEventNotFound
	}

	return &result.List[0], nil
}

// CreateEvent creates a new calendar event
func (c *Client) CreateEvent(ctx context.Context, event *CalendarEvent) (*CalendarEvent, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// Check if calendars capability is available
	if _, ok := session.Capabilities[calendarsCapability]; !ok {
		return nil, ErrCalendarsNotEnabled
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", calendarsCapability},
		MethodCalls: []MethodCall{
			{"CalendarEvent/set", map[string]any{
				"accountId": session.AccountID,
				"create": map[string]any{
					"new-event": event,
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
		Created map[string]CalendarEvent `json:"created"`
	}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	created, ok := result.Created["new-event"]
	if !ok {
		return nil, fmt.Errorf("event creation failed")
	}

	return &created, nil
}

// UpdateEvent updates an existing calendar event
func (c *Client) UpdateEvent(ctx context.Context, id string, updates map[string]interface{}) (*CalendarEvent, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// Check if calendars capability is available
	if _, ok := session.Capabilities[calendarsCapability]; !ok {
		return nil, ErrCalendarsNotEnabled
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", calendarsCapability},
		MethodCalls: []MethodCall{
			{"CalendarEvent/set", map[string]any{
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
		Updated map[string]CalendarEvent `json:"updated"`
	}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	updated, ok := result.Updated[id]
	if !ok {
		return nil, fmt.Errorf("event update failed")
	}

	return &updated, nil
}

// DeleteEvent deletes a calendar event by ID
func (c *Client) DeleteEvent(ctx context.Context, id string) error {
	session, err := c.GetSession(ctx)
	if err != nil {
		return err
	}

	// Check if calendars capability is available
	if _, ok := session.Capabilities[calendarsCapability]; !ok {
		return ErrCalendarsNotEnabled
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", calendarsCapability},
		MethodCalls: []MethodCall{
			{"CalendarEvent/set", map[string]any{
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
