package tracking

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// PixelPayload is encrypted into the tracking pixel URL
// to be decrypted by the worker.
type PixelPayload struct {
	Recipient   string `json:"r"`
	SubjectHash string `json:"s"`
	SentAt      int64  `json:"t"`
}

var errCiphertextTooShort = errors.New("ciphertext too short")

const defaultTrackingKeyVersion = byte(1)

// MaxKeyVersion is the maximum allowed key version (single byte in wire format).
const MaxKeyVersion = 255

// ValidateKeyVersion checks that a version number fits in the wire format.
func ValidateKeyVersion(version int) error {
	if version < 1 || version > MaxKeyVersion {
		return fmt.Errorf("key version %d out of range [1, %d]", version, MaxKeyVersion)
	}
	return nil
}

// Encrypt encrypts a PixelPayload into a URL-safe base64 blob using AES-GCM
func Encrypt(payload *PixelPayload, keyBase64 string) (string, error) {
	return EncryptWithVersion(payload, keyBase64, defaultTrackingKeyVersion)
}

func EncryptWithVersion(payload *PixelPayload, keyBase64 string, keyVersion byte) (string, error) {
	if keyVersion == 0 {
		keyVersion = defaultTrackingKeyVersion
	}

	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return "", fmt.Errorf("decode key: %w", err)
	}

	plaintext, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}

	ciphertext := aead.Seal(nonce, nonce, plaintext, nil)
	encodedPayload := make([]byte, 1+len(ciphertext))
	encodedPayload[0] = keyVersion
	copy(encodedPayload[1:], ciphertext)

	// URL-safe base64 encode with version byte
	return base64.RawURLEncoding.EncodeToString(encodedPayload), nil
}

// Decrypt decrypts a URL-safe base64 blob using AES-GCM
func Decrypt(blob string, keyBase64 string) (*PixelPayload, error) {
	decrypted, err := DecryptWithVersion(blob, keyBase64, defaultTrackingKeyVersion)
	if err == nil {
		return decrypted, nil
	}

	// Try legacy format (no version byte) as fallback
	legacyResult, legacyErr := decryptLegacy(blob, keyBase64)
	if legacyErr == nil {
		return legacyResult, nil
	}

	// Both failed — return the versioned error (more informative)
	return nil, fmt.Errorf("decrypt failed (versioned: %w; legacy: %v)", err, legacyErr)
}

func DecryptWithVersion(blob string, keyBase64 string, keyVersion byte) (*PixelPayload, error) {
	if keyVersion == 0 {
		keyVersion = defaultTrackingKeyVersion
	}

	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}

	raw, err := base64.RawURLEncoding.DecodeString(blob)
	if err != nil {
		return nil, fmt.Errorf("decode blob: %w", err)
	}

	if len(raw) < 1 {
		return nil, errCiphertextTooShort
	}

	if int(raw[0]) != int(keyVersion) {
		return nil, errors.New("key version mismatch")
	}

	ciphertext := raw[1:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}

	if len(ciphertext) < aead.NonceSize() {
		return nil, errCiphertextTooShort
	}

	nonce := ciphertext[:aead.NonceSize()]
	ciphertext = ciphertext[aead.NonceSize():]

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	var payload PixelPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	return &payload, nil
}

func decryptLegacy(blob string, keyBase64 string) (*PixelPayload, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}

	ciphertext, err := base64.RawURLEncoding.DecodeString(blob)
	if err != nil {
		return nil, fmt.Errorf("decode blob: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}

	if len(ciphertext) < aead.NonceSize() {
		return nil, errCiphertextTooShort
	}

	nonce := ciphertext[:aead.NonceSize()]
	ciphertext = ciphertext[aead.NonceSize():]

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	var payload PixelPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	return &payload, nil
}

// GenerateKey generates a new 256-bit AES key as base64
func GenerateKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}

	return base64.StdEncoding.EncodeToString(key), nil
}
