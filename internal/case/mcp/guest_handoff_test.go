package mcp

import (
	"testing"
	"time"

	"trip2g/internal/db"

	"github.com/stretchr/testify/require"
)

func guestHandoffEnv(secret []byte) federationVerifyEnvMock {
	return federationVerifyEnvMock{
		secret:    db.FederationSecret{Kid: "edge", SecretCrypt: secret},
		secretOK:  true,
		subgraphs: []string{"team"},
	}
}

func TestAGuestHandoffVerifiesWithTheEdgeSecret(t *testing.T) {
	secret := []byte("12345678901234567890123456789012")
	handoff := GuestHandoff{Path: "/", Hop: 1, Reader: "abc"}

	token, err := SignGuestHandoff(secret, "edge", "https://hub.test", handoff)
	require.NoError(t, err)

	got, err := VerifyGuestHandoff(t.Context(), guestHandoffEnv(secret), token)
	require.NoError(t, err)
	require.Equal(t, "edge", got.Caller.KID)
	require.Equal(t, []string{"team"}, got.Caller.AllowedSubgraphs)
	require.Equal(t, handoff, got.GuestHandoff)
	require.NotEmpty(t, got.ID)
	require.WithinDuration(t, time.Now().Add(GuestHandoffTTL), got.ExpiresAt, 2*time.Second)
}

func TestAnMCPTokenIsNotAGuestHandoff(t *testing.T) {
	secret := []byte("12345678901234567890123456789012")

	token, err := signOutbound(secret, "edge", "https://hub.test", "rid-1", nil)
	require.NoError(t, err)

	_, err = VerifyGuestHandoff(t.Context(), guestHandoffEnv(secret), token)
	require.ErrorIs(t, err, ErrFedAuthWrongUse)
}

func TestAGuestHandoffIsNotAnMCPToken(t *testing.T) {
	secret := []byte("12345678901234567890123456789012")

	token, err := SignGuestHandoff(secret, "edge", "https://hub.test", GuestHandoff{Path: "/", Hop: 1, Reader: "abc"})
	require.NoError(t, err)

	_, err = VerifyFederationBearer(t.Context(), guestHandoffEnv(secret), token, nil)
	require.ErrorIs(t, err, ErrFedAuthWrongUse, "a token a browser carried must not open the tool surface")
}

func TestAGuestHandoffSignedWithAnotherSecretIsRefused(t *testing.T) {
	token, err := SignGuestHandoff([]byte("another-secret-another-secret-xx"), "edge", "https://hub.test", GuestHandoff{Path: "/", Hop: 1, Reader: "abc"})
	require.NoError(t, err)

	_, err = VerifyGuestHandoff(t.Context(), guestHandoffEnv([]byte("12345678901234567890123456789012")), token)
	require.ErrorIs(t, err, ErrFedAuthBadSig)
}
