package shortapitoken

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

// TestParse_ForeignJWTRejected guards against token-type confusion: a validly
// signed HS256 JWT that lacks the short-API-token discriminator claim (e.g. a
// session-login JWT sharing the same signing secret) must NOT be accepted as a
// short API token.
func TestParse_ForeignJWTRejected(t *testing.T) {
	secret := "shared-secret"

	// Mimic a session JWT: same secret, HS256, but no short-API discriminator.
	foreign := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  int64(42),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := foreign.SignedString([]byte(secret))
	require.NoError(t, err)

	_, err = Parse(signed, secret)
	require.Error(t, err, "a JWT without the short-API discriminator must be rejected")
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
