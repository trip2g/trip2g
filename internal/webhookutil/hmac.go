package webhookutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// SignHMAC computes HMAC-SHA256 of body using secret.
// Returns "sha256=<hex>" format string.
func SignHMAC(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// VerifyHMAC verifies a signature in "sha256=<hex>" format.
func VerifyHMAC(body []byte, secret string, signature string) bool {
	expected := SignHMAC(body, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}
