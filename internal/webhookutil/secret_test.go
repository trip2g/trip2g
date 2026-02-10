package webhookutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateSecret(t *testing.T) {
	s1, err := GenerateSecret()
	require.NoError(t, err)
	require.Len(t, s1, 64) // 32 bytes hex = 64 chars.

	s2, err := GenerateSecret()
	require.NoError(t, err)
	require.NotEqual(t, s1, s2)
}
