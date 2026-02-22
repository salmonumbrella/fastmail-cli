package jmap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetVacationResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     *VacationResponse
		wantErr  bool
	}{
		{
			name: "successful get with all fields",
			response: `{
				"methodResponses": [
					["VacationResponse/get", {
						"accountId": "acc123",
						"state": "state1",
						"list": [{
							"id": "singleton",
							"isEnabled": true,
							"fromDate": "2024-01-01T00:00:00Z",
							"toDate": "2024-12-31T23:59:59Z",
							"subject": "Out of Office",
							"textBody": "I'm on vacation",
							"htmlBody": "<p>I'm on vacation</p>"
						}]
					}, "getVacation"]
				]
			}`,
			want: &VacationResponse{
				ID:        "singleton",
				IsEnabled: true,
				FromDate:  "2024-01-01T00:00:00Z",
				ToDate:    "2024-12-31T23:59:59Z",
				Subject:   "Out of Office",
				TextBody:  "I'm on vacation",
				HTMLBody:  "<p>I'm on vacation</p>",
			},
			wantErr: false,
		},
		{
			name: "successful get with minimal fields",
			response: `{
				"methodResponses": [
					["VacationResponse/get", {
						"accountId": "acc123",
						"state": "state1",
						"list": [{
							"id": "singleton",
							"isEnabled": true
						}]
					}, "getVacation"]
				]
			}`,
			want: &VacationResponse{
				ID:        "singleton",
				IsEnabled: true,
			},
			wantErr: false,
		},
		{
			name: "disabled vacation response",
			response: `{
				"methodResponses": [
					["VacationResponse/get", {
						"accountId": "acc123",
						"state": "state1",
						"list": [{
							"id": "singleton",
							"isEnabled": false,
							"subject": "Old subject",
							"textBody": "Old text"
						}]
					}, "getVacation"]
				]
			}`,
			want: &VacationResponse{
				ID:        "singleton",
				IsEnabled: false,
				Subject:   "Old subject",
				TextBody:  "Old text",
			},
			wantErr: false,
		},
		{
			name: "empty list",
			response: `{
				"methodResponses": [
					["VacationResponse/get", {
						"accountId": "acc123",
						"state": "state1",
						"list": []
					}, "getVacation"]
				]
			}`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid response format - missing list",
			response: `{
				"methodResponses": [
					["VacationResponse/get", {
						"accountId": "acc123",
						"state": "state1"
					}, "getVacation"]
				]
			}`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid response format - not an array",
			response: `{
				"methodResponses": [
					["VacationResponse/get", {
						"accountId": "acc123",
						"state": "state1",
						"list": "not an array"
					}, "getVacation"]
				]
			}`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			defer apiServer.Close()

			sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{
					"apiUrl": "` + apiServer.URL + `",
					"uploadUrl": "` + apiServer.URL + `/{accountId}/",
					"downloadUrl": "` + apiServer.URL + `",
					"accounts": {"acc123": {}},
					"primaryAccounts": {"urn:ietf:params:jmap:mail": "acc123"}
				}`))
			}))
			defer sessionServer.Close()

			client := NewClientWithBaseURL("test-token", sessionServer.URL)

			got, err := client.GetVacationResponse(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if got.ID != tt.want.ID {
				t.Errorf("ID: got %v, want %v", got.ID, tt.want.ID)
			}
			if got.IsEnabled != tt.want.IsEnabled {
				t.Errorf("IsEnabled: got %v, want %v", got.IsEnabled, tt.want.IsEnabled)
			}
			if got.FromDate != tt.want.FromDate {
				t.Errorf("FromDate: got %v, want %v", got.FromDate, tt.want.FromDate)
			}
			if got.ToDate != tt.want.ToDate {
				t.Errorf("ToDate: got %v, want %v", got.ToDate, tt.want.ToDate)
			}
			if got.Subject != tt.want.Subject {
				t.Errorf("Subject: got %v, want %v", got.Subject, tt.want.Subject)
			}
			if got.TextBody != tt.want.TextBody {
				t.Errorf("TextBody: got %v, want %v", got.TextBody, tt.want.TextBody)
			}
			if got.HTMLBody != tt.want.HTMLBody {
				t.Errorf("HTMLBody: got %v, want %v", got.HTMLBody, tt.want.HTMLBody)
			}
		})
	}
}

func TestSetVacationResponse(t *testing.T) {
	tests := []struct {
		name        string
		opts        SetVacationResponseOpts
		wantUpdate  map[string]any
		serverError bool
		wantErr     bool
	}{
		{
			name: "enable vacation with all fields",
			opts: SetVacationResponseOpts{
				IsEnabled: true,
				FromDate:  "2024-06-01T00:00:00Z",
				ToDate:    "2024-06-30T23:59:59Z",
				Subject:   "On vacation",
				TextBody:  "I'm away",
				HTMLBody:  "<p>I'm away</p>",
			},
			wantUpdate: map[string]any{
				"isEnabled": true,
				"fromDate":  "2024-06-01T00:00:00Z",
				"toDate":    "2024-06-30T23:59:59Z",
				"subject":   "On vacation",
				"textBody":  "I'm away",
				"htmlBody":  "<p>I'm away</p>",
			},
			wantErr: false,
		},
		{
			name: "enable vacation with minimal fields",
			opts: SetVacationResponseOpts{
				IsEnabled: true,
				TextBody:  "Out of office",
			},
			wantUpdate: map[string]any{
				"isEnabled": true,
				"fromDate":  nil,
				"toDate":    nil,
				"textBody":  "Out of office",
			},
			wantErr: false,
		},
		{
			name: "disable vacation",
			opts: SetVacationResponseOpts{
				IsEnabled: false,
			},
			wantUpdate: map[string]any{
				"isEnabled": false,
				"fromDate":  nil,
				"toDate":    nil,
			},
			wantErr: false,
		},
		{
			name: "set dates only",
			opts: SetVacationResponseOpts{
				IsEnabled: true,
				FromDate:  "2024-12-20T00:00:00Z",
				ToDate:    "2025-01-05T23:59:59Z",
			},
			wantUpdate: map[string]any{
				"isEnabled": true,
				"fromDate":  "2024-12-20T00:00:00Z",
				"toDate":    "2025-01-05T23:59:59Z",
			},
			wantErr: false,
		},
		{
			name: "clear dates by passing empty strings",
			opts: SetVacationResponseOpts{
				IsEnabled: true,
				FromDate:  "",
				ToDate:    "",
				TextBody:  "Always on vacation",
			},
			wantUpdate: map[string]any{
				"isEnabled": true,
				"fromDate":  nil,
				"toDate":    nil,
				"textBody":  "Always on vacation",
			},
			wantErr: false,
		},
		{
			name: "server returns error",
			opts: SetVacationResponseOpts{
				IsEnabled: true,
			},
			serverError: true,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedUpdate map[string]any

			apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				// Parse request to inspect what was sent
				var req Request
				if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
					if len(req.MethodCalls) > 0 {
						// Check if this is a Set request
						if req.MethodCalls[0][0] == "VacationResponse/set" {
							if args, ok := req.MethodCalls[0][1].(map[string]any); ok {
								if updateMap, ok := args["update"].(map[string]any); ok {
									if update, ok := updateMap["singleton"].(map[string]any); ok {
										receivedUpdate = update
									}
								}
							}
						}
					}
				}

				// Handle Set response
				if len(req.MethodCalls) > 0 && req.MethodCalls[0][0] == "VacationResponse/set" {
					if tt.serverError {
						_, _ = w.Write([]byte(`{
							"methodResponses": [
								["VacationResponse/set", {
									"accountId": "acc123",
									"notUpdated": {
										"singleton": {
											"type": "serverError",
											"description": "Server error occurred"
										}
									}
								}, "setVacation"]
							]
						}`))
					} else {
						_, _ = w.Write([]byte(`{
							"methodResponses": [
								["VacationResponse/set", {
									"accountId": "acc123",
									"updated": {
										"singleton": {}
									}
								}, "setVacation"]
							]
						}`))
					}
					return
				}

				// Handle Get response (needed for SetVacationResponse to get current ID)
				_, _ = w.Write([]byte(`{
					"methodResponses": [
						["VacationResponse/get", {
							"accountId": "acc123",
							"state": "state1",
							"list": [{
								"id": "singleton",
								"isEnabled": false
							}]
						}, "getVacation"]
					]
				}`))
			}))
			defer apiServer.Close()

			sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{
					"apiUrl": "` + apiServer.URL + `",
					"uploadUrl": "` + apiServer.URL + `/{accountId}/",
					"downloadUrl": "` + apiServer.URL + `",
					"accounts": {"acc123": {}},
					"primaryAccounts": {"urn:ietf:params:jmap:mail": "acc123"}
				}`))
			}))
			defer sessionServer.Close()

			client := NewClientWithBaseURL("test-token", sessionServer.URL)

			err := client.SetVacationResponse(context.Background(), tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify the update payload sent to the server
			if tt.wantUpdate != nil && receivedUpdate != nil {
				for key, wantVal := range tt.wantUpdate {
					gotVal, exists := receivedUpdate[key]
					if !exists {
						// Only check if the field was expected
						if wantVal != nil && wantVal != "" {
							t.Errorf("missing field %s in update", key)
						}
						continue
					}
					if gotVal != wantVal {
						t.Errorf("field %s: got %v, want %v", key, gotVal, wantVal)
					}
				}
			}
		})
	}
}

