package crypto

import (
	"testing"
)

const testKey = "test-key-32-bytes-for-aes-256!!"

func TestEncryptDecrypt(t *testing.T) {
	original := "sensitive-data-123"
	ciphertext, err := EncryptWithKey(original, testKey)
	if err != nil {
		t.Fatalf("EncryptWithKey failed: %v", err)
	}
	if ciphertext == "" {
		t.Fatal("ciphertext should not be empty")
	}
	if ciphertext == original {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := DecryptWithKey(ciphertext, testKey)
	if err != nil {
		t.Fatalf("DecryptWithKey failed: %v", err)
	}
	if decrypted != original {
		t.Fatalf("expected '%s', got '%s'", original, decrypted)
	}
}

func TestEncryptDeterministicWithDifferentNonce(t *testing.T) {
	plaintext := "same-data"

	cipher1, _ := EncryptWithKey(plaintext, testKey)
	cipher2, _ := EncryptWithKey(plaintext, testKey)

	// Each encryption should produce different output (due to random nonce)
	if cipher1 == cipher2 {
		t.Fatal("encryption should produce different outputs due to random nonce")
	}

	// Both should decrypt correctly
	dec1, _ := DecryptWithKey(cipher1, testKey)
	dec2, _ := DecryptWithKey(cipher2, testKey)
	if dec1 != plaintext || dec2 != plaintext {
		t.Fatal("both should decrypt to original plaintext")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	original := "secret-message"
	ciphertext, err := EncryptWithKey(original, "key-one-1234567890123456")
	if err != nil {
		t.Fatalf("EncryptWithKey failed: %v", err)
	}

	_, err = DecryptWithKey(ciphertext, "key-two-9876543210987654")
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}
}

func TestDecryptInvalidBase64(t *testing.T) {
	_, err := DecryptWithKey("not-base64!!!", testKey)
	if err == nil {
		t.Fatal("expected error for invalid base64 input")
	}
}

func TestDecryptTooShort(t *testing.T) {
	_, err := DecryptWithKey("aGVsbG8=", testKey)
	if err == nil {
		t.Fatal("expected error for too-short ciphertext")
	}
}

func TestRoundTripEmptyString(t *testing.T) {
	ciphertext, err := EncryptWithKey("", testKey)
	if err != nil {
		t.Fatalf("EncryptWithKey empty string failed: %v", err)
	}

	decrypted, err := DecryptWithKey(ciphertext, testKey)
	if err != nil {
		t.Fatalf("DecryptWithKey empty string failed: %v", err)
	}
	if decrypted != "" {
		t.Fatalf("expected empty string, got '%s'", decrypted)
	}
}

func TestPadKey(t *testing.T) {
	// Key exactly 32 bytes
	if len(padKey("12345678901234567890123456789012")) != 32 {
		t.Fatal("32-byte key should remain 32 bytes")
	}

	// Key shorter than 32 bytes
	padded := padKey("short-key")
	if len(padded) != 32 {
		t.Fatal("short key should be padded to 32 bytes")
	}

	// Key longer than 32 bytes
	if len(padKey("this-is-a-very-long-key-that-exceeds-thirty-two-bytes")) != 32 {
		t.Fatal("long key should be truncated to 32 bytes")
	}
}
