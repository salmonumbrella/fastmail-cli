package jmap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateMailboxOpts(t *testing.T) {
	opts := CreateMailboxOpts{
		Name:     "Projects",
		ParentID: "parent-123",
	}

	if opts.Name != "Projects" {
		t.Errorf("expected name Projects, got %s", opts.Name)
	}
	if opts.ParentID != "parent-123" {
		t.Errorf("expected parentID parent-123, got %s", opts.ParentID)
	}
}

func TestCreateMailbox(t *testing.T) {
	tests := []struct {
		name     string
		opts     CreateMailboxOpts
		response string
		wantID   string
		wantName string
		wantErr  bool
	}{
		{
			name: "successful create",
			opts: CreateMailboxOpts{
				Name: "Work",
			},
			response: `{
				"methodResponses": [
					["Mailbox/set", {
						"accountId": "acc123",
						"created": {
							"new": {
								"id": "mbox123"
							}
						}
					}, "createMailbox"]
				]
			}`,
			wantID:   "mbox123",
			wantName: "Work",
			wantErr:  false,
		},
		{
			name: "create with parent",
			opts: CreateMailboxOpts{
				Name:     "Urgent",
				ParentID: "parent456",
			},
			response: `{
				"methodResponses": [
					["Mailbox/set", {
						"accountId": "acc123",
						"created": {
							"new": {
								"id": "mbox789"
							}
						}
					}, "createMailbox"]
				]
			}`,
			wantID:   "mbox789",
			wantName: "Urgent",
			wantErr:  false,
		},
		{
			name: "server error",
			opts: CreateMailboxOpts{
				Name: "Test",
			},
			response: `{
				"methodResponses": [
					["Mailbox/set", {
						"accountId": "acc123",
						"notCreated": {
							"new": {
								"type": "invalidArguments",
								"description": "Invalid mailbox name"
							}
						}
					}, "createMailbox"]
				]
			}`,
			wantErr: true,
		},
		{
			name: "empty name",
			opts: CreateMailboxOpts{
				Name: "",
			},
			wantErr: true,
		},
		{
			name: "created but no ID returned",
			opts: CreateMailboxOpts{
				Name: "NoID",
			},
			response: `{
				"methodResponses": [
					["Mailbox/set", {
						"accountId": "acc123",
						"created": {}
					}, "createMailbox"]
				]
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip server setup for validation-only tests
			if tt.opts.Name == "" {
				client := NewClientWithBaseURL("test-token", "http://localhost")
				_, err := client.CreateMailbox(context.Background(), tt.opts)
				if err == nil {
					t.Error("expected error for empty name but got none")
				}
				return
			}

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

			got, err := client.CreateMailbox(context.Background(), tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if got.ID != tt.wantID {
				t.Errorf("ID: got %v, want %v", got.ID, tt.wantID)
			}
			if got.Name != tt.wantName {
				t.Errorf("Name: got %v, want %v", got.Name, tt.wantName)
			}
		})
	}
}

func TestDeleteMailbox(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		response string
		wantErr  bool
	}{
		{
			name: "successful delete",
			id:   "mbox123",
			response: `{
				"methodResponses": [
					["Mailbox/set", {
						"accountId": "acc123",
						"destroyed": ["mbox123"]
					}, "deleteMailbox"]
				]
			}`,
			wantErr: false,
		},
		{
			name: "mailbox not found",
			id:   "mbox999",
			response: `{
				"methodResponses": [
					["Mailbox/set", {
						"accountId": "acc123",
						"notDestroyed": {
							"mbox999": {
								"type": "notFound",
								"description": "Mailbox not found"
							}
						}
					}, "deleteMailbox"]
				]
			}`,
			wantErr: true,
		},
		{
			name: "mailbox has child mailboxes",
			id:   "mboxParent",
			response: `{
				"methodResponses": [
					["Mailbox/set", {
						"accountId": "acc123",
						"notDestroyed": {
							"mboxParent": {
								"type": "mailboxHasChild",
								"description": "Mailbox has child mailboxes"
							}
						}
					}, "deleteMailbox"]
				]
			}`,
			wantErr: true,
		},
		{
			name: "server error",
			id:   "mboxErr",
			response: `{
				"methodResponses": [
					["Mailbox/set", {
						"accountId": "acc123",
						"notDestroyed": {
							"mboxErr": {
								"type": "serverError",
								"description": "Internal server error"
							}
						}
					}, "deleteMailbox"]
				]
			}`,
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

			err := client.DeleteMailbox(context.Background(), tt.id)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
		})
	}
}

func TestRenameMailbox(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		newName  string
		response string
		wantErr  bool
	}{
		{
			name:    "successful rename",
			id:      "mbox123",
			newName: "Projects Renamed",
			response: `{
				"methodResponses": [
					["Mailbox/set", {
						"accountId": "acc123",
						"updated": {
							"mbox123": {}
						}
					}, "renameMailbox"]
				]
			}`,
			wantErr: false,
		},
		{
			name:    "mailbox not found",
			id:      "mbox999",
			newName: "New Name",
			response: `{
				"methodResponses": [
					["Mailbox/set", {
						"accountId": "acc123",
						"notUpdated": {
							"mbox999": {
								"type": "notFound",
								"description": "Mailbox not found"
							}
						}
					}, "renameMailbox"]
				]
			}`,
			wantErr: true,
		},
		{
			name:    "duplicate name",
			id:      "mbox456",
			newName: "Inbox",
			response: `{
				"methodResponses": [
					["Mailbox/set", {
						"accountId": "acc123",
						"notUpdated": {
							"mbox456": {
								"type": "invalidArguments",
								"description": "Mailbox name already exists"
							}
						}
					}, "renameMailbox"]
				]
			}`,
			wantErr: true,
		},
		{
			name:    "empty name",
			id:      "mbox123",
			newName: "",
			wantErr: true,
		},
		{
			name:    "server error",
			id:      "mboxErr",
			newName: "Error Test",
			response: `{
				"methodResponses": [
					["Mailbox/set", {
						"accountId": "acc123",
						"notUpdated": {
							"mboxErr": {
								"type": "serverError",
								"description": "Internal server error"
							}
						}
					}, "renameMailbox"]
				]
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip server setup for validation-only tests
			if tt.newName == "" {
				client := NewClientWithBaseURL("test-token", "http://localhost")
				err := client.RenameMailbox(context.Background(), tt.id, tt.newName)
				if err == nil {
					t.Error("expected error for empty name but got none")
				}
				return
			}

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

			err := client.RenameMailbox(context.Background(), tt.id, tt.newName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
		})
	}
}
