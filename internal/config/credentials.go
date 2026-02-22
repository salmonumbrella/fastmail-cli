package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/99designs/keyring"
	"github.com/salmonumbrella/fastmail-cli/internal/keyringutil"
)

// Token represents a stored API token with metadata
type Token struct {
	Email           string    `json:"email"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
	IsPrimary       bool      `json:"is_primary,omitempty"`
	DefaultIdentity string    `json:"default_identity,omitempty"` // Preferred sending identity for this account
	APIToken        string    `json:"-"`                          // Never serialize the token
}

type storedToken struct {
	APIToken        string    `json:"api_token"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
	IsPrimary       bool      `json:"is_primary,omitempty"`
	DefaultIdentity string    `json:"default_identity,omitempty"`
}

var openKeyring = func() (keyring.Keyring, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName:      AppName,
		AllowedBackends:  keyringAllowedBackends(),
		FileDir:          configDir(),
		FilePasswordFunc: keyringPasswordFunc,
	})
	if err != nil {
		return nil, err
	}
	return keyringutil.Wrap(ring, keyringutil.DefaultTimeout), nil
}

func configDir() string {
	if dir := strings.TrimSpace(os.Getenv(CredentialsDirEnvVarName)); dir != "" {
		return filepath.Join(dir, AppName, "keyring")
	}
	if dir := strings.TrimSpace(os.Getenv(SharedCredentialsDirEnvVarName)); dir != "" {
		return filepath.Join(dir, AppName, "keyring")
	}

	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, AppName, "keyring")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", AppName, "keyring")
	}
	return filepath.Join("."+AppName, "keyring")
}

func keyringAllowedBackends() []keyring.BackendType {
	return allowedBackendsFor(runtime.GOOS, os.Getenv("DBUS_SESSION_BUS_ADDRESS"), os.Getenv(KeyringBackendEnvVarName))
}

func keyringPasswordFunc(prompt string) (string, error) {
	if password, ok := os.LookupEnv(KeyringPasswordEnvVarName); ok {
		return password, nil
	}
	return keyring.TerminalPrompt(prompt)
}

func allowedBackendsFor(goos, dbusSessionBusAddress, backendSetting string) []keyring.BackendType {
	defaultBackends := []keyring.BackendType{
		keyring.KeychainBackend,      // macOS (requires CGO)
		keyring.WinCredBackend,       // Windows
		keyring.SecretServiceBackend, // Linux (GNOME Keyring/KWallet)
		keyring.FileBackend,          // Fallback: encrypted file
	}

	backend := normalize(strings.TrimSpace(backendSetting))
	if backend == "" || backend == "auto" || backend == "default" {
		// Linux headless environments often lack a DBus session; force file
		// backend to avoid Secret Service failures/hangs in non-interactive runs.
		if goos == "linux" && strings.TrimSpace(dbusSessionBusAddress) == "" {
			return []keyring.BackendType{keyring.FileBackend}
		}
		return defaultBackends
	}

	switch backend {
	case "file":
		return []keyring.BackendType{keyring.FileBackend}
	case "keychain":
		return []keyring.BackendType{keyring.KeychainBackend, keyring.FileBackend}
	case "wincred", "windows":
		return []keyring.BackendType{keyring.WinCredBackend, keyring.FileBackend}
	case "secret-service", "secretservice":
		return []keyring.BackendType{keyring.SecretServiceBackend, keyring.FileBackend}
	default:
		return defaultBackends
	}
}

// SaveToken stores an API token in the OS keychain
// If this is the first account, it's automatically set as primary
func SaveToken(email, token string) error {
	email = normalize(email)
	if email == "" {
		return fmt.Errorf("missing email")
	}
	if token == "" {
		return fmt.Errorf("missing token")
	}

	ring, err := openKeyring()
	if err != nil {
		return err
	}

	// Check if this is the first account (make it primary)
	accounts, _ := ListAccounts() //nolint:errcheck // best-effort check for existing accounts
	isPrimary := len(accounts) == 0

	payload, err := json.Marshal(storedToken{
		APIToken:  token,
		CreatedAt: time.Now().UTC(),
		IsPrimary: isPrimary,
	})
	if err != nil {
		return err
	}

	return ring.Set(keyring.Item{
		Key:  tokenKey(email),
		Data: payload,
	})
}

// SetPrimaryAccount sets the specified email as the primary account
func SetPrimaryAccount(email string) error {
	email = normalize(email)
	if email == "" {
		return fmt.Errorf("missing email")
	}

	ring, err := openKeyring()
	if err != nil {
		return err
	}

	keys, err := ring.Keys()
	if err != nil {
		return err
	}

	// Update all accounts - set primary only for the specified email
	found := false
	for _, k := range keys {
		keyEmail, ok := parseTokenKey(k)
		if !ok {
			continue
		}

		item, err := ring.Get(k)
		if err != nil {
			continue
		}

		var st storedToken
		if unmarshalErr := json.Unmarshal(item.Data, &st); unmarshalErr != nil {
			continue
		}

		isTarget := keyEmail == email
		if isTarget {
			found = true
		}
		st.IsPrimary = isTarget

		payload, err := json.Marshal(st)
		if err != nil {
			continue
		}

		if setErr := ring.Set(keyring.Item{
			Key:  k,
			Data: payload,
		}); setErr != nil && isTarget {
			return fmt.Errorf("failed to set primary account: %w", setErr)
		}
	}

	if !found {
		return fmt.Errorf("account not found: %s", email)
	}

	return nil
}

