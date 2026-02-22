package config

import "testing"

func TestSaveSieveCredentials(t *testing.T) {
	_ = setupMockKeyring(t)

	email := "test@fastmail.com"
	token := "fma1-test-token"
	cookie := "__Host-s_abc123=xyz789"

	err := SaveSieveCredentials(email, token, cookie)
	if err != nil {
		t.Fatalf("SaveSieveCredentials failed: %v", err)
	}

	gotToken, gotCookie, err := GetSieveCredentials(email)
	if err != nil {
		t.Fatalf("GetSieveCredentials failed: %v", err)
	}
	if gotToken != token {
		t.Errorf("token mismatch: got %q, want %q", gotToken, token)
	}
	if gotCookie != cookie {
		t.Errorf("cookie mismatch: got %q, want %q", gotCookie, cookie)
	}
}

func TestDeleteSieveCredentials(t *testing.T) {
	_ = setupMockKeyring(t)

	email := "test@fastmail.com"
	_ = SaveSieveCredentials(email, "fma1-token", "cookie")

	err := DeleteSieveCredentials(email)
	if err != nil {
		t.Fatalf("DeleteSieveCredentials failed: %v", err)
	}

	_, _, err = GetSieveCredentials(email)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}
