package rotatefederationsecret_test

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"trip2g/internal/case/admin/rotatefederationsecret"
	"trip2g/internal/db"
	graphmodel "trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

// envStub is one stored pairing plus a record of what was asked of it.
// "Encryption" is a prefix so a test can read what would have been written.
type envStub struct {
	row      db.FederationSecret
	notFound bool

	peer    *peerStub
	rotated *db.RotateFederationSecretParams
	cleared []db.ClearFederationSecretPrevParams

	allowPlainHTTP bool
	rotateBlocked  bool
	// Every peer the case built, in order: the rotation first, then the probe.
	// Keeping only the last would hide which key each call was signed with.
	seenPeers []model.FederationPeer
}

func (e *envStub) OutboundFederationSecretByKID(context.Context, string) (db.FederationSecret, error) {
	if e.notFound {
		return db.FederationSecret{}, sql.ErrNoRows
	}
	return e.row, nil
}

func (e *envStub) RotateFederationSecret(_ context.Context, arg db.RotateFederationSecretParams) (int64, error) {
	e.rotated = &arg
	if e.rotateBlocked {
		return 0, nil
	}
	return 1, nil
}

func (e *envStub) ClearFederationSecretPrev(_ context.Context, arg db.ClearFederationSecretPrevParams) error {
	e.cleared = append(e.cleared, arg)
	return nil
}

func (e *envStub) FederationPeerClient(peer model.FederationPeer) model.Federation {
	e.seenPeers = append(e.seenPeers, peer)
	return e.peer
}

func (e *envStub) EncryptData(value []byte) ([]byte, error) {
	return append([]byte("enc:"), value...), nil
}

func (e *envStub) DecryptData(value []byte) ([]byte, error) {
	return value[len("enc:"):], nil
}

func (e *envStub) FederationAllowsPlainHTTP() bool { return e.allowPlainHTTP }
func (e *envStub) PublicURL() string               { return "https://hub.example" }
func (e *envStub) AuditLogger() logger.Logger      { return &logger.DummyLogger{} }

// peerStub answers a rotation and the probe that follows it.
type peerStub struct {
	model.Federation

	rotateErr    error
	rotateResult model.FederationResult
	proposed     []byte

	searchAnswers bool
}

func (p *peerStub) RotateSecret(_ context.Context, params model.MCPRotateSecretParams) (model.FederationResult, error) {
	if p.rotateErr != nil {
		return model.FederationResult{}, p.rotateErr
	}
	decoded, err := hex.DecodeString(params.SecretHex)
	if err != nil {
		return model.FederationResult{}, err
	}
	p.proposed = decoded
	return p.rotateResult, nil
}

func (p *peerStub) Search(context.Context, model.MCPSearchParams) (model.FederationResult, error) {
	if !p.searchAnswers {
		return model.FederationResult{}, errors.New("unreachable")
	}
	return model.FederationResult{}, nil
}

func liveRow() db.FederationSecret {
	kbURL := "https://peer.example/_system/mcp"
	return db.FederationSecret{
		ID:          7,
		Kid:         "peer",
		KbUrl:       &kbURL,
		SecretCrypt: []byte("enc:current-key"),
	}
}

func TestResolveStoresWhatThePeerAccepted(t *testing.T) {
	t.Parallel()

	env := &envStub{row: liveRow(), peer: &peerStub{searchAnswers: true}}

	payload, err := rotatefederationsecret.Resolve(context.Background(), env, "peer")

	require.NoError(t, err)
	require.IsType(t, &graphmodel.RotateFederationSecretPayload{}, payload)

	require.Equal(t, []byte("current-key"), env.seenPeers[0].Secret,
		"the rotation is authorised by the key the peer still holds")
	require.Equal(t, env.peer.proposed, env.seenPeers[1].Secret,
		"the probe has to prove the NEW key verifies")
	require.Empty(t, env.seenPeers[1].PrevSecret,
		"the probe fell back to the old key, so a peer that never rotated would confirm one that did")
	require.Len(t, env.peer.proposed, 32)
	require.NotNil(t, env.rotated)
	require.Equal(t, int64(7), env.rotated.ID)
	require.Equal(t, append([]byte("enc:"), env.peer.proposed...), env.rotated.NewSecretCrypt)
	require.Equal(t, []byte("enc:current-key"), env.rotated.ExpectedSecretCrypt)
}

// The probe is what closes the two-key window, and it names the stored
// ciphertext the rotation moved aside rather than the row id alone.
func TestResolveRetiresTheOldKeyWhenTheProbeAnswers(t *testing.T) {
	t.Parallel()

	env := &envStub{row: liveRow(), peer: &peerStub{searchAnswers: true}}

	_, err := rotatefederationsecret.Resolve(context.Background(), env, "peer")

	require.NoError(t, err)
	require.Equal(t, []db.ClearFederationSecretPrevParams{{ID: 7, PrevSecretCrypt: []byte("enc:current-key")}}, env.cleared)
}