func TestDisableVacationResponse(t *testing.T) {
	var receivedIsEnabled *bool

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Parse request to capture isEnabled value
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			if len(req.MethodCalls) > 0 {
				// Check if this is a Set request
				if req.MethodCalls[0][0] == "VacationResponse/set" {
					if args, ok := req.MethodCalls[0][1].(map[string]any); ok {
						if updateMap, ok := args["update"].(map[string]any); ok {
							if update, ok := updateMap["singleton"].(map[string]any); ok {
								if isEnabled, ok := update["isEnabled"].(bool); ok {
									receivedIsEnabled = &isEnabled
								}
							}
						}
					}
				}
			}
		}

		// Handle Set response
		if len(req.MethodCalls) > 0 && req.MethodCalls[0][0] == "VacationResponse/set" {
			_, _ = w.Write([]byte(`{
				"methodResponses": [
					["VacationResponse/set", {
						"accountId": "acc123",
						"updated": {
							"singleton": {}
						}
					}, "setVacation"]
				]
			}`))
			return
		}

		// Handle Get response (needed for SetVacationResponse to get current ID)
		_, _ = w.Write([]byte(`{
			"methodResponses": [
				["VacationResponse/get", {
					"accountId": "acc123",
					"state": "state1",
					"list": [{
						"id": "singleton",
						"isEnabled": true,
						"subject": "Old vacation",
						"textBody": "Old message"
					}]
				}, "getVacation"]
			]
		}`))
	}))
	defer apiServer.Close()

	sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"apiUrl": "` + apiServer.URL + `",
			"uploadUrl": "` + apiServer.URL + `/{accountId}/",
			"downloadUrl": "` + apiServer.URL + `",
			"accounts": {"acc123": {}},
			"primaryAccounts": {"urn:ietf:params:jmap:mail": "acc123"}
		}`))
	}))
	defer sessionServer.Close()

	client := NewClientWithBaseURL("test-token", sessionServer.URL)

	err := client.DisableVacationResponse(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Verify that isEnabled=false was sent
	if receivedIsEnabled == nil {
		t.Error("isEnabled field not found in request")
	} else if *receivedIsEnabled != false {
		t.Errorf("isEnabled: got %v, want false", *receivedIsEnabled)
	}
}

func TestSetVacationResponse_GetError(t *testing.T) {
	// Test that SetVacationResponse returns error when GetVacationResponse fails
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty list to cause GetVacationResponse to fail
		_, _ = w.Write([]byte(`{
			"methodResponses": [
				["VacationResponse/get", {
					"accountId": "acc123",
					"state": "state1",
					"list": []
				}, "getVacation"]
			]
		}`))
	}))
	defer apiServer.Close()

	sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"apiUrl": "` + apiServer.URL + `",
			"uploadUrl": "` + apiServer.URL + `/{accountId}/",
			"downloadUrl": "` + apiServer.URL + `",
			"accounts": {"acc123": {}},
			"primaryAccounts": {"urn:ietf:params:jmap:mail": "acc123"}
		}`))
	}))
	defer sessionServer.Close()

	client := NewClientWithBaseURL("test-token", sessionServer.URL)

	err := client.SetVacationResponse(context.Background(), SetVacationResponseOpts{
		IsEnabled: true,
		TextBody:  "Test",
	})

	if err == nil {
		t.Error("expected error when GetVacationResponse fails, but got none")
	}
}
