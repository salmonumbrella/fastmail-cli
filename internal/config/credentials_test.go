package config

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/99designs/keyring"
)

// mockKeyring is a simple in-memory keyring for testing
type mockKeyring struct {
	items map[string]keyring.Item
}

func newMockKeyring() *mockKeyring {
	return &mockKeyring{items: make(map[string]keyring.Item)}
}

func (m *mockKeyring) Get(key string) (keyring.Item, error) {
	if item, ok := m.items[key]; ok {
		return item, nil
	}
	return keyring.Item{}, keyring.ErrKeyNotFound
}

func (m *mockKeyring) Set(item keyring.Item) error {
	m.items[item.Key] = item
	return nil
}

func (m *mockKeyring) Remove(key string) error {
	delete(m.items, key)
	return nil
}

func (m *mockKeyring) Keys() ([]string, error) {
	keys := make([]string, 0, len(m.items))
	for k := range m.items {
		keys = append(keys, k)
	}
	return keys, nil
}

func (m *mockKeyring) GetMetadata(_ string) (keyring.Metadata, error) {
	return keyring.Metadata{}, nil
}

func setupMockKeyring(t *testing.T) *mockKeyring {
	t.Helper()
	mock := newMockKeyring()
	originalOpenKeyring := openKeyring
	openKeyring = func() (keyring.Keyring, error) {
		return mock, nil
	}
	t.Cleanup(func() {
		openKeyring = originalOpenKeyring
	})
	return mock
}

func TestSetDefaultIdentity(t *testing.T) {
	mock := setupMockKeyring(t)

	// First, save a token so the account exists
	accountEmail := "user@example.com"
	payload, _ := json.Marshal(storedToken{
		APIToken:  "test-token",
		IsPrimary: true,
	})
	_ = mock.Set(keyring.Item{
		Key:  tokenKey(accountEmail),
		Data: payload,
	})

	// Test setting default identity
	identityEmail := "alias@example.com"
	err := SetDefaultIdentity(accountEmail, identityEmail)
	if err != nil {
		t.Fatalf("SetDefaultIdentity failed: %v", err)
	}

	// Verify it was saved
	result, err := GetDefaultIdentity(accountEmail)
	if err != nil {
		t.Fatalf("GetDefaultIdentity failed: %v", err)
	}
	if result != identityEmail {
		t.Errorf("GetDefaultIdentity = %q, want %q", result, identityEmail)
	}
}

func TestSetDefaultIdentity_NonexistentAccount(t *testing.T) {
	_ = setupMockKeyring(t)

	err := SetDefaultIdentity("nonexistent@example.com", "identity@example.com")
	if err == nil {
		t.Fatal("SetDefaultIdentity should fail for nonexistent account")
	}
}

func TestSetDefaultIdentity_EmptyAccount(t *testing.T) {
	_ = setupMockKeyring(t)

	err := SetDefaultIdentity("", "identity@example.com")
	if err == nil {
		t.Fatal("SetDefaultIdentity should fail for empty account email")
	}
}

func TestSetDefaultIdentity_EmptyIdentity(t *testing.T) {
	_ = setupMockKeyring(t)

	err := SetDefaultIdentity("user@example.com", "")
	if err == nil {
		t.Fatal("SetDefaultIdentity should fail for empty identity email")
	}
}

func TestGetDefaultIdentity_NoDefault(t *testing.T) {
	mock := setupMockKeyring(t)

	// Create account without default identity
	accountEmail := "user@example.com"
	payload, _ := json.Marshal(storedToken{
		APIToken:  "test-token",
		IsPrimary: true,
	})
	_ = mock.Set(keyring.Item{
		Key:  tokenKey(accountEmail),
		Data: payload,
	})

	// Should return empty string when no default is set
	result, err := GetDefaultIdentity(accountEmail)
	if err != nil {
		t.Fatalf("GetDefaultIdentity failed: %v", err)
	}
	if result != "" {
		t.Errorf("GetDefaultIdentity = %q, want empty string", result)
	}
}

func TestGetDefaultIdentity_NonexistentAccount(t *testing.T) {
	_ = setupMockKeyring(t)

	// Should return empty string for nonexistent account (not error)
	result, err := GetDefaultIdentity("nonexistent@example.com")
	if err != nil {
		t.Fatalf("GetDefaultIdentity should not error for nonexistent account: %v", err)
	}
	if result != "" {
		t.Errorf("GetDefaultIdentity = %q, want empty string", result)
	}
}

