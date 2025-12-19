package jmap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetAddressBooks(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		wantErr      bool
		wantErrMsg   string
		wantNumBooks int
	}{
		{
			name: "successful get with address books",
			response: `{
				"methodResponses": [
					["AddressBook/get", {
						"accountId": "acc123",
						"state": "state1",
						"list": [
							{
								"id": "ab1",
								"name": "Personal",
								"isDefault": true,
								"isSubscribed": true
							},
							{
								"id": "ab2",
								"name": "Work",
								"isDefault": false,
								"isSubscribed": true
							}
						]
					}, "0"]
				]
			}`,
			wantErr:      false,
			wantNumBooks: 2,
		},
		{
			name: "empty address book list",
			response: `{
				"methodResponses": [
					["AddressBook/get", {
						"accountId": "acc123",
						"state": "state1",
						"list": []
					}, "0"]
				]
			}`,
			wantErr:      false,
			wantNumBooks: 0,
		},
		{
			name: "error response",
			response: `{
				"methodResponses": [
					["error", {
						"type": "accountNotFound"
					}, "0"]
				]
			}`,
			wantErr:    true,
			wantErrMsg: "API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.response))
			}))
			defer apiServer.Close()

			sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{
					"apiUrl": "` + apiServer.URL + `",
					"downloadUrl": "` + apiServer.URL + `",
					"accounts": {"acc123": {}},
					"primaryAccounts": {
						"urn:ietf:params:jmap:mail": "acc123",
						"urn:ietf:params:jmap:contacts": "acc123"
					},
					"capabilities": {
						"urn:ietf:params:jmap:core": {},
						"urn:ietf:params:jmap:contacts": {}
					}
				}`))
			}))
			defer sessionServer.Close()

			client := NewClient("test-token")
			client.baseURL = sessionServer.URL

			addressBooks, err := client.GetAddressBooks(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetAddressBooks() expected error, got nil")
				}
				if tt.wantErrMsg != "" && err != nil {
					if err.Error() != tt.wantErrMsg && !contains(err.Error(), tt.wantErrMsg) {
						t.Errorf("GetAddressBooks() error = %v, want error containing %v", err, tt.wantErrMsg)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("GetAddressBooks() unexpected error = %v", err)
				return
			}

			if len(addressBooks) != tt.wantNumBooks {
				t.Errorf("GetAddressBooks() got %d address books, want %d", len(addressBooks), tt.wantNumBooks)
			}
		})
	}
}

func TestGetContacts(t *testing.T) {
	tests := []struct {
		name            string
		response        string
		addressBookID   string
		limit           int
		wantErr         bool
		wantErrMsg      string
		wantNumContacts int
	}{
		{
			name:          "successful get with contacts",
			addressBookID: "",
			limit:         100,
			response: `{
				"methodResponses": [
					["ContactCard/query", {
						"accountId": "acc123",
						"ids": ["c1", "c2"]
					}, "0"],
					["ContactCard/get", {
						"accountId": "acc123",
						"state": "state1",
						"list": [
							{
								"id": "c1",
								"name": "John Doe",
								"emails": [{"type": "work", "value": "john@example.com"}],
								"company": "Acme Corp",
								"updated": "2024-01-01T00:00:00Z"
							},
							{
								"id": "c2",
								"name": "Jane Smith",
								"emails": [{"type": "home", "value": "jane@example.com"}],
								"updated": "2024-01-02T00:00:00Z"
							}
						]
					}, "1"]
				]
			}`,
			wantErr:         false,
			wantNumContacts: 2,
		},
		{
			name:          "empty contacts list",
			addressBookID: "",
			limit:         100,
			response: `{
				"methodResponses": [
					["ContactCard/query", {
						"accountId": "acc123",
						"ids": []
					}, "0"],
					["ContactCard/get", {
						"accountId": "acc123",
						"state": "state1",
						"list": []
					}, "1"]
				]
			}`,
			wantErr:         false,
			wantNumContacts: 0,
		},
		{
			name:          "error response",
			addressBookID: "",
			limit:         100,
			response: `{
				"methodResponses": [
					["ContactCard/query", {
						"accountId": "acc123",
						"ids": []
					}, "0"],
					["error", {
						"type": "accountNotFound"
					}, "1"]
				]
			}`,
			wantErr:    true,
			wantErrMsg: "API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.response))
			}))
			defer apiServer.Close()

			sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{
					"apiUrl": "` + apiServer.URL + `",
					"downloadUrl": "` + apiServer.URL + `",
					"accounts": {"acc123": {}},
					"primaryAccounts": {
						"urn:ietf:params:jmap:mail": "acc123",
						"urn:ietf:params:jmap:contacts": "acc123"
					},
					"capabilities": {
						"urn:ietf:params:jmap:core": {},
						"urn:ietf:params:jmap:contacts": {}
					}
				}`))
			}))
			defer sessionServer.Close()

			client := NewClient("test-token")
			client.baseURL = sessionServer.URL

			contacts, err := client.GetContacts(context.Background(), tt.addressBookID, tt.limit)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetContacts() expected error, got nil")
				}
				if tt.wantErrMsg != "" && err != nil {
					if !contains(err.Error(), tt.wantErrMsg) {
						t.Errorf("GetContacts() error = %v, want error containing %v", err, tt.wantErrMsg)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("GetContacts() unexpected error = %v", err)
				return
			}

			if len(contacts) != tt.wantNumContacts {
				t.Errorf("GetContacts() got %d contacts, want %d", len(contacts), tt.wantNumContacts)
			}
		})
	}
}