// SetDefaultIdentity sets the default sending identity for an account
func SetDefaultIdentity(accountEmail, identityEmail string) error {
	accountEmail = normalize(accountEmail)
	identityEmail = normalize(identityEmail)
	if accountEmail == "" {
		return fmt.Errorf("missing account email")
	}
	if identityEmail == "" {
		return fmt.Errorf("missing identity email")
	}

	ring, err := openKeyring()
	if err != nil {
		return err
	}

	key := tokenKey(accountEmail)
	item, err := ring.Get(key)
	if err != nil {
		return fmt.Errorf("account not found: %s", accountEmail)
	}

	var st storedToken
	if unmarshalErr := json.Unmarshal(item.Data, &st); unmarshalErr != nil {
		return unmarshalErr
	}

	st.DefaultIdentity = identityEmail

	payload, marshalErr := json.Marshal(st)
	if marshalErr != nil {
		return marshalErr
	}

	return ring.Set(keyring.Item{
		Key:  key,
		Data: payload,
	})
}

// GetDefaultIdentity returns the default sending identity for an account
func GetDefaultIdentity(accountEmail string) (string, error) {
	accountEmail = normalize(accountEmail)
	if accountEmail == "" {
		return "", fmt.Errorf("missing account email")
	}

	ring, err := openKeyring()
	if err != nil {
		return "", err
	}

	item, err := ring.Get(tokenKey(accountEmail))
	if err != nil {
		return "", nil // No default set, return empty
	}

	var st storedToken
	if err := json.Unmarshal(item.Data, &st); err != nil {
		return "", err
	}

	return st.DefaultIdentity, nil
}

// GetPrimaryAccount returns the primary account email, or empty string if none
func GetPrimaryAccount() (string, error) {
	tokens, err := ListTokens()
	if err != nil {
		return "", err
	}

	for _, t := range tokens {
		if t.IsPrimary {
			return t.Email, nil
		}
	}

	// If no primary is set but accounts exist, return the first one
	if len(tokens) > 0 {
		return tokens[0].Email, nil
	}

	return "", nil
}

// GetToken retrieves an API token from the OS keychain
func GetToken(email string) (string, error) {
	email = normalize(email)
	if email == "" {
		return "", fmt.Errorf("missing email")
	}

	ring, err := openKeyring()
	if err != nil {
		return "", err
	}

	item, err := ring.Get(tokenKey(email))
	if err != nil {
		return "", err
	}

	var st storedToken
	if err := json.Unmarshal(item.Data, &st); err != nil {
		return "", err
	}

	return st.APIToken, nil
}

// DeleteToken removes an API token from the OS keychain
func DeleteToken(email string) error {
	email = normalize(email)
	if email == "" {
		return fmt.Errorf("missing email")
	}

	ring, err := openKeyring()
	if err != nil {
		return err
	}

	return ring.Remove(tokenKey(email))
}

// ListAccounts returns a list of all configured account emails
func ListAccounts() ([]string, error) {
	ring, err := openKeyring()
	if err != nil {
		return nil, err
	}

	keys, err := ring.Keys()
	if err != nil {
		return nil, err
	}

	accounts := make([]string, 0)
	for _, k := range keys {
		email, ok := parseTokenKey(k)
		if !ok {
			continue
		}
		accounts = append(accounts, email)
	}

	return accounts, nil
}

// ListTokens returns all stored tokens with metadata.
// NOTE: The underlying keyring Get() call loads the full stored item (including the
// API token) into memory during unmarshalling; however, the token is NOT included in
// the returned Token structs. Use GetToken() to retrieve the actual API token.
func ListTokens() ([]Token, error) {
	ring, err := openKeyring()
	if err != nil {
		return nil, err
	}

	keys, err := ring.Keys()
	if err != nil {
		return nil, err
	}

	tokens := make([]Token, 0)
	for _, k := range keys {
		email, ok := parseTokenKey(k)
		if !ok {
			continue
		}

		item, err := ring.Get(k)
		if err != nil {
			return nil, err
		}

		var st storedToken
		if err := json.Unmarshal(item.Data, &st); err != nil {
			return nil, err
		}

		// SECURITY: Do NOT include APIToken in returned data.
		// Tokens should only be retrieved when actually needed via GetToken().
		tokens = append(tokens, Token{
			Email:           email,
			CreatedAt:       st.CreatedAt,
			IsPrimary:       st.IsPrimary,
			DefaultIdentity: st.DefaultIdentity,
			// APIToken intentionally omitted - use GetToken() when needed
		})
	}

	return tokens, nil
}

func parseTokenKey(k string) (email string, ok bool) {
	const prefix = "token:"
	if !strings.HasPrefix(k, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(k, prefix)
	if strings.TrimSpace(rest) == "" {
		return "", false
	}
	return rest, true
}

func tokenKey(email string) string {
	return fmt.Sprintf("token:%s", email)
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
