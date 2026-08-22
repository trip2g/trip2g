package rotatefederationsecret_test

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
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

func (e *envStub) RotateFederationSecret(_ context.Context, arg db.RotateFederationSecretParams) error {
	e.rotated = &arg
	return nil
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
		"the rotation is authorised by the key being replaced")
	require.Equal(t, env.peer.proposed, env.seenPeers[1].Secret,
		"the probe has to prove the NEW key verifies")
	require.Empty(t, env.seenPeers[1].PrevSecret,
		"the probe fell back to the old key, so a peer that never rotated would confirm one that did")
	require.Len(t, env.peer.proposed, 32)
	require.NotNil(t, env.rotated)
	require.Equal(t, int64(7), env.rotated.ID)
	require.Equal(t, append([]byte("enc:"), env.peer.proposed...), env.rotated.SecretCrypt)
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

func TestResolveAnswersWhenThePeerRefuses(t *testing.T) {
	t.Parallel()

	env := &envStub{row: liveRow(), peer: &peerStub{rotateErr: errors.New("connection refused")}}

	payload, err := rotatefederationsecret.Resolve(context.Background(), env, "peer")

	require.NoError(t, err)
	require.IsType(t, &graphmodel.ErrorPayload{}, payload)
	require.Nil(t, env.rotated, "the key moved on a rotation the peer never took")
}

// The peer answering with an error result is a refusal too — an old peer that
// does not know the tool replies method-not-found rather than failing the
// transport.
func TestResolveAnswersWhenThePeerReportsAnError(t *testing.T) {
	t.Parallel()

	env := &envStub{row: liveRow(), peer: &peerStub{
		rotateResult: model.FederationResult{
			IsError: true,
			Content: []model.FederationContent{{Type: "text", Text: "Method not found: rotate_secret"}},
		},
	}}

	payload, err := rotatefederationsecret.Resolve(context.Background(), env, "peer")

	require.NoError(t, err)
	errPayload, ok := payload.(*graphmodel.ErrorPayload)
	require.True(t, ok)
	require.Contains(t, errPayload.Message, "Method not found")
	require.Nil(t, env.rotated)
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

// A rotation whose response was lost left this side still calling the old key
// current. Carrying both is what lets the next rotation reach a peer that has
// already moved on.
func TestResolveCarriesThePreviousKeyInsideTheGrace(t *testing.T) {
	t.Parallel()

	row := liveRow()
	rotatedAt := time.Now()
	row.PrevSecretCrypt = []byte("enc:older-key")
	row.RotatedAt = &rotatedAt

	env := &envStub{row: row, peer: &peerStub{searchAnswers: true}}

	_, err := rotatefederationsecret.Resolve(context.Background(), env, "peer")

	require.NoError(t, err)
	require.Equal(t, []byte("older-key"), env.seenPeers[0].PrevSecret)
}

// Outside the window the peer has stopped accepting it too, so offering it
// would buy a refused request per call and nothing else.
func TestResolveDropsThePreviousKeyAfterTheGrace(t *testing.T) {
	t.Parallel()

	row := liveRow()
	rotatedAt := time.Now().Add(-model.RotationGrace - time.Minute)
	row.PrevSecretCrypt = []byte("enc:older-key")
	row.RotatedAt = &rotatedAt

	env := &envStub{row: row, peer: &peerStub{searchAnswers: true}}

	_, err := rotatefederationsecret.Resolve(context.Background(), env, "peer")

	require.NoError(t, err)
	require.Empty(t, env.seenPeers[0].PrevSecret)
}

// Rotation puts a fresh key on the wire. Over plain http to a stranger it would
// move the secret from one channel nobody controls to another, while leaving the
// operator believing the first one is now safe.
func TestExchangeRefusesPlainHTTP(t *testing.T) {
	t.Parallel()

	env := &envStub{peer: &peerStub{}}
	peer := model.FederationPeer{KBURL: "http://peer.example/_system/mcp", KID: "peer", Secret: []byte("k")}

	_, err := rotatefederationsecret.Exchange(context.Background(), env, peer)

	require.ErrorIs(t, err, rotatefederationsecret.ErrInsecureURL)
	require.Nil(t, env.peer.proposed, "a key was put on the wire the guard was meant to stop")
}

// Unless the deployment has already said it federates over addresses that are
// not on the public internet, where there is no third party on the path.
func TestExchangeAllowsPlainHTTPWhereTheDeploymentSaysSo(t *testing.T) {
	t.Parallel()

	env := &envStub{peer: &peerStub{}, allowPlainHTTP: true}
	peer := model.FederationPeer{KBURL: "http://peer.example/_system/mcp", KID: "peer", Secret: []byte("k")}

	secret, err := rotatefederationsecret.Exchange(context.Background(), env, peer)

	require.NoError(t, err)
	require.Len(t, secret, 32)
}

// Two rotations must not propose the same key.
func TestExchangeMintsAFreshKeyEachTime(t *testing.T) {
	t.Parallel()

	env := &envStub{peer: &peerStub{}}
	peer := model.FederationPeer{KBURL: "https://peer.example/_system/mcp", KID: "peer", Secret: []byte("k")}

	first, err := rotatefederationsecret.Exchange(context.Background(), env, peer)
	require.NoError(t, err)

	second, err := rotatefederationsecret.Exchange(context.Background(), env, peer)
	require.NoError(t, err)

	require.NotEqual(t, first, second)
}
