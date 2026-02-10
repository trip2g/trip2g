package shortapitoken

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSignParse_Roundtrip(t *testing.T) {
	secret := "test-secret-key-for-jwt"

	d := Data{
		Depth:         1,
		ReadPatterns:  []string{"blog/**", "docs/*"},
		WritePatterns: []string{"blog/**"},
	}

	token, err := Sign(d, secret, time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	parsed, err := Parse(token, secret)
	require.NoError(t, err)
	require.Equal(t, d.Depth, parsed.Depth)
	require.Equal(t, d.ReadPatterns, parsed.ReadPatterns)
	require.Equal(t, d.WritePatterns, parsed.WritePatterns)
}

func TestParse_ExpiredToken(t *testing.T) {
	secret := "test-secret"

	d := Data{Depth: 0}

	// Sign with negative TTL (already expired).
	token, err := Sign(d, secret, -time.Hour)
	require.NoError(t, err)

	_, err = Parse(token, secret)
	require.Error(t, err)
	require.Contains(t, err.Error(), "token is expired")
}

func TestParse_WrongSecret(t *testing.T) {
	d := Data{Depth: 0}

	token, err := Sign(d, "secret-1", time.Hour)
	require.NoError(t, err)

	_, err = Parse(token, "secret-2")
	require.Error(t, err)
}

func TestSignParse_EmptyPatterns(t *testing.T) {
	secret := "test-secret"

	d := Data{
		Depth:         0,
		ReadPatterns:  []string{"*"},
		WritePatterns: []string{},
	}

	token, err := Sign(d, secret, time.Hour)
	require.NoError(t, err)

	parsed, err := Parse(token, secret)
	require.NoError(t, err)
	require.Equal(t, []string{"*"}, parsed.ReadPatterns)
	require.Empty(t, parsed.WritePatterns)
}
