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

// A pointer's generated agent_instruction names the bare id; the hop regenerates
// it for the composed one. A custom instruction is left alone.
func TestPrefixKBIDRegeneratesAgentInstruction(t *testing.T) {
	items := []SearchResultItem{{
		Title: "pointer",
		KBID:  "cellbio",
		Federation: &FederationRef{
			KBID:             "cellbio",
			AgentInstruction: federationAgentInstruction("cellbio"),
		},
	}}

	prefixKBID("science", items)

	require.Equal(t, federationAgentInstruction("science/cellbio"), items[0].Federation.AgentInstruction)
}

func TestNotConfiguredMessage(t *testing.T) {
	tests := []struct {
		name        string
		status      FederationStatusPayload
		want        []string
		wantMissing []string
	}{
		{
			name:   "no bases at all",
			status: FederationStatusPayload{},
			want:   []string{"Federation is not configured for this hub", "mcp_federation_kb_url"},
		},
		{
			name:   "kb_ids selection matched nothing",
			status: FederationStatusPayload{ConnectedKBIDs: []string{"a", "b"}},
			want:   []string{"None of the requested kb_ids", "Connected bases: a, b"},
		},
		{
			name:   "flat id, no local peer of that name",
			status: FederationStatusPayload{KBID: "montaigne", ConnectedKBIDs: []string{"philosophers"}},
			want: []string{
				`Federation is not configured for kb_id "montaigne"`,
				`no connected base on this hub is named "montaigne"`,
				"Connected bases: philosophers",
				"<hub>/montaigne",
			},
			wantMissing: []string{"philosophers/montaigne"},
		},
		{
			name:   "unknown head, a later segment is a local peer",
			status: FederationStatusPayload{KBID: "trip2g/markavrelii", ConnectedKBIDs: []string{"markavrelii", "philosophers"}},
			want: []string{
				`no connected base on this hub is named "trip2g"`,
				`"markavrelii" is a connected base — address it as kb_id="markavrelii"`,
			},
			wantMissing: []string{"<hub>/", `address this base as`},
		},
		{
			name:   "unknown head with no bases at all",
			status: FederationStatusPayload{KBID: "ghost"},
			want:   []string{`Federation is not configured for kb_id "ghost"`, "this hub has no connected bases"},
		},
		{
			name: "miss reported by a hub, with its bases",
			status: FederationStatusPayload{
				KBID:           "philosophers/marcus-aurelius",
				Hub:            "philosophers",
				ConnectedKBIDs: []string{"philosophers/epictetus"},
			},
			want: []string{
				`hub "philosophers" has no base "marcus-aurelius"`,
				`Bases connected under "philosophers": philosophers/epictetus`,
			},
			wantMissing: []string{"<hub>/", `federated_search(kb_id="philosophers", query=`},
		},
		{
			name: "miss reported by a hub, a later segment is one of its bases",
			status: FederationStatusPayload{
				KBID:           "philosophers/hub/epictetus",
				Hub:            "philosophers",
				ConnectedKBIDs: []string{"philosophers/epictetus"},
			},
			want: []string{
				`hub "philosophers" has no base "hub"`,
				`"philosophers/epictetus" is a connected base — address it as kb_id="philosophers/epictetus"`,
			},
		},
		{
			name:   "miss reported by a hub that sent no list",
			status: FederationStatusPayload{KBID: "philosophers/ghost", Hub: "philosophers"},
			want: []string{
				`hub "philosophers" has no base "ghost"`,
				`federated_search(kb_id="philosophers", query="ghost")`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := notConfiguredMessage(tt.status)
			for _, w := range tt.want {
				require.Contains(t, got, w)
			}
			for _, w := range tt.wantMissing {
				require.NotContains(t, got, w)
			}
		})
	}
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