func TestGetContactByID(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		contactID  string
		wantErr    bool
		wantErrMsg string
		wantName   string
	}{
		{
			name:      "successful get",
			contactID: "c1",
			response: `{
				"methodResponses": [
					["ContactCard/get", {
						"accountId": "acc123",
						"state": "state1",
						"list": [{
							"id": "c1",
							"name": "John Doe",
							"emails": [{"type": "work", "value": "john@example.com"}],
							"company": "Acme Corp",
							"updated": "2024-01-01T00:00:00Z"
						}]
					}, "0"]
				]
			}`,
			wantErr:  false,
			wantName: "John Doe",
		},
		{
			name:      "contact not found",
			contactID: "c999",
			response: `{
				"methodResponses": [
					["ContactCard/get", {
						"accountId": "acc123",
						"state": "state1",
						"list": []
					}, "0"]
				]
			}`,
			wantErr:    true,
			wantErrMsg: ErrContactNotFound.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.response))
			}))
			defer apiServer.Close()

			sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{
					"apiUrl": "` + apiServer.URL + `",
					"downloadUrl": "` + apiServer.URL + `",
					"accounts": {"acc123": {}},
					"primaryAccounts": {
						"urn:ietf:params:jmap:mail": "acc123",
						"urn:ietf:params:jmap:contacts": "acc123"
					},
					"capabilities": {
						"urn:ietf:params:jmap:core": {},
						"urn:ietf:params:jmap:contacts": {}
					}
				}`))
			}))
			defer sessionServer.Close()

			client := NewClient("test-token")
			client.baseURL = sessionServer.URL

			contact, err := client.GetContactByID(context.Background(), tt.contactID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetContactByID() expected error, got nil")
				}
				if tt.wantErrMsg != "" && err != nil {
					if !contains(err.Error(), tt.wantErrMsg) {
						t.Errorf("GetContactByID() error = %v, want error containing %v", err, tt.wantErrMsg)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("GetContactByID() unexpected error = %v", err)
				return
			}

			if contact.Name != tt.wantName {
				t.Errorf("GetContactByID() got name %s, want %s", contact.Name, tt.wantName)
			}
		})
	}
}

func TestContactsNotEnabled(t *testing.T) {
	// Test that ErrContactsNotEnabled is returned when capability is missing
	sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"apiUrl": "http://localhost",
			"downloadUrl": "http://localhost",
			"accounts": {"acc123": {}},
			"primaryAccounts": {
				"urn:ietf:params:jmap:mail": "acc123"
			},
			"capabilities": {
				"urn:ietf:params:jmap:core": {}
			}
		}`))
	}))
	defer sessionServer.Close()

	client := NewClient("test-token")
	client.baseURL = sessionServer.URL

	_, err := client.GetAddressBooks(context.Background())
	if err != ErrContactsNotEnabled {
		t.Errorf("GetAddressBooks() expected ErrContactsNotEnabled, got %v", err)
	}

	_, err = client.GetContacts(context.Background(), "", 100)
	if err != ErrContactsNotEnabled {
		t.Errorf("GetContacts() expected ErrContactsNotEnabled, got %v", err)
	}

	_, err = client.GetContactByID(context.Background(), "c1")
	if err != ErrContactsNotEnabled {
		t.Errorf("GetContactByID() expected ErrContactsNotEnabled, got %v", err)
	}

	_, err = client.SearchContacts(context.Background(), "test", 50)
	if err != ErrContactsNotEnabled {
		t.Errorf("SearchContacts() expected ErrContactsNotEnabled, got %v", err)
	}
}

