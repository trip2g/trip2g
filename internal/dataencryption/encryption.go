package dataencryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

type Config struct {
	Key string
}

func DefaultConfig() Config {
	return Config{
		Key: "please-change-me-to-32-byte-key!",
	}
}

// Manager handles encryption and decryption of sensitive data.
type Manager struct {
	key []byte
}

// NewManager creates a new Manager instance.
func NewManager(config Config) (*Manager, error) {
	key := []byte(config.Key)
	if len(key) != 32 {
		return nil, fmt.Errorf("data encryption key must be exactly 32 bytes, got %d", len(key))
	}

	return &Manager{
		key: key,
	}, nil
}

// EncryptData encrypts plaintext using AES-GCM and returns ciphertext as []byte.
func (m *Manager) EncryptData(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Seal appends the encrypted data to nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// DecryptData decrypts ciphertext using AES-GCM.
func (m *Manager) DecryptData(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}
