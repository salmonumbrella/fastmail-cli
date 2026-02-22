package config

import (
	"encoding/json"
	"fmt"

	"github.com/99designs/keyring"
)

type sieveCredentials struct {
	Token  string `json:"token"`  // fma1-xxx format from browser
	Cookie string `json:"cookie"` // __Host-s_xxx=yyy format
}

// SaveSieveCredentials stores browser session credentials for Sieve API access.
// These are separate from API tokens and must be extracted from browser dev tools.
func SaveSieveCredentials(email, token, cookie string) error {
	email = normalize(email)
	if email == "" {
		return fmt.Errorf("missing email")
	}
	if token == "" {
		return fmt.Errorf("missing token")
	}
	if cookie == "" {
		return fmt.Errorf("missing cookie")
	}

	ring, err := openKeyring()
	if err != nil {
		return err
	}

	payload, err := json.Marshal(sieveCredentials{
		Token:  token,
		Cookie: cookie,
	})
	if err != nil {
		return err
	}

	return ring.Set(keyring.Item{
		Key:  sieveKey(email),
		Data: payload,
	})
}

// GetSieveCredentials retrieves browser session credentials for Sieve API access.
func GetSieveCredentials(email string) (token, cookie string, err error) {
	email = normalize(email)
	if email == "" {
		return "", "", fmt.Errorf("missing email")
	}

	ring, err := openKeyring()
	if err != nil {
		return "", "", err
	}

	item, err := ring.Get(sieveKey(email))
	if err != nil {
		return "", "", fmt.Errorf("sieve credentials not found for %s: %w", email, err)
	}

	var creds sieveCredentials
	if err := json.Unmarshal(item.Data, &creds); err != nil {
		return "", "", err
	}

	return creds.Token, creds.Cookie, nil
}

// DeleteSieveCredentials removes browser session credentials.
func DeleteSieveCredentials(email string) error {
	email = normalize(email)
	if email == "" {
		return fmt.Errorf("missing email")
	}

	ring, err := openKeyring()
	if err != nil {
		return err
	}

	return ring.Remove(sieveKey(email))
}

// HasSieveCredentials checks if sieve credentials exist for an account.
func HasSieveCredentials(email string) bool {
	_, _, err := GetSieveCredentials(email)
	return err == nil
}

func sieveKey(email string) string {
	return fmt.Sprintf("sieve:%s", email)
}