// A probe that does not come back decides nothing: both keys stay, and the
// grace window retires the old one on its own.
func TestResolveKeepsBothKeysWhenTheProbeDoesNot(t *testing.T) {
	t.Parallel()

	env := &envStub{row: liveRow(), peer: &peerStub{searchAnswers: false}}

	_, err := rotatefederationsecret.Resolve(context.Background(), env, "peer")

	require.NoError(t, err)
	require.NotNil(t, env.rotated, "the rotation itself still happened")
	require.Empty(t, env.cleared, "a silent probe retired the key the peer may still be using")
}

// A peer that ANSWERED no demonstrably still holds the old key. Moving this side
// off it would leave the link running on the previous key until the grace closes
// and then dead — from a rotation that never happened.
func TestResolveRecordsNothingWhenThePeerRefuses(t *testing.T) {
	t.Parallel()

	env := &envStub{row: liveRow(), peer: &peerStub{
		rotateResult: model.FederationResult{
			IsError: true,
			Content: []model.FederationContent{{Type: "text", Text: "Method not found: rotate_secret"}},
		},
	}}

	payload, err := rotatefederationsecret.Resolve(context.Background(), env, "peer")

	require.NoError(t, err)
	require.IsType(t, &graphmodel.ErrorPayload{}, payload)
	require.Nil(t, env.rotated, "this side moved off a key the peer said it kept")
}

// The real transport answers a refusal as a typed JSON-RPC or HTTP error,
// never as an IsError tool result. An answer that precedes execution proves
// the peer still holds only the old key; anything ambiguous — an internal
// error, a 5xx, an unknown code — may have come after the peer committed, so
// it stays on the silence path.
func TestResolveClassifiesAnswers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rotateErr error
		recorded  bool
	}{
		{"parse error is a refusal", &model.FederationRPCError{Code: -32700, Message: "answered"}, false},
		{"invalid request is a refusal", &model.FederationRPCError{Code: -32600, Message: "answered"}, false},
		{"method not found is a refusal", &model.FederationRPCError{Code: -32601, Message: "answered"}, false},
		{"invalid params is a refusal", &model.FederationRPCError{Code: -32602, Message: "answered"}, false},
		{"auth failure is a refusal", &model.FederationRPCError{Code: model.FederationAuthErrorCode, Message: "answered"}, false},
		{"a wrapped refusal is still a refusal", fmt.Errorf("call rotate_secret: %w", &model.FederationRPCError{Code: -32601, Message: "answered"}), false},
		{"internal error is silence", &model.FederationRPCError{Code: -32603, Message: "answered"}, true},
		{"an unknown code is silence", &model.FederationRPCError{Code: -32000, Message: "answered"}, true},
		{"http not found is a refusal", &model.FederationHTTPError{Status: 404}, false},
		{"http method not allowed is a refusal", &model.FederationHTTPError{Status: 405}, false},
		{"http bad gateway is silence", &model.FederationHTTPError{Status: 502}, true},
		{"http too many requests is silence", &model.FederationHTTPError{Status: 429}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := &envStub{row: liveRow(), peer: &peerStub{rotateErr: tt.rotateErr}}

			payload, err := rotatefederationsecret.Resolve(context.Background(), env, "peer")

			require.NoError(t, err)
			errPayload, ok := payload.(*graphmodel.ErrorPayload)
			require.True(t, ok)
			require.Empty(t, env.cleared, "nothing was confirmed, so nothing may be retired")
			if tt.recorded {
				require.NotNil(t, env.rotated, "the peer may have committed before the answer was lost")
			} else {
				require.Nil(t, env.rotated, "the peer answered without executing, so it still holds only the old key")
				require.Contains(t, errPayload.Message, tt.rotateErr.Error(),
					"the operator acts on WHY the peer said no")
			}
		})
	}
}

// Silence is the other outcome and needs the opposite treatment: the peer may
// have committed before the answer was lost, so the proposal is kept and a retry
// re-proposes it.
func TestResolveKeepsTheProposalWhenThePeerIsSilent(t *testing.T) {
	t.Parallel()

	env := &envStub{row: liveRow(), peer: &peerStub{rotateErr: errors.New("timeout")}}

	payload, err := rotatefederationsecret.Resolve(context.Background(), env, "peer")

	require.NoError(t, err)
	require.IsType(t, &graphmodel.ErrorPayload{}, payload)
	require.NotNil(t, env.rotated, "the peer may hold a key this side never wrote down")
}

func TestResolveAnswersForAnUnknownKid(t *testing.T) {
	t.Parallel()

	env := &envStub{notFound: true}

	payload, err := rotatefederationsecret.Resolve(context.Background(), env, "ghost")

	require.NoError(t, err)
	errPayload, ok := payload.(*graphmodel.ErrorPayload)
	require.True(t, ok)
	require.Contains(t, errPayload.Message, "ghost")
}

