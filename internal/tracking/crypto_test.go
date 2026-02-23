package tracking

import (
	"strings"
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

func TestValidateKeyVersion(t *testing.T) {
	tests := []struct {
		version int
		wantErr bool
	}{
		{1, false},
		{255, false},
		{256, true},
		{0, true},
		{-1, true},
	}
	for _, tt := range tests {
		err := ValidateKeyVersion(tt.version)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateKeyVersion(%d) err=%v, wantErr=%v", tt.version, err, tt.wantErr)
		}
	}
}

func TestDecrypt_PreservesVersionError(t *testing.T) {
	key1, _ := GenerateKey()
	key2, _ := GenerateKey()
	payload := &PixelPayload{Recipient: "test@example.com", SentAt: 1234567890}

	blob, err := EncryptWithVersion(payload, key1, 2)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Decrypt(blob, key2)
	if err == nil {
		t.Fatal("expected error decrypting with wrong key")
	}
	if !strings.Contains(err.Error(), "decrypt") {
		t.Fatalf("error should mention decrypt failure: %v", err)
	}
}
