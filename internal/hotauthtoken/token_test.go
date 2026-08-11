package hotauthtoken_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/hotauthtoken"
	"trip2g/internal/model"
)

// TestNewTokenParseToken_RoundTrip mirrors the construction used by
// cmd/server/loginlink.go: mint an admin-enter token for an owner email, then
// confirm a Manager built with the same secret (as signinbyhat.Resolve does
// via ParseHotAuthToken) accepts it and recovers the same claims.
func TestNewTokenParseToken_RoundTrip(t *testing.T) {
	manager := hotauthtoken.NewManager(hotauthtoken.Config{Secret: "test-secret", ExpiresIn: 5 * time.Minute})

	token, err := manager.NewToken(model.HotAuthToken{Email: "owner@example.com", AdminEnter: true})
	require.NoError(t, err)
	require.NotEmpty(t, token)

	got, err := manager.ParseToken(token)
	require.NoError(t, err)
	require.Equal(t, "owner@example.com", got.Email)
	require.True(t, got.AdminEnter)
}

func TestParseToken_WrongSecret_Rejected(t *testing.T) {
	minter := hotauthtoken.NewManager(hotauthtoken.Config{Secret: "secret-a", ExpiresIn: 5 * time.Minute})
	verifier := hotauthtoken.NewManager(hotauthtoken.Config{Secret: "secret-b", ExpiresIn: 5 * time.Minute})

	token, err := minter.NewToken(model.HotAuthToken{Email: "owner@example.com", AdminEnter: true})
	require.NoError(t, err)

	_, err = verifier.ParseToken(token)
	require.Error(t, err)
}

// TestNewTokenWithTTL_OverridesConfiguredLifetime covers the admin createHatLink
// path, which caps the link lifetime per request instead of using the manager's
// configured ExpiresIn.
func TestNewTokenWithTTL_OverridesConfiguredLifetime(t *testing.T) {
	manager := hotauthtoken.NewManager(hotauthtoken.Config{Secret: "test-secret", ExpiresIn: time.Hour})

	expired, err := manager.NewTokenWithTTL(model.HotAuthToken{Email: "owner@example.com"}, -time.Minute)
	require.NoError(t, err)
	_, err = manager.ParseToken(expired)
	require.Error(t, err)

	live, err := manager.NewTokenWithTTL(model.HotAuthToken{Email: "owner@example.com"}, 30*time.Minute)
	require.NoError(t, err)
	got, err := manager.ParseToken(live)
	require.NoError(t, err)
	require.Equal(t, "owner@example.com", got.Email)
}