func TestSetDefaultIdentity_CaseInsensitive(t *testing.T) {
	mock := setupMockKeyring(t)

	// Create account
	accountEmail := "User@Example.COM"
	payload, _ := json.Marshal(storedToken{
		APIToken:  "test-token",
		IsPrimary: true,
	})
	_ = mock.Set(keyring.Item{
		Key:  tokenKey(normalize(accountEmail)),
		Data: payload,
	})

	// Set identity with different case
	identityEmail := "Alias@Example.COM"
	err := SetDefaultIdentity(accountEmail, identityEmail)
	if err != nil {
		t.Fatalf("SetDefaultIdentity failed: %v", err)
	}

	// Verify (should be normalized to lowercase)
	result, err := GetDefaultIdentity(accountEmail)
	if err != nil {
		t.Fatalf("GetDefaultIdentity failed: %v", err)
	}
	if result != "alias@example.com" {
		t.Errorf("GetDefaultIdentity = %q, want %q", result, "alias@example.com")
	}
}

func TestListTokens_IncludesDefaultIdentity(t *testing.T) {
	mock := setupMockKeyring(t)

	// Create account with default identity
	accountEmail := "user@example.com"
	identityEmail := "alias@example.com"
	payload, _ := json.Marshal(storedToken{
		APIToken:        "test-token",
		IsPrimary:       true,
		DefaultIdentity: identityEmail,
	})
	_ = mock.Set(keyring.Item{
		Key:  tokenKey(accountEmail),
		Data: payload,
	})

	// List tokens should include the default identity
	tokens, err := ListTokens()
	if err != nil {
		t.Fatalf("ListTokens failed: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("ListTokens returned %d tokens, want 1", len(tokens))
	}
	if tokens[0].DefaultIdentity != identityEmail {
		t.Errorf("Token.DefaultIdentity = %q, want %q", tokens[0].DefaultIdentity, identityEmail)
	}
}

func TestConfigDir_UsesCredentialsDirEnv(t *testing.T) {
	base := filepath.Join(t.TempDir(), "credentials")
	t.Setenv(CredentialsDirEnvVarName, base)
	t.Setenv(SharedCredentialsDirEnvVarName, filepath.Join(t.TempDir(), "shared"))

	got := configDir()
	want := filepath.Join(base, AppName, "keyring")
	if got != want {
		t.Fatalf("configDir() = %q, want %q", got, want)
	}
}

func TestConfigDir_UsesSharedCredentialsDirEnv(t *testing.T) {
	base := filepath.Join(t.TempDir(), "shared-credentials")
	t.Setenv(CredentialsDirEnvVarName, "")
	t.Setenv(SharedCredentialsDirEnvVarName, base)

	got := configDir()
	want := filepath.Join(base, AppName, "keyring")
	if got != want {
		t.Fatalf("configDir() = %q, want %q", got, want)
	}
}

func TestAllowedBackendsFor_LinuxHeadlessAutoUsesFile(t *testing.T) {
	got := allowedBackendsFor("linux", "", "auto")
	want := []keyring.BackendType{keyring.FileBackend}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("allowedBackendsFor(linux, empty dbus, auto) = %v, want %v", got, want)
	}
}

func TestAllowedBackendsFor_LinuxHeadlessDefaultUsesFile(t *testing.T) {
	got := allowedBackendsFor("linux", "", "default")
	want := []keyring.BackendType{keyring.FileBackend}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("allowedBackendsFor(linux, empty dbus, default) = %v, want %v", got, want)
	}
}

func TestAllowedBackendsFor_ExplicitFile(t *testing.T) {
	got := allowedBackendsFor("darwin", "ignored", "file")
	want := []keyring.BackendType{keyring.FileBackend}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("allowedBackendsFor(darwin, ignored, file) = %v, want %v", got, want)
	}
}

func TestKeyringPasswordFunc_UsesEnv(t *testing.T) {
	t.Setenv(KeyringPasswordEnvVarName, "test-password")

	got, err := keyringPasswordFunc("prompt")
	if err != nil {
		t.Fatalf("keyringPasswordFunc() error = %v", err)
	}
	if got != "test-password" {
		t.Fatalf("keyringPasswordFunc() = %q, want %q", got, "test-password")
	}
}
