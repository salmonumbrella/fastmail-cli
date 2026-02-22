package tracking

import (
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	payload := &PixelPayload{
		Recipient:   "test@example.com",
		SubjectHash: "abc123",
		SentAt:      1234567890,
	}

	blob, err := Encrypt(payload, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	decrypted, err := Decrypt(blob, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if decrypted.Recipient != payload.Recipient {
		t.Errorf("Recipient: got %q, want %q", decrypted.Recipient, payload.Recipient)
	}
	if decrypted.SubjectHash != payload.SubjectHash {
		t.Errorf("SubjectHash: got %q, want %q", decrypted.SubjectHash, payload.SubjectHash)
	}
	if decrypted.SentAt != payload.SentAt {
		t.Errorf("SentAt: got %d, want %d", decrypted.SentAt, payload.SentAt)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1, _ := GenerateKey()
	key2, _ := GenerateKey()

	payload := &PixelPayload{Recipient: "test@example.com", SubjectHash: "abc", SentAt: 123}
	blob, _ := Encrypt(payload, key1)

	_, err := Decrypt(blob, key2)
	if err == nil {
		t.Error("expected error decrypting with wrong key")
	}
}

func TestDecryptInvalidBlob(t *testing.T) {
	key, _ := GenerateKey()

	_, err := Decrypt("not-valid-base64!!!", key)
	if err == nil {
		t.Error("expected error decrypting invalid blob")
	}
}

func TestGenerateKeyLength(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	// Base64 encoded 32 bytes = 44 characters (with padding) or 43 without
	if len(key) < 40 {
		t.Errorf("key too short: %d chars", len(key))
	}
}