func TestCreateContact(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse the request to verify it contains expected data
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"methodResponses": [
				["ContactCard/set", {
					"accountId": "acc123",
					"created": {
						"new-contact": {
							"id": "c1",
							"name": "John Doe",
							"emails": [{"type": "work", "value": "john@example.com"}],
							"updated": "2024-01-01T00:00:00Z"
						}
					}
				}, "0"]
			]
		}`))
	}))
	defer apiServer.Close()

	sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"apiUrl": "` + apiServer.URL + `",
			"downloadUrl": "` + apiServer.URL + `",
			"accounts": {"acc123": {}},
			"primaryAccounts": {
				"urn:ietf:params:jmap:mail": "acc123",
				"urn:ietf:params:jmap:contacts": "acc123"
			},
			"capabilities": {
				"urn:ietf:params:jmap:core": {},
				"urn:ietf:params:jmap:contacts": {}
			}
		}`))
	}))
	defer sessionServer.Close()

	client := NewClient("test-token")
	client.baseURL = sessionServer.URL

	contact := &Contact{
		Name: "John Doe",
		Emails: []ContactEmail{
			{Type: "work", Value: "john@example.com"},
		},
	}

	created, err := client.CreateContact(context.Background(), contact)
	if err != nil {
		t.Errorf("CreateContact() unexpected error = %v", err)
		return
	}

	if created.ID != "c1" {
		t.Errorf("CreateContact() got ID %s, want c1", created.ID)
	}

	if created.Name != "John Doe" {
		t.Errorf("CreateContact() got name %s, want John Doe", created.Name)
	}
}

func TestDeleteContact(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"methodResponses": [
				["ContactCard/set", {
					"accountId": "acc123",
					"destroyed": ["c1"]
				}, "0"]
			]
		}`))
	}))
	defer apiServer.Close()

	sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"apiUrl": "` + apiServer.URL + `",
			"downloadUrl": "` + apiServer.URL + `",
			"accounts": {"acc123": {}},
			"primaryAccounts": {
				"urn:ietf:params:jmap:mail": "acc123",
				"urn:ietf:params:jmap:contacts": "acc123"
			},
			"capabilities": {
				"urn:ietf:params:jmap:core": {},
				"urn:ietf:params:jmap:contacts": {}
			}
		}`))
	}))
	defer sessionServer.Close()

	client := NewClient("test-token")
	client.baseURL = sessionServer.URL

	err := client.DeleteContact(context.Background(), "c1")
	if err != nil {
		t.Errorf("DeleteContact() unexpected error = %v", err)
	}
}

