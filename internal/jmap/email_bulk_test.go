package jmap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestDeleteEmails_Multiple(t *testing.T) {
	// Response for successful deletion of 3 emails
	response := `{
		"methodResponses": [
			["Email/set", {
				"accountId": "acc123",
				"updated": {
					"email1": {},
					"email2": {},
					"email3": {}
				}
			}, "moveToTrash"]
		]
	}`

	// Mock mailboxes response
	mailboxesResponse := `{
		"methodResponses": [
			["Mailbox/get", {
				"accountId": "acc123",
				"list": [
					{
						"id": "trash-123",
						"name": "Trash",
						"role": "trash",
						"totalEmails": 0,
						"unreadEmails": 0
					}
				]
			}, "mailboxes"]
		]
	}`

	var requestCount int
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		requestCount++
		if requestCount == 1 {
			// First request: GetMailboxes
			_, _ = w.Write([]byte(mailboxesResponse))
		} else {
			// Second request: Email/set
			_, _ = w.Write([]byte(response))
		}
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

	ids := []string{"email1", "email2", "email3"}
	result, err := client.DeleteEmails(context.Background(), ids)

	if err != nil {
		t.Fatalf("DeleteEmails() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("DeleteEmails() returned nil result")
	}

	// All should succeed
	if len(result.Succeeded) != 3 {
		t.Errorf("DeleteEmails() succeeded count = %d, want 3", len(result.Succeeded))
	}

	// Check all IDs are in succeeded
	expectedIDs := map[string]bool{"email1": true, "email2": true, "email3": true}
	for _, id := range result.Succeeded {
		if !expectedIDs[id] {
			t.Errorf("DeleteEmails() unexpected succeeded ID: %s", id)
		}
	}

	// No failures
	if len(result.Failed) != 0 {
		t.Errorf("DeleteEmails() failed count = %d, want 0", len(result.Failed))
	}
}

func TestDeleteEmails_PartialFailure(t *testing.T) {
	// Response with some successes and some failures
	response := `{
		"methodResponses": [
			["Email/set", {
				"accountId": "acc123",
				"updated": {
					"email1": {},
					"email3": {}
				},
				"notUpdated": {
					"email2": {
						"type": "notFound",
						"description": "Email not found"
					}
				}
			}, "moveToTrash"]
		]
	}`

	mailboxesResponse := `{
		"methodResponses": [
			["Mailbox/get", {
				"accountId": "acc123",
				"list": [
					{
						"id": "trash-123",
						"name": "Trash",
						"role": "trash",
						"totalEmails": 0,
						"unreadEmails": 0
					}
				]
			}, "mailboxes"]
		]
	}`

	var requestCount int
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		requestCount++
		if requestCount == 1 {
			_, _ = w.Write([]byte(mailboxesResponse))
		} else {
			_, _ = w.Write([]byte(response))
		}
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

	ids := []string{"email1", "email2", "email3"}
	result, err := client.DeleteEmails(context.Background(), ids)

	if err != nil {
		t.Fatalf("DeleteEmails() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("DeleteEmails() returned nil result")
	}

	// 2 should succeed
	if len(result.Succeeded) != 2 {
		t.Errorf("DeleteEmails() succeeded count = %d, want 2", len(result.Succeeded))
	}

	// Check succeeded IDs
	expectedSucceeded := map[string]bool{"email1": true, "email3": true}
	for _, id := range result.Succeeded {
		if !expectedSucceeded[id] {
			t.Errorf("DeleteEmails() unexpected succeeded ID: %s", id)
		}
	}

	// 1 should fail
	if len(result.Failed) != 1 {
		t.Errorf("DeleteEmails() failed count = %d, want 1", len(result.Failed))
	}

	// Check failed ID and error message
	if errMsg, exists := result.Failed["email2"]; !exists {
		t.Errorf("DeleteEmails() email2 should be in failed map")
	} else if errMsg == "" {
		t.Errorf("DeleteEmails() email2 should have error message")
	}
}

func TestDeleteEmails_EmptyInput(t *testing.T) {
	client := NewClientWithBaseURL("test-token", "http://dummy")

	result, err := client.DeleteEmails(context.Background(), []string{})

	if err != nil {
		t.Fatalf("DeleteEmails() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("DeleteEmails() returned nil result")
	}

	// Empty input should return empty result
	if len(result.Succeeded) != 0 {
		t.Errorf("DeleteEmails() succeeded count = %d, want 0", len(result.Succeeded))
	}

	if len(result.Failed) != 0 {
		t.Errorf("DeleteEmails() failed count = %d, want 0", len(result.Failed))
	}
}

func TestDeleteEmails_NilInput(t *testing.T) {
	client := NewClientWithBaseURL("test-token", "http://dummy")

	result, err := client.DeleteEmails(context.Background(), nil)

	if err != nil {
		t.Fatalf("DeleteEmails() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("DeleteEmails() returned nil result")
	}

	// Nil input should return empty result
	if len(result.Succeeded) != 0 {
		t.Errorf("DeleteEmails() succeeded count = %d, want 0", len(result.Succeeded))
	}

	if len(result.Failed) != 0 {
		t.Errorf("DeleteEmails() failed count = %d, want 0", len(result.Failed))
	}
}

func TestDeleteEmails_NoTrashMailbox(t *testing.T) {
	// Mailboxes response without trash
	mailboxesResponse := `{
		"methodResponses": [
			["Mailbox/get", {
				"accountId": "acc123",
				"list": [
					{
						"id": "inbox-123",
						"name": "Inbox",
						"role": "inbox",
						"totalEmails": 10,
						"unreadEmails": 5
					}
				]
			}, "mailboxes"]
		]
	}`

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mailboxesResponse))
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

	ids := []string{"email1", "email2"}
	_, err := client.DeleteEmails(context.Background(), ids)

	if err != ErrNoTrashMailbox {
		t.Errorf("DeleteEmails() error = %v, want %v", err, ErrNoTrashMailbox)
	}
}

func TestDeleteEmails_AllFailed(t *testing.T) {
	// Response where all emails fail to delete
	response := `{
		"methodResponses": [
			["Email/set", {
				"accountId": "acc123",
				"notUpdated": {
					"email1": {
						"type": "notFound",
						"description": "Email not found"
					},
					"email2": {
						"type": "forbidden",
						"description": "Cannot delete this email"
					}
				}
			}, "moveToTrash"]
		]
	}`

	mailboxesResponse := `{
		"methodResponses": [
			["Mailbox/get", {
				"accountId": "acc123",
				"list": [
					{
						"id": "trash-123",
						"name": "Trash",
						"role": "trash",
						"totalEmails": 0,
						"unreadEmails": 0
					}
				]
			}, "mailboxes"]
		]
	}`

	var requestCount int
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		requestCount++
		if requestCount == 1 {
			_, _ = w.Write([]byte(mailboxesResponse))
		} else {
			_, _ = w.Write([]byte(response))
		}
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

	ids := []string{"email1", "email2"}
	result, err := client.DeleteEmails(context.Background(), ids)

	if err != nil {
		t.Fatalf("DeleteEmails() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("DeleteEmails() returned nil result")
	}

	// None should succeed
	if len(result.Succeeded) != 0 {
		t.Errorf("DeleteEmails() succeeded count = %d, want 0", len(result.Succeeded))
	}

	// All should fail
	if len(result.Failed) != 2 {
		t.Errorf("DeleteEmails() failed count = %d, want 2", len(result.Failed))
	}

	// Check both failed
	if _, exists := result.Failed["email1"]; !exists {
		t.Errorf("DeleteEmails() email1 should be in failed map")
	}
	if _, exists := result.Failed["email2"]; !exists {
		t.Errorf("DeleteEmails() email2 should be in failed map")
	}
}

func TestParseBulkUpdateResult(t *testing.T) {
	tests := []struct {
		name     string
		result   map[string]any
		wantSucc []string
		wantFail map[string]string
	}{
		{
			name: "all succeeded",
			result: map[string]any{
				"updated": map[string]any{
					"id1": map[string]any{},
					"id2": map[string]any{},
				},
			},
			wantSucc: []string{"id1", "id2"},
			wantFail: map[string]string{},
		},
		{
			name: "all failed",
			result: map[string]any{
				"notUpdated": map[string]any{
					"id1": map[string]any{
						"type":        "notFound",
						"description": "Not found",
					},
					"id2": map[string]any{
						"type":        "forbidden",
						"description": "Access denied",
					},
				},
			},
			wantSucc: []string{},
			wantFail: map[string]string{
				"id1": "notFound: Not found",
				"id2": "forbidden: Access denied",
			},
		},
		{
			name: "mixed success and failure",
			result: map[string]any{
				"updated": map[string]any{
					"id1": map[string]any{},
					"id3": map[string]any{},
				},
				"notUpdated": map[string]any{
					"id2": map[string]any{
						"type":        "notFound",
						"description": "Email not found",
					},
				},
			},
			wantSucc: []string{"id1", "id3"},
			wantFail: map[string]string{
				"id2": "notFound: Email not found",
			},
		},
		{
			name:     "empty result",
			result:   map[string]any{},
			wantSucc: []string{},
			wantFail: map[string]string{},
		},
		{
			name: "error without description",
			result: map[string]any{
				"notUpdated": map[string]any{
					"id1": map[string]any{
						"type": "serverError",
					},
				},
			},
			wantSucc: []string{},
			wantFail: map[string]string{
				"id1": "serverError",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			succeeded, failed := parseBulkUpdateResult(tt.result)

			// Sort for comparison
			if !stringSlicesEqual(succeeded, tt.wantSucc) {
				t.Errorf("parseBulkUpdateResult() succeeded = %v, want %v", succeeded, tt.wantSucc)
			}

			if !reflect.DeepEqual(failed, tt.wantFail) {
				t.Errorf("parseBulkUpdateResult() failed = %v, want %v", failed, tt.wantFail)
			}
		})
	}
}

// Helper function to compare string slices (order-independent)
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]bool)
	for _, s := range a {
		aMap[s] = true
	}
	for _, s := range b {
		if !aMap[s] {
			return false
		}
	}
	return true
}

