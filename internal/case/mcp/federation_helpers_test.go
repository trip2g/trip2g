package mcp

import (
	"context"
	"testing"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/logger"

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

func (m federationVerifyEnvMock) ClearFederationSecretPrev(context.Context, db.ClearFederationSecretPrevParams) error {
	return nil
}

func (m federationVerifyEnvMock) Logger() logger.Logger {
	return &logger.DummyLogger{}
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
			KBID:  "cellbio",
			Federation: &FederationRef{
				KBID:             "cellbio",
				KBURL:            "https://science.example/_system/mcp",
				AgentInstruction: "open remote",
			},
		},
	}

	prefixKBID("science", items)

	// An ordinary leaf note answered here is stamped with the local segment.
	require.Equal(t, "science", items[0].KBID)
	require.Nil(t, items[0].Federation)
	// A pointer note reported one hop down is prefixed into the caller's frame.
	require.Equal(t, "science/cellbio", items[1].KBID)
	require.Equal(t, "science/cellbio", items[1].Federation.KBID)
	require.Equal(t, "https://science.example/_system/mcp", items[1].Federation.KBURL)
	require.Equal(t, "open remote", items[1].Federation.AgentInstruction)
}

func TestSignOutboundVerifyInboundRoundTrip(t *testing.T) {
	secret := []byte("12345678901234567890123456789012")
	token, err := signOutbound(secret, "alice", "https://peer.example/_system/mcp", "rid-1", nil)
	require.NoError(t, err)

	auth, err := verifyInbound(context.Background(), federationVerifyEnvMock{
		secret:    db.FederationSecret{Kid: "alice", SecretCrypt: secret},
		secretOK:  true,
		subgraphs: []string{"team"},
	}, token, nil)

	require.NoError(t, err)
	require.Equal(t, "alice", auth.KID)
	require.Equal(t, []string{"team"}, auth.AllowedSubgraphs)
}

func TestVerifyInboundSentinels(t *testing.T) {
	secret := []byte("12345678901234567890123456789012")
	now := time.Now()

	valid, err := signOutbound(secret, "alice", "https://hub.local", "rid-1", nil)
	require.NoError(t, err)

	t.Run("unknown kid", func(t *testing.T) {
		//nolint:govet // err in closure shadows outer test err intentionally
		_, err := verifyInbound(context.Background(), federationVerifyEnvMock{}, valid, nil)
		require.ErrorIs(t, err, ErrFedAuthUnknownKid)
	})

	t.Run("bad signature", func(t *testing.T) {
		_, err := verifyInbound(context.Background(), federationVerifyEnvMock{ //nolint:govet // err in closure shadows outer test err intentionally
			secret:   db.FederationSecret{Kid: "alice", SecretCrypt: []byte("other-secret")},
			secretOK: true,
		}, valid, nil)
		require.ErrorIs(t, err, ErrFedAuthBadSig)
	})

	t.Run("expired", func(t *testing.T) {
		//nolint:govet // err in closure shadows outer test err intentionally
		token, err := signOutboundAt(secret, "alice", "https://hub.local", "rid-1", now.Add(-time.Minute), 30*time.Second, nil)
		require.NoError(t, err)

		_, err = verifyInbound(context.Background(), federationVerifyEnvMock{
			secret:   db.FederationSecret{Kid: "alice", SecretCrypt: secret},
			secretOK: true,
		}, token, nil)
		require.ErrorIs(t, err, ErrFedAuthExpired)
	})

	t.Run("future iat", func(t *testing.T) {
		//nolint:govet // err in closure shadows outer test err intentionally
		token, err := signOutboundAt(secret, "alice", "https://hub.local", "rid-1", now.Add(10*time.Second), 30*time.Second, nil)
		require.NoError(t, err)

		_, err = verifyInbound(context.Background(), federationVerifyEnvMock{
			secret:   db.FederationSecret{Kid: "alice", SecretCrypt: secret},
			secretOK: true,
		}, token, nil)
		require.ErrorIs(t, err, ErrFedAuthFutureIAT)
	})

	// No "revoked" case: FederationSecretByKID filters revoked rows, so a
	// revoked kid reaches verification as no row at all and answers unknown kid.
	// The branch that used to answer otherwise was reachable only from a mock
	// that ignored the where clause.
}
