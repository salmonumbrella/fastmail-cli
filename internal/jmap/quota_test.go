package jmap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetQuotas(t *testing.T) {
	tests := []struct {
		name           string
		sessionResp    string
		apiResp        string
		wantErr        bool
		expectedQuotas int
	}{
		{
			name: "successful quota retrieval",
			sessionResp: `{
				"apiUrl": "API_URL",
				"uploadUrl": "API_URL/{accountId}/",
				"downloadUrl": "API_URL",
				"accounts": {"test-account": {}},
				"capabilities": {
					"urn:ietf:params:jmap:core": {},
					"urn:ietf:params:jmap:quota": {}
				}
			}`,
			apiResp: `{
				"methodResponses": [
					["Quota/get", {
						"list": [
							{
								"id": "quota-1",
								"name": "Mail Storage",
								"description": "Email and attachments storage",
								"used": 2576980377,
								"limit": 32212254720,
								"scope": "account",
								"resourceType": "octets"
							}
						]
					}, "q0"]
				]
			}`,
			expectedQuotas: 1,
			wantErr:        false,
		},
		{
			name: "multiple quotas",
			sessionResp: `{
				"apiUrl": "API_URL",
				"accounts": {"test-account": {}},
				"capabilities": {
					"urn:ietf:params:jmap:core": {},
					"urn:ietf:params:jmap:quota": {}
				}
			}`,
			apiResp: `{
				"methodResponses": [
					["Quota/get", {
						"list": [
							{
								"id": "quota-1",
								"name": "Mail Storage",
								"used": 2576980377,
								"limit": 32212254720,
								"scope": "account",
								"resourceType": "octets"
							},
							{
								"id": "quota-2",
								"name": "Message Count",
								"used": 15234,
								"limit": 100000,
								"scope": "account",
								"resourceType": "count"
							}
						]
					}, "q0"]
				]
			}`,
			expectedQuotas: 2,
			wantErr:        false,
		},
		{
			name: "capability not available",
			sessionResp: `{
				"apiUrl": "API_URL",
				"accounts": {"test-account": {}},
				"capabilities": {
					"urn:ietf:params:jmap:core": {},
					"urn:ietf:params:jmap:mail": {}
				}
			}`,
			apiResp:        `{}`,
			wantErr:        true,
			expectedQuotas: 0,
		},
		{
			name: "error response from server",
			sessionResp: `{
				"apiUrl": "API_URL",
				"accounts": {"test-account": {}},
				"capabilities": {
					"urn:ietf:params:jmap:core": {},
					"urn:ietf:params:jmap:quota": {}
				}
			}`,
			apiResp: `{
				"methodResponses": [
					["error", {
						"type": "serverFail",
						"description": "Internal server error"
					}, "q0"]
				]
			}`,
			wantErr:        true,
			expectedQuotas: 0,
		},
		{
			name: "empty quota list",
			sessionResp: `{
				"apiUrl": "API_URL",
				"accounts": {"test-account": {}},
				"capabilities": {
					"urn:ietf:params:jmap:core": {},
					"urn:ietf:params:jmap:quota": {}
				}
			}`,
			apiResp: `{
				"methodResponses": [
					["Quota/get", {
						"list": []
					}, "q0"]
				]
			}`,
			expectedQuotas: 0,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.apiResp))
			}))
			defer apiServer.Close()

			sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resp := tt.sessionResp
				// Replace API_URL placeholder with actual server URL
				resp = strings.ReplaceAll(resp, "API_URL", apiServer.URL)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(resp))
			}))
			defer sessionServer.Close()

			client := NewClientWithBaseURL("test-token", sessionServer.URL)

			quotas, err := client.GetQuotas(context.Background())

			// Check error conditions
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check quota count
			if len(quotas) != tt.expectedQuotas {
				t.Errorf("expected %d quotas, got %d", tt.expectedQuotas, len(quotas))
			}

			// Validate quota fields if we got results
			if tt.expectedQuotas > 0 {
				quota := quotas[0]
				if quota.ID == "" {
					t.Error("quota ID should not be empty")
				}
				if quota.Name == "" {
					t.Error("quota name should not be empty")
				}
				if quota.Used < 0 {
					t.Error("quota used should not be negative")
				}
				if quota.Limit < 0 {
					t.Error("quota limit should not be negative")
				}
			}
		})
	}
}
