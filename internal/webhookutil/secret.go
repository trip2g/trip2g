package webhookutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateSecret generates a cryptographically random secret string (32 bytes hex).
func GenerateSecret() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate webhook secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}
