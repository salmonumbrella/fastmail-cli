package jmap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/google/uuid"
)

// SieveBlocks represents the Fastmail Sieve script blocks.
// Only SieveAtStart, SieveAtMiddle, and SieveAtEnd are writable.
type SieveBlocks struct {
	ID              string `json:"id"`              // Always "singleton"
	SieveRequire    string `json:"sieveRequire"`    // Read-only: require statements
	SieveAtStart    string `json:"sieveAtStart"`    // Writable: custom sieve at start
	SieveForBlocked string `json:"sieveForBlocked"` // Read-only: blocked senders
	SieveAtMiddle   string `json:"sieveAtMiddle"`   // Writable: custom sieve in middle
	SieveForRules   string `json:"sieveForRules"`   // Read-only: UI-managed rules
	SieveAtEnd      string `json:"sieveAtEnd"`      // Writable: custom sieve at end
}

// SieveClient is a specialized JMAP client for Sieve operations.
// It uses browser session credentials instead of API tokens.
type SieveClient struct {
	token      string
	cookie     string
	sessionURL string
	apiURL     string
	http       *http.Client
	accountID  string
}

// NewSieveClient creates a Sieve client with browser session credentials.
func NewSieveClient(token, cookie, sessionURL, apiURL string) *SieveClient {
	return &SieveClient{
		token:      token,
		cookie:     cookie,
		sessionURL: sessionURL,
		apiURL:     apiURL,
		http:       newSecureHTTPClient(),
	}
}

// NewSieveClientFromCredentials creates a Sieve client using default Fastmail URLs.
func NewSieveClientFromCredentials(token, cookie string) *SieveClient {
	return &SieveClient{
		token:      token,
		cookie:     cookie,
		sessionURL: "https://api.fastmail.com/jmap/session",
		apiURL:     "https://api.fastmail.com/jmap/api",
		http:       newSecureHTTPClient(),
	}
}

func (c *SieveClient) getAccountID(ctx context.Context) (string, error) {
	if c.accountID != "" {
		return c.accountID, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.sessionURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Cookie", c.cookie)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("session request failed: %s - %s", resp.Status, string(body))
	}

	var session struct {
		Accounts map[string]any `json:"accounts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return "", err
	}

	ids := make([]string, 0, len(session.Accounts))
	for id := range session.Accounts {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("no accounts found in session")
	}
	sort.Strings(ids)
	c.accountID = ids[0]
	return ids[0], nil
}

// GetSieveBlocks retrieves the current Sieve script blocks.
func (c *SieveClient) GetSieveBlocks(ctx context.Context) (*SieveBlocks, error) {
	accountID, err := c.getAccountID(ctx)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"using": []string{
			"urn:ietf:params:jmap:core",
			"https://www.fastmail.com/dev/rules",
			"https://www.fastmail.com/dev/user",
		},
		"methodCalls": [][]any{
			{"SieveBlocks/get", map[string]any{
				"accountId": accountID,
				"ids":       []string{"singleton"},
			}, "0"},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Cookie", c.cookie)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", uuid.New().String())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sieve request failed: %s - %s", resp.Status, string(respBody))
	}

	var result struct {
		MethodResponses [][]any `json:"methodResponses"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.MethodResponses) == 0 {
		return nil, fmt.Errorf("empty response from server")
	}

	data, ok := result.MethodResponses[0][1].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	list, ok := data["list"].([]any)
	if !ok || len(list) == 0 {
		return nil, fmt.Errorf("no sieve blocks found")
	}

	blockData, ok := list[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected sieve block format")
	}

	return &SieveBlocks{
		ID:              getString(blockData, "id"),
		SieveRequire:    getString(blockData, "sieveRequire"),
		SieveAtStart:    getString(blockData, "sieveAtStart"),
		SieveForBlocked: getString(blockData, "sieveForBlocked"),
		SieveAtMiddle:   getString(blockData, "sieveAtMiddle"),
		SieveForRules:   getString(blockData, "sieveForRules"),
		SieveAtEnd:      getString(blockData, "sieveAtEnd"),
	}, nil
}

// SetSieveBlocksOpts contains the writable Sieve block fields.
type SetSieveBlocksOpts struct {
	SieveAtStart  *string // nil = don't change
	SieveAtMiddle *string // nil = don't change
	SieveAtEnd    *string // nil = don't change
}

// SetSieveBlocks updates the writable Sieve script blocks.
func (c *SieveClient) SetSieveBlocks(ctx context.Context, opts SetSieveBlocksOpts) error {
	accountID, err := c.getAccountID(ctx)
	if err != nil {
		return err
	}

	update := map[string]any{}
	if opts.SieveAtStart != nil {
		update["sieveAtStart"] = *opts.SieveAtStart
	}
	if opts.SieveAtMiddle != nil {
		update["sieveAtMiddle"] = *opts.SieveAtMiddle
	}
	if opts.SieveAtEnd != nil {
		update["sieveAtEnd"] = *opts.SieveAtEnd
	}

	if len(update) == 0 {
		return nil // Nothing to update
	}

	reqBody := map[string]any{
		"using": []string{
			"urn:ietf:params:jmap:core",
			"https://www.fastmail.com/dev/rules",
			"https://www.fastmail.com/dev/user",
		},
		"methodCalls": [][]any{
			{"SieveBlocks/set", map[string]any{
				"accountId": accountID,
				"update": map[string]any{
					"singleton": update,
				},
			}, "0"},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Cookie", c.cookie)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", uuid.New().String())

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sieve update failed: %s - %s", resp.Status, string(respBody))
	}

	var result struct {
		MethodResponses [][]any `json:"methodResponses"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// Check for errors
	if len(result.MethodResponses) > 0 {
		if data, ok := result.MethodResponses[0][1].(map[string]any); ok {
			if notUpdated, ok := data["notUpdated"].(map[string]any); ok {
				if errInfo, exists := notUpdated["singleton"]; exists {
					return fmt.Errorf("failed to update sieve: %v", errInfo)
				}
			}
		}
	}

	return nil
}