func TestMoveEmails_Multiple(t *testing.T) {
	// Response for successful move of 3 emails
	response := `{
		"methodResponses": [
			["Email/set", {
				"accountId": "acc123",
				"updated": {
					"email1": {},
					"email2": {},
					"email3": {}
				}
			}, "moveEmails"]
		]
	}`

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
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

	ids := []string{"email1", "email2", "email3"}
	targetMailboxID := "archive-456"
	result, err := client.MoveEmails(context.Background(), ids, targetMailboxID)

	if err != nil {
		t.Fatalf("MoveEmails() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("MoveEmails() returned nil result")
	}

	// All should succeed
	if len(result.Succeeded) != 3 {
		t.Errorf("MoveEmails() succeeded count = %d, want 3", len(result.Succeeded))
	}

	// Check all IDs are in succeeded
	expectedIDs := map[string]bool{"email1": true, "email2": true, "email3": true}
	for _, id := range result.Succeeded {
		if !expectedIDs[id] {
			t.Errorf("MoveEmails() unexpected succeeded ID: %s", id)
		}
	}

	// No failures
	if len(result.Failed) != 0 {
		t.Errorf("MoveEmails() failed count = %d, want 0", len(result.Failed))
	}
}

func TestMoveEmails_PartialFailure(t *testing.T) {
	// Response with some successes and some failures
	response := `{
		"methodResponses": [
			["Email/set", {
				"accountId": "acc123",
				"updated": {
					"email1": {},
					"email3": {}
				},
				"notUpdated": {
					"email2": {
						"type": "notFound",
						"description": "Email not found"
					}
				}
			}, "moveEmails"]
		]
	}`

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
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

	ids := []string{"email1", "email2", "email3"}
	targetMailboxID := "archive-456"
	result, err := client.MoveEmails(context.Background(), ids, targetMailboxID)

	if err != nil {
		t.Fatalf("MoveEmails() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("MoveEmails() returned nil result")
	}

	// 2 should succeed
	if len(result.Succeeded) != 2 {
		t.Errorf("MoveEmails() succeeded count = %d, want 2", len(result.Succeeded))
	}

	// Check succeeded IDs
	expectedSucceeded := map[string]bool{"email1": true, "email3": true}
	for _, id := range result.Succeeded {
		if !expectedSucceeded[id] {
			t.Errorf("MoveEmails() unexpected succeeded ID: %s", id)
		}
	}

	// 1 should fail
	if len(result.Failed) != 1 {
		t.Errorf("MoveEmails() failed count = %d, want 1", len(result.Failed))
	}

	// Check failed ID and error message
	if errMsg, exists := result.Failed["email2"]; !exists {
		t.Errorf("MoveEmails() email2 should be in failed map")
	} else if errMsg == "" {
		t.Errorf("MoveEmails() email2 should have error message")
	}
}

func TestMoveEmails_EmptyInput(t *testing.T) {
	client := NewClientWithBaseURL("test-token", "http://dummy")

	result, err := client.MoveEmails(context.Background(), []string{}, "target-123")

	if err != nil {
		t.Fatalf("MoveEmails() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("MoveEmails() returned nil result")
	}

	// Empty input should return empty result
	if len(result.Succeeded) != 0 {
		t.Errorf("MoveEmails() succeeded count = %d, want 0", len(result.Succeeded))
	}

	if len(result.Failed) != 0 {
		t.Errorf("MoveEmails() failed count = %d, want 0", len(result.Failed))
	}
}

func TestMarkEmailsRead_Multiple(t *testing.T) {
	// Response for successfully marking 3 emails as read
	response := `{
		"methodResponses": [
			["Email/set", {
				"accountId": "acc123",
				"updated": {
					"email1": {},
					"email2": {},
					"email3": {}
				}
			}, "markRead"]
		]
	}`

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
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

	ids := []string{"email1", "email2", "email3"}
	result, err := client.MarkEmailsRead(context.Background(), ids, true)

	if err != nil {
		t.Fatalf("MarkEmailsRead() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("MarkEmailsRead() returned nil result")
	}

	// All should succeed
	if len(result.Succeeded) != 3 {
		t.Errorf("MarkEmailsRead() succeeded count = %d, want 3", len(result.Succeeded))
	}

	// Check all IDs are in succeeded
	expectedIDs := map[string]bool{"email1": true, "email2": true, "email3": true}
	for _, id := range result.Succeeded {
		if !expectedIDs[id] {
			t.Errorf("MarkEmailsRead() unexpected succeeded ID: %s", id)
		}
	}

	// No failures
	if len(result.Failed) != 0 {
		t.Errorf("MarkEmailsRead() failed count = %d, want 0", len(result.Failed))
	}
}

func TestMarkEmailsRead_Unread(t *testing.T) {
	// Response for successfully marking 2 emails as unread
	response := `{
		"methodResponses": [
			["Email/set", {
				"accountId": "acc123",
				"updated": {
					"email1": {},
					"email2": {}
				}
			}, "markUnread"]
		]
	}`

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
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

	ids := []string{"email1", "email2"}
	result, err := client.MarkEmailsRead(context.Background(), ids, false)

	if err != nil {
		t.Fatalf("MarkEmailsRead() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("MarkEmailsRead() returned nil result")
	}

	// All should succeed
	if len(result.Succeeded) != 2 {
		t.Errorf("MarkEmailsRead() succeeded count = %d, want 2", len(result.Succeeded))
	}

	// Check all IDs are in succeeded
	expectedIDs := map[string]bool{"email1": true, "email2": true}
	for _, id := range result.Succeeded {
		if !expectedIDs[id] {
			t.Errorf("MarkEmailsRead() unexpected succeeded ID: %s", id)
		}
	}

	// No failures
	if len(result.Failed) != 0 {
		t.Errorf("MarkEmailsRead() failed count = %d, want 0", len(result.Failed))
	}
}

func TestMarkEmailsRead_PartialFailure(t *testing.T) {
	// Response with some successes and some failures
	response := `{
		"methodResponses": [
			["Email/set", {
				"accountId": "acc123",
				"updated": {
					"email1": {},
					"email3": {}
				},
				"notUpdated": {
					"email2": {
						"type": "notFound",
						"description": "Email not found"
					}
				}
			}, "markRead"]
		]
	}`

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
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

	ids := []string{"email1", "email2", "email3"}
	result, err := client.MarkEmailsRead(context.Background(), ids, true)

	if err != nil {
		t.Fatalf("MarkEmailsRead() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("MarkEmailsRead() returned nil result")
	}

	// 2 should succeed
	if len(result.Succeeded) != 2 {
		t.Errorf("MarkEmailsRead() succeeded count = %d, want 2", len(result.Succeeded))
	}

	// Check succeeded IDs
	expectedSucceeded := map[string]bool{"email1": true, "email3": true}
	for _, id := range result.Succeeded {
		if !expectedSucceeded[id] {
			t.Errorf("MarkEmailsRead() unexpected succeeded ID: %s", id)
		}
	}

	// 1 should fail
	if len(result.Failed) != 1 {
		t.Errorf("MarkEmailsRead() failed count = %d, want 1", len(result.Failed))
	}

	// Check failed ID and error message
	if errMsg, exists := result.Failed["email2"]; !exists {
		t.Errorf("MarkEmailsRead() email2 should be in failed map")
	} else if errMsg == "" {
		t.Errorf("MarkEmailsRead() email2 should have error message")
	}
}

func TestMarkEmailsRead_EmptyInput(t *testing.T) {
	client := NewClientWithBaseURL("test-token", "http://dummy")

	result, err := client.MarkEmailsRead(context.Background(), []string{}, true)

	if err != nil {
		t.Fatalf("MarkEmailsRead() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("MarkEmailsRead() returned nil result")
	}

	// Empty input should return empty result
	if len(result.Succeeded) != 0 {
		t.Errorf("MarkEmailsRead() succeeded count = %d, want 0", len(result.Succeeded))
	}

	if len(result.Failed) != 0 {
		t.Errorf("MarkEmailsRead() failed count = %d, want 0", len(result.Failed))
	}
}

func TestBulkResult_Empty(t *testing.T) {
	// Test that an empty BulkResult behaves correctly
	result := &BulkResult{
		Succeeded: []string{},
		Failed:    map[string]string{},
	}

	if len(result.Succeeded) != 0 {
		t.Errorf("Empty BulkResult succeeded count = %d, want 0", len(result.Succeeded))
	}

	if len(result.Failed) != 0 {
		t.Errorf("Empty BulkResult failed count = %d, want 0", len(result.Failed))
	}

	// Test with nil initialization
	var nilResult *BulkResult
	if nilResult != nil {
		t.Error("Nil BulkResult should be nil")
	}

	// Test zero value initialization
	var zeroResult BulkResult
	if zeroResult.Succeeded != nil {
		t.Errorf("Zero value BulkResult.Succeeded should be nil, got %v", zeroResult.Succeeded)
	}
	if zeroResult.Failed != nil {
		t.Errorf("Zero value BulkResult.Failed should be nil, got %v", zeroResult.Failed)
	}
}