// A proposal older than the grace window is not a retry any more: the peer has
// stopped accepting that key, so this is a fresh rotation and the expired key is
// not carried into it.
func TestResolveStartsFreshAfterTheGrace(t *testing.T) {
	t.Parallel()

	row := liveRow()
	rotatedAt := time.Now().Add(-model.RotationGrace - time.Minute)
	row.PrevSecretCrypt = []byte("enc:older-key")
	row.RotatedAt = &rotatedAt

	env := &envStub{row: row, peer: &peerStub{searchAnswers: true}}

	_, err := rotatefederationsecret.Resolve(context.Background(), env, "peer")

	require.NoError(t, err)
	require.NotNil(t, env.rotated, "an expired proposal was re-proposed instead of replaced")
	require.NotEqual(t, []byte("older-key"), env.peer.proposed, "an expired proposal was re-proposed")
	require.Empty(t, env.seenPeers[0].PrevSecret, "an expired key was still offered to the peer")
}

// Rotation puts a fresh key on the wire. Over plain http to a stranger it would
// move the secret from one channel nobody controls to another, while leaving the
// operator believing the first one is now safe.
func TestProposeRefusesPlainHTTP(t *testing.T) {
	t.Parallel()

	env := &envStub{peer: &peerStub{}}
	peer := model.FederationPeer{KBURL: "http://peer.example/_system/mcp", KID: "peer", Secret: []byte("k")}

	err := rotatefederationsecret.Propose(context.Background(), env, peer, make([]byte, 32))

	require.ErrorIs(t, err, rotatefederationsecret.ErrInsecureURL)
	require.Nil(t, env.peer.proposed, "a key was put on the wire the guard was meant to stop")
}

// Unless the deployment has already said it federates over addresses that are
// not on the public internet, where there is no third party on the path.
func TestProposeAllowsPlainHTTPWhereTheDeploymentSaysSo(t *testing.T) {
	t.Parallel()

	env := &envStub{peer: &peerStub{}, allowPlainHTTP: true}
	peer := model.FederationPeer{KBURL: "http://peer.example/_system/mcp", KID: "peer", Secret: []byte("k")}

	err := rotatefederationsecret.Propose(context.Background(), env, peer, make([]byte, 32))

	require.NoError(t, err)
	require.Len(t, env.peer.proposed, 32)
}

// Two mints must not produce the same key.
func TestNewSecretIsFreshEachTime(t *testing.T) {
	t.Parallel()

	first, err := rotatefederationsecret.NewSecret()
	require.NoError(t, err)

	second, err := rotatefederationsecret.NewSecret()
	require.NoError(t, err)

	require.Len(t, first, 32)
	require.NotEqual(t, first, second)
}

// And a retry re-proposes that same key rather than minting another, which is
// what a peer that already applied it answers as a no-op.
func TestResolveReproposesAStagedKey(t *testing.T) {
	t.Parallel()

	row := liveRow()
	rotatedAt := time.Now()
	row.SecretCrypt = []byte("enc:staged-key")
	row.PrevSecretCrypt = []byte("enc:older-key")
	row.RotatedAt = &rotatedAt

	env := &envStub{row: row, peer: &peerStub{searchAnswers: true}}

	_, err := rotatefederationsecret.Resolve(context.Background(), env, "peer")

	require.NoError(t, err)
	require.Nil(t, env.rotated, "a retry minted a second key instead of re-proposing the staged one")
	require.Equal(t, []byte("staged-key"), env.peer.proposed)
	require.Equal(t, []byte("older-key"), env.seenPeers[0].PrevSecret,
		"the retry must still reach a peer that never learned of the staged key")
	require.Equal(t, []byte("staged-key"), env.seenPeers[0].Secret)
}

// A rotation that raced this one has already moved the row. The peer is asked
// first — that ordering is what lets a refusal cost nothing — so by the time the
// race is visible the peer may hold this key too. What must not happen is
// writing over the row blind: the winner's key would be lost and the two sides
// would hold no key in common. The loser is told, and the pairing keeps working
// on the keys the winner recorded.
func TestResolveRefusesToRaceAnotherRotation(t *testing.T) {
	t.Parallel()

	env := &envStub{row: liveRow(), peer: &peerStub{}, rotateBlocked: true}

	payload, err := rotatefederationsecret.Resolve(context.Background(), env, "peer")

	require.NoError(t, err)
	errPayload, ok := payload.(*graphmodel.ErrorPayload)
	require.True(t, ok)
	require.Contains(t, errPayload.Message, "in flight")
	require.Empty(t, env.cleared, "the loser retired a key it does not know the state of")
}
