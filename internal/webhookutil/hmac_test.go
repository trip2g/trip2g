package webhookutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSignHMAC(t *testing.T) {
	body := []byte(`{"test":"data"}`)
	secret := "test-secret"

	sig := SignHMAC(body, secret)
	require.Contains(t, sig, "sha256=")
	require.True(t, VerifyHMAC(body, secret, sig))
}

func TestVerifyHMAC_WrongSecret(t *testing.T) {
	body := []byte(`{"test":"data"}`)

	sig := SignHMAC(body, "secret-1")
	require.False(t, VerifyHMAC(body, "secret-2", sig))
}

func TestVerifyHMAC_DifferentBody(t *testing.T) {
	secret := "test-secret"

	sig := SignHMAC([]byte("body-1"), secret)
	require.False(t, VerifyHMAC([]byte("body-2"), secret, sig))
}

func TestSignHMAC_DifferentSecrets(t *testing.T) {
	body := []byte(`same body`)

	sig1 := SignHMAC(body, "secret-1")
	sig2 := SignHMAC(body, "secret-2")
	require.NotEqual(t, sig1, sig2)
}
