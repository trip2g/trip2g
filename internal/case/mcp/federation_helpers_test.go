package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"trip2g/internal/db"

	"github.com/stretchr/testify/require"
)

type federationVerifyEnvMock struct {
	secret    db.FederationSecret
	secretOK  bool
	subgraphs []string
}

func (m federationVerifyEnvMock) FederationSecretByKID(context.Context, string) (db.FederationSecret, bool, error) {
	if !m.secretOK {
		return db.FederationSecret{}, false, nil
	}
	return m.secret, true, nil
}

func (m federationVerifyEnvMock) ListFederationSecretSubgraphsByKID(context.Context, string) ([]string, error) {
	return m.subgraphs, nil
}

func (m federationVerifyEnvMock) DecryptData(data []byte) ([]byte, error) {
	return data, nil
}

func TestSplitKBID(t *testing.T) {
	tests := []struct {
		in       string
		wantHead string
		wantRest string
	}{
		{in: "", wantHead: "", wantRest: ""},
		{in: "a", wantHead: "a", wantRest: ""},
		{in: "a/b/c", wantHead: "a", wantRest: "b/c"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			head, rest := splitKBID(tt.in)
			require.Equal(t, tt.wantHead, head)
			require.Equal(t, tt.wantRest, rest)
		})
	}
}

func TestPrefixKBID(t *testing.T) {
	items := []SearchResultItem{
		{Title: "local"},
		{
			Title: "remote",
			Federation: &FederationRef{
				KBID:             "cellbio",
				KBURL:            "https://science.example/_system/mcp",
				AgentInstruction: "open remote",
			},
		},
	}

	prefixKBID("science", items)

	require.Nil(t, items[0].Federation)
	require.Equal(t, "science/cellbio", items[1].Federation.KBID)
	require.Equal(t, "https://science.example/_system/mcp", items[1].Federation.KBURL)
	require.Equal(t, "open remote", items[1].Federation.AgentInstruction)
}

func TestSignOutboundVerifyInboundRoundTrip(t *testing.T) {
	secret := []byte("12345678901234567890123456789012")
	token, err := signOutbound(secret, "alice", "https://hub.local", "rid-1")
	require.NoError(t, err)

	kid, allowed, err := verifyInbound(context.Background(), federationVerifyEnvMock{
		secret:    db.FederationSecret{Kid: "alice", SecretCrypt: secret},
		secretOK:  true,
		subgraphs: []string{"team"},
	}, token)

	require.NoError(t, err)
	require.Equal(t, "alice", kid)
	require.Equal(t, []string{"team"}, allowed)
}

func TestVerifyInboundSentinels(t *testing.T) {
	secret := []byte("12345678901234567890123456789012")
	now := time.Now()

	valid, err := signOutbound(secret, "alice", "https://hub.local", "rid-1")
	require.NoError(t, err)

	t.Run("unknown kid", func(t *testing.T) {
		_, _, err := verifyInbound(context.Background(), federationVerifyEnvMock{}, valid) //nolint:govet
		require.True(t, errors.Is(err, ErrFedAuthUnknownKid))
	})

	t.Run("bad signature", func(t *testing.T) {
		_, _, err := verifyInbound(context.Background(), federationVerifyEnvMock{ //nolint:govet
			secret:   db.FederationSecret{Kid: "alice", SecretCrypt: []byte("other-secret")},
			secretOK: true,
		}, valid)
		require.True(t, errors.Is(err, ErrFedAuthBadSig))
	})

	t.Run("expired", func(t *testing.T) {
		token, err := signOutboundAt(secret, "alice", "https://hub.local", "rid-1", now.Add(-time.Minute), 30*time.Second) //nolint:govet
		require.NoError(t, err)

		_, _, err = verifyInbound(context.Background(), federationVerifyEnvMock{
			secret:   db.FederationSecret{Kid: "alice", SecretCrypt: secret},
			secretOK: true,
		}, token)
		require.True(t, errors.Is(err, ErrFedAuthExpired))
	})

	t.Run("future iat", func(t *testing.T) {
		token, err := signOutboundAt(secret, "alice", "https://hub.local", "rid-1", now.Add(10*time.Second), 30*time.Second)
		require.NoError(t, err)

		_, _, err = verifyInbound(context.Background(), federationVerifyEnvMock{
			secret:   db.FederationSecret{Kid: "alice", SecretCrypt: secret},
			secretOK: true,
		}, token)
		require.True(t, errors.Is(err, ErrFedAuthFutureIAT))
	})

	t.Run("revoked", func(t *testing.T) {
		_, _, err := verifyInbound(context.Background(), federationVerifyEnvMock{
			secret:   db.FederationSecret{Kid: "alice", SecretCrypt: secret, RevokedAt: &now},
			secretOK: true,
		}, valid)
		require.True(t, errors.Is(err, ErrFedAuthRevoked))
	})
}