func TestUpdateContact(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		contactID  string
		updates    map[string]interface{}
		wantErr    bool
		wantErrMsg string
		wantName   string
	}{
		{
			name:      "successful update",
			contactID: "c1",
			updates: map[string]interface{}{
				"name": "Jane Updated",
			},
			response: `{
				"methodResponses": [
					["ContactCard/set", {
						"accountId": "acc123",
						"updated": {
							"c1": {
								"id": "c1",
								"name": "Jane Updated",
								"emails": [{"type": "work", "value": "jane@example.com"}],
								"updated": "2024-01-02T00:00:00Z"
							}
						}
					}, "0"]
				]
			}`,
			wantErr:  false,
			wantName: "Jane Updated",
		},
		{
			name:      "update error",
			contactID: "c999",
			updates: map[string]interface{}{
				"name": "Should Fail",
			},
			response: `{
				"methodResponses": [
					["error", {
						"type": "notFound"
					}, "0"]
				]
			}`,
			wantErr:    true,
			wantErrMsg: "API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.response))
			}))
			defer apiServer.Close()

			sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{
					"apiUrl": "` + apiServer.URL + `",
					"downloadUrl": "` + apiServer.URL + `",
					"accounts": {"acc123": {}},
					"primaryAccounts": {
						"urn:ietf:params:jmap:mail": "acc123",
						"urn:ietf:params:jmap:contacts": "acc123"
					},
					"capabilities": {
						"urn:ietf:params:jmap:core": {},
						"urn:ietf:params:jmap:contacts": {}
					}
				}`))
			}))
			defer sessionServer.Close()

			client := NewClient("test-token")
			client.baseURL = sessionServer.URL

			contact, err := client.UpdateContact(context.Background(), tt.contactID, tt.updates)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateContact() expected error, got nil")
				}
				if tt.wantErrMsg != "" && err != nil {
					if !contains(err.Error(), tt.wantErrMsg) {
						t.Errorf("UpdateContact() error = %v, want error containing %v", err, tt.wantErrMsg)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("UpdateContact() unexpected error = %v", err)
				return
			}

			if contact.Name != tt.wantName {
				t.Errorf("UpdateContact() got name %s, want %s", contact.Name, tt.wantName)
			}
		})
	}
}

func TestSearchContacts(t *testing.T) {
	tests := []struct {
		name            string
		response        string
		query           string
		limit           int
		wantErr         bool
		wantErrMsg      string
		wantNumContacts int
	}{
		{
			name:  "successful search with results",
			query: "john",
			limit: 50,
			response: `{
				"methodResponses": [
					["ContactCard/query", {
						"accountId": "acc123",
						"ids": ["c1"]
					}, "0"],
					["ContactCard/get", {
						"accountId": "acc123",
						"state": "state1",
						"list": [
							{
								"id": "c1",
								"name": "John Doe",
								"emails": [{"type": "work", "value": "john@example.com"}],
								"updated": "2024-01-01T00:00:00Z"
							}
						]
					}, "1"]
				]
			}`,
			wantErr:         false,
			wantNumContacts: 1,
		},
		{
			name:  "search with no results",
			query: "nonexistent",
			limit: 50,
			response: `{
				"methodResponses": [
					["ContactCard/query", {
						"accountId": "acc123",
						"ids": []
					}, "0"],
					["ContactCard/get", {
						"accountId": "acc123",
						"state": "state1",
						"list": []
					}, "1"]
				]
			}`,
			wantErr:         false,
			wantNumContacts: 0,
		},
		{
			name:  "search error",
			query: "test",
			limit: 50,
			response: `{
				"methodResponses": [
					["ContactCard/query", {
						"accountId": "acc123",
						"ids": []
					}, "0"],
					["error", {
						"type": "serverError"
					}, "1"]
				]
			}`,
			wantErr:    true,
			wantErrMsg: "API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.response))
			}))
			defer apiServer.Close()

			sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{
					"apiUrl": "` + apiServer.URL + `",
					"downloadUrl": "` + apiServer.URL + `",
					"accounts": {"acc123": {}},
					"primaryAccounts": {
						"urn:ietf:params:jmap:mail": "acc123",
						"urn:ietf:params:jmap:contacts": "acc123"
					},
					"capabilities": {
						"urn:ietf:params:jmap:core": {},
						"urn:ietf:params:jmap:contacts": {}
					}
				}`))
			}))
			defer sessionServer.Close()

			client := NewClient("test-token")
			client.baseURL = sessionServer.URL

			contacts, err := client.SearchContacts(context.Background(), tt.query, tt.limit)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SearchContacts() expected error, got nil")
				}
				if tt.wantErrMsg != "" && err != nil {
					if !contains(err.Error(), tt.wantErrMsg) {
						t.Errorf("SearchContacts() error = %v, want error containing %v", err, tt.wantErrMsg)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("SearchContacts() unexpected error = %v", err)
				return
			}

			if len(contacts) != tt.wantNumContacts {
				t.Errorf("SearchContacts() got %d contacts, want %d", len(contacts), tt.wantNumContacts)
			}
		})
	}
}

// Helper function to check if a string contains another string
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Suppress unused warning for time import
var _ = time.Now
