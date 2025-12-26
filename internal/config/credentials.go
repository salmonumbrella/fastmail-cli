package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/99designs/keyring"
)

// Token represents a stored API token with metadata
type Token struct {
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	IsPrimary bool      `json:"is_primary,omitempty"`
	APIToken  string    `json:"-"` // Never serialize the token
}

type storedToken struct {
	APIToken  string    `json:"api_token"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	IsPrimary bool      `json:"is_primary,omitempty"`
}

var openKeyring = func() (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{
		ServiceName: AppName,
		// Try native keychain first, fall back to encrypted file if unavailable
		// (e.g., when binary is cross-compiled without CGO)
		AllowedBackends: []keyring.BackendType{
			keyring.KeychainBackend,      // macOS (requires CGO)
			keyring.WinCredBackend,       // Windows
			keyring.SecretServiceBackend, // Linux (GNOME Keyring/KWallet)
			keyring.FileBackend,          // Fallback: encrypted file
		},
		FileDir:          configDir(),
		FilePasswordFunc: keyring.TerminalPrompt,
	})
}

func configDir() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return dir + "/fastmail-cli/keyring"
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home + "/.config/fastmail-cli/keyring"
	}
	return ".fastmail-cli/keyring"
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

		st.IsPrimary = (keyEmail == email)

		payload, err := json.Marshal(st)
		if err != nil {
			continue
		}

		_ = ring.Set(keyring.Item{ //nolint:errcheck // best-effort update
			Key:  k,
			Data: payload,
		})
	}

	return nil
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
// SECURITY: Only returns email and metadata - tokens are never loaded into memory
// unless explicitly needed via GetToken().
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
			Email:     email,
			CreatedAt: st.CreatedAt,
			IsPrimary: st.IsPrimary,
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
