package dataencryption

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	config := Config{
		Key: "12345678901234567890123456789012", // 32 bytes
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	testCases := []struct {
		name      string
		plaintext []byte
	}{
		{"empty", []byte{}},
		{"short", []byte("hello")},
		{"medium", []byte("this is a longer test message with more content")},
		{"binary", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, encryptErr := manager.EncryptData(tc.plaintext)
			if encryptErr != nil {
				t.Fatalf("encryption failed: %v", encryptErr)
			}

			// Encrypted should be different from plaintext (unless empty)
			if len(tc.plaintext) > 0 && bytes.Equal(encrypted, tc.plaintext) {
				t.Error("encrypted data should be different from plaintext")
			}

			decrypted, decryptErr := manager.DecryptData(encrypted)
			if decryptErr != nil {
				t.Fatalf("decryption failed: %v", decryptErr)
			}

			if !bytes.Equal(decrypted, tc.plaintext) {
				t.Errorf("decrypted data doesn't match original: got %v, want %v", decrypted, tc.plaintext)
			}
		})
	}
}

func TestNewManagerInvalidKeyLength(t *testing.T) {
	testCases := []struct {
		name string
		key  string
	}{
		{"too short", "short"},
		{"31 bytes", "1234567890123456789012345678901"},
		{"33 bytes", "123456789012345678901234567890123"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewManager(Config{Key: tc.key})
			if err == nil {
				t.Error("expected error for invalid key length")
			}
		})
	}
}

func TestDecryptInvalidData(t *testing.T) {
	config := Config{
		Key: "12345678901234567890123456789012",
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	testCases := []struct {
		name       string
		ciphertext []byte
	}{
		{"too short", []byte("abc")},
		{"corrupted", []byte{0x00, 0x01, 0x02}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, decryptErr := manager.DecryptData(tc.ciphertext)
			if decryptErr == nil {
				t.Error("expected error for invalid ciphertext")
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if len(config.Key) != 32 {
		t.Errorf("default key should be 32 bytes, got %d", len(config.Key))
	}
}
