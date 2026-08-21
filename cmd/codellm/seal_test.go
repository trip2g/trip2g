package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/cmd/codellm/internal/codellm"
)

const sealTestKey = "12345678901234567890123456789012"

func TestRunSeal_RoundTripsThroughTheServiceFormat(t *testing.T) {
	t.Setenv(codellm.DefaultSealEnvKey, sealTestKey)
	var out bytes.Buffer

	require.NoError(t, runSeal(nil, strings.NewReader("krisp-token\n"), &out))

	blob := strings.TrimSpace(out.String())
	require.True(t, strings.HasPrefix(blob, "sealed:v1:"))

	// What the CLI prints must be what the running service can open.
	got, err := codellm.OpenSealedForTest(sealTestKey, blob)
	require.NoError(t, err)
	require.Equal(t, "krisp-token", got, "the shell's trailing newline is not part of the secret")
}

func TestRunSeal_CustomEnvKey(t *testing.T) {
	t.Setenv("SEAL_KEY_V2", sealTestKey)
	var out bytes.Buffer

	require.NoError(t, runSeal([]string{"--env-key", "SEAL_KEY_V2"}, strings.NewReader("v"), &out))
	require.Contains(t, out.String(), "sealed:v1:")
}

func TestRunSeal_UnsetKeyIsAnError(t *testing.T) {
	t.Setenv(codellm.DefaultSealEnvKey, "")
	err := runSeal(nil, strings.NewReader("v"), &bytes.Buffer{})
	require.ErrorContains(t, err, codellm.DefaultSealEnvKey)
}

func TestRunSeal_ShortKeyIsAnError(t *testing.T) {
	t.Setenv(codellm.DefaultSealEnvKey, "too-short")
	err := runSeal(nil, strings.NewReader("v"), &bytes.Buffer{})
	require.Error(t, err)
}
