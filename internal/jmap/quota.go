package jmap

import (
	"context"
	"fmt"
)

// Quota represents storage or resource quota information
// Implements JMAP Quota extension (RFC 9425)
type Quota struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Used         int64  `json:"used"`         // bytes or count
	Limit        int64  `json:"limit"`        // bytes or count, 0 = unlimited
	Scope        string `json:"scope"`        // account, mailbox
	ResourceType string `json:"resourceType"` // octets, message count
}

// GetQuotas retrieves all quotas for the account
func (c *Client) GetQuotas(ctx context.Context) ([]Quota, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// Check if quota capability is available
	if _, hasQuota := session.Capabilities["urn:ietf:params:jmap:quota"]; !hasQuota {
		return nil, ErrQuotaNotEnabled
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:quota"},
		MethodCalls: []MethodCall{
			{"Quota/get", map[string]any{
				"accountId": session.AccountID,
			}, "q0"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(resp.MethodResponses) == 0 {
		return nil, fmt.Errorf("no method responses received")
	}

	// Parse the response
	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	// Check for error
	if errType, errOK := result["type"].(string); errOK {
		desc := ""
		if d, descOK := result["description"].(string); descOK {
			desc = d
		}
		return nil, fmt.Errorf("quota error: %s - %s", errType, desc)
	}

	list, ok := result["list"].([]any)
	if !ok {
		return nil, fmt.Errorf("no quota list found in response")
	}

	quotas := make([]Quota, 0, len(list))
	for _, item := range list {
		quotaData, ok := item.(map[string]any)
		if !ok {
			continue
		}

		quota := Quota{
			ID:           getStringField(quotaData, "id"),
			Name:         getStringField(quotaData, "name"),
			Description:  getStringField(quotaData, "description"),
			Used:         getInt64Field(quotaData, "used"),
			Limit:        getInt64Field(quotaData, "limit"),
			Scope:        getStringField(quotaData, "scope"),
			ResourceType: getStringField(quotaData, "resourceType"),
		}
		quotas = append(quotas, quota)
	}

	return quotas, nil
}

// Helper function to safely extract string fields
func getStringField(data map[string]any, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

// Helper function to safely extract int64 fields
func getInt64Field(data map[string]any, key string) int64 {
	switch v := data[key].(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	default:
		return 0
	}
}
