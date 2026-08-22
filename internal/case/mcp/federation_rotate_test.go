package mcp

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

// rotateEnvStub is the base's side of a rotation: one secret row, plaintext
// "encryption" so a test can read what was stored, and a record of what it was
// asked to do. Everything the handler must not touch panics.
type rotateEnvStub struct {
	Env
	row         db.FederationSecret
	rotated     *db.RotateFederationSecretParams
	cleared     []int64
	clearedWith []db.ClearFederationSecretPrevParams
	clearErr    error
	notFound    bool
}

func (e *rotateEnvStub) FederationSecretByID(context.Context, int64) (db.FederationSecret, error) {
	if e.notFound {
		return db.FederationSecret{}, sql.ErrNoRows
	}
	return e.row, nil
}

func (e *rotateEnvStub) FederationSecretByKID(context.Context, string) (db.FederationSecret, bool, error) {
	if e.notFound {
		return db.FederationSecret{}, false, nil
	}
	return e.row, true, nil
}

func (e *rotateEnvStub) ListFederationSecretSubgraphsByKID(context.Context, string) ([]string, error) {
	return nil, nil
}

func (e *rotateEnvStub) RotateFederationSecret(_ context.Context, arg db.RotateFederationSecretParams) (int64, error) {
	e.rotated = &arg
	return 1, nil
}

func (e *rotateEnvStub) ClearFederationSecretPrev(_ context.Context, arg db.ClearFederationSecretPrevParams) error {
	e.clearedWith = append(e.clearedWith, arg)
	if e.clearErr != nil {
		return e.clearErr
	}
	e.cleared = append(e.cleared, arg.ID)
	return nil
}

func (e *rotateEnvStub) Logger() logger.Logger { return &logger.DummyLogger{} }

func (e *rotateEnvStub) Features() features.Features              { return features.Features{} }
func (e *rotateEnvStub) FederationMaxDepth() int                  { return 3 }
func (e *rotateEnvStub) LatestNoteViews() *model.NoteViews        { return model.NewNoteViews() }
func (e *rotateEnvStub) FederatedGraphQLEnabled() bool            { return true }
func (e *rotateEnvStub) DecryptData(value []byte) ([]byte, error) { return value, nil }
func (e *rotateEnvStub) EncryptData(value []byte) ([]byte, error) { return value, nil }
func (e *rotateEnvStub) AuditLogger() logger.Logger               { return &logger.DummyLogger{} }

func rotateArgs(t *testing.T, secret []byte) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(RotateSecretArguments{SecretHex: hex.EncodeToString(secret)})
	require.NoError(t, err)
	return raw
}

func boundAuth(secretID int64) context.Context {
	return contextWithFederationAuth(context.Background(), federationAuth{
		KID:       "peer",
		SecretID:  secretID,
		BodyBound: true,
	})
}

func TestRotateSecretReplacesTheKey(t *testing.T) {
	current := []byte("00000000000000000000000000000001")
	fresh := []byte("00000000000000000000000000000002")
	env := &rotateEnvStub{row: db.FederationSecret{ID: 7, Kid: "peer", SecretCrypt: current}}

	resp := handleRotateSecret(boundAuth(7), env, 1, rotateArgs(t, fresh))

	require.Nil(t, resp.Error)
	require.NotNil(t, env.rotated, "the key was not stored")
	require.Equal(t, int64(7), env.rotated.ID)
	require.Equal(t, fresh, env.rotated.NewSecretCrypt)
	require.Equal(t, current, env.rotated.ExpectedSecretCrypt)
}

// A peer whose response was lost retries with the same key. Answering success
// without shifting the previous one again is what makes the retry safe to
// repeat — otherwise the second attempt would push the key the peer still holds
// out of the pairing entirely.
func TestRotateSecretIsIdempotent(t *testing.T) {
	current := []byte("00000000000000000000000000000001")
	env := &rotateEnvStub{row: db.FederationSecret{ID: 7, Kid: "peer", SecretCrypt: current}}

	resp := handleRotateSecret(boundAuth(7), env, 1, rotateArgs(t, current))

	require.Nil(t, resp.Error)
	require.Nil(t, env.rotated, "a repeat of the same key rotated the pairing a second time")
}

// The key is in the arguments, so a signature that does not cover them leaves it
// replaceable in flight for the whole 30-second window.
func TestRotateSecretRefusesAnUnboundBody(t *testing.T) {
	env := &rotateEnvStub{row: db.FederationSecret{ID: 7, Kid: "peer", SecretCrypt: []byte("x")}}
	ctx := contextWithFederationAuth(context.Background(), federationAuth{KID: "peer", SecretID: 7})

	resp := handleRotateSecret(ctx, env, 1, rotateArgs(t, []byte("00000000000000000000000000000002")))

	require.NotNil(t, resp.Error)
	require.Nil(t, env.rotated)
}

// To anyone who is not a peer the tool does not exist — the same answer as an
// unknown method, so a caller outside the federation learns nothing about what
// this endpoint can do.
func TestRotateSecretIsInvisibleWithoutFederationAuth(t *testing.T) {
	env := &rotateEnvStub{row: db.FederationSecret{ID: 7, Kid: "peer", SecretCrypt: []byte("x")}}

	resp := handleRotateSecret(context.Background(), env, 1, rotateArgs(t, []byte("00000000000000000000000000000002")))

	require.NotNil(t, resp.Error)
	require.Equal(t, ErrCodeMethodNotFound, resp.Error.Code)
	require.Nil(t, env.rotated)
}

func TestRotateSecretRefusesAKeyOfTheWrongShape(t *testing.T) {
	env := &rotateEnvStub{row: db.FederationSecret{ID: 7, Kid: "peer", SecretCrypt: []byte("x")}}

	for _, given := range []string{"", "abcd", hex.EncodeToString(make([]byte, 32))} {
		raw, err := json.Marshal(RotateSecretArguments{SecretHex: given})
		require.NoError(t, err)

		resp := handleRotateSecret(boundAuth(7), env, 1, raw)

		require.NotNil(t, resp.Error, "accepted %q", given)
		require.Nil(t, env.rotated)
	}
}

// The two-key window: a peer that has not yet learned of a rotation still calls
// with the key it was given, and that call has to work or the rotation would
// break a link that was fine.
func TestVerifyInboundAcceptsThePreviousKeyInsideTheGrace(t *testing.T) {
	current := []byte("00000000000000000000000000000002")
	previous := []byte("00000000000000000000000000000001")
	rotatedAt := time.Now()

	token, err := signOutbound(previous, "peer", "https://hub.local", "rid", nil)
	require.NoError(t, err)

	env := &rotateEnvStub{row: db.FederationSecret{
		ID:              7,
		Kid:             "peer",
		SecretCrypt:     current,
		PrevSecretCrypt: previous,
		RotatedAt:       &rotatedAt,
	}}

	auth, err := verifyInbound(context.Background(), env, token, nil)

	require.NoError(t, err)
	require.Equal(t, "peer", auth.KID)
	require.Empty(t, env.cleared, "the previous key was retired by a call that used it")
}

// And stops working once the window closes, which is what keeps the key that
// travelled through a chat from outliving the rotation that replaced it.
func TestVerifyInboundRefusesThePreviousKeyAfterTheGrace(t *testing.T) {
	current := []byte("00000000000000000000000000000002")
	previous := []byte("00000000000000000000000000000001")
	rotatedAt := time.Now().Add(-model.RotationGrace - time.Minute)

	token, err := signOutbound(previous, "peer", "https://hub.local", "rid", nil)
	require.NoError(t, err)

	env := &rotateEnvStub{row: db.FederationSecret{
		ID:              7,
		Kid:             "peer",
		SecretCrypt:     current,
		PrevSecretCrypt: previous,
		RotatedAt:       &rotatedAt,
	}}

	_, err = verifyInbound(context.Background(), env, token, nil)

	require.ErrorIs(t, err, ErrFedAuthBadSig)
}

// The peer signed with the current key, so it holds it, so the old one covers
// nothing. Retiring it here is what keeps the two-key state to the moments
// around a rotation rather than the whole window.
func TestVerifyInboundRetiresThePreviousKeyOnASuccessfulCall(t *testing.T) {
	current := []byte("00000000000000000000000000000002")
	previous := []byte("00000000000000000000000000000001")
	rotatedAt := time.Now()

	token, err := signOutbound(current, "peer", "https://hub.local", "rid", nil)
	require.NoError(t, err)

	env := &rotateEnvStub{row: db.FederationSecret{
		ID:              7,
		Kid:             "peer",
		SecretCrypt:     current,
		PrevSecretCrypt: previous,
		RotatedAt:       &rotatedAt,
	}}

	_, err = verifyInbound(context.Background(), env, token, nil)

	require.NoError(t, err)
	require.Equal(t, []int64{7}, env.cleared)
}

func TestVerifyInboundChecksTheBodyWhenTheSignatureCoversIt(t *testing.T) {
	secret := []byte("00000000000000000000000000000001")
	body := []byte(`{"jsonrpc":"2.0"}`)

	token, err := signOutbound(secret, "peer", "https://hub.local", "rid", body)
	require.NoError(t, err)

	env := &rotateEnvStub{row: db.FederationSecret{ID: 7, Kid: "peer", SecretCrypt: secret}}

	auth, err := verifyInbound(context.Background(), env, token, body)
	require.NoError(t, err)
	require.True(t, auth.BodyBound)

	_, err = verifyInbound(context.Background(), env, token, []byte(`{"jsonrpc":"2.0","tampered":1}`))
	require.ErrorIs(t, err, ErrFedAuthBodyChanged)
}

// A peer from before body binding sends no bh and is authenticated as it always
// was. Only the calls that need the guarantee insist on it.
func TestVerifyInboundAcceptsAPeerThatBindsNothing(t *testing.T) {
	secret := []byte("00000000000000000000000000000001")
	env := &rotateEnvStub{row: db.FederationSecret{ID: 7, Kid: "peer", SecretCrypt: secret}}

	token, err := signOutbound(secret, "peer", "https://hub.local", "rid", nil)
	require.NoError(t, err)

	auth, err := verifyInbound(context.Background(), env, token, []byte("anything at all"))

	require.NoError(t, err)
	require.False(t, auth.BodyBound)
}

// tools/list is the contract third-party adapters mirror and the menu an LLM
// agent reads. Rotation is control-plane and belongs in neither: an advertised
// rotation tool is one a model will reach for on its own initiative.
func TestRotateSecretIsNotAdvertised(t *testing.T) {
	env := &rotateEnvStub{}

	for _, tool := range builtinTools(contextWithMCPAPIKeyAuth(context.Background(), true), env) {
		require.NotEqual(t, model.RotateSecretTool, tool.Name, "rotation is advertised in tools/list")
	}
}

// The clear names the key this request actually saw, so a rotation that lands
// between the read and the write leaves a different previous key staged and the
// stale clear does nothing.
func TestVerifyInboundRetiresOnlyTheKeyItObserved(t *testing.T) {
	current := []byte("00000000000000000000000000000002")
	previous := []byte("00000000000000000000000000000001")
	rotatedAt := time.Now()

	token, err := signOutbound(current, "peer", "https://hub.local", "rid", nil)
	require.NoError(t, err)

	env := &rotateEnvStub{row: db.FederationSecret{
		ID:              7,
		Kid:             "peer",
		SecretCrypt:     current,
		PrevSecretCrypt: previous,
		RotatedAt:       &rotatedAt,
	}}

	_, err = verifyInbound(context.Background(), env, token, nil)

	require.NoError(t, err)
	require.Equal(t, []db.ClearFederationSecretPrevParams{{ID: 7, PrevSecretCrypt: previous}}, env.clearedWith)
}

// Retiring the old key is housekeeping. A search that happens to fall inside a
// grace window must not come back as an authentication failure because a
// bookkeeping write did not land.
func TestVerifyInboundSurvivesAFailedRetire(t *testing.T) {
	current := []byte("00000000000000000000000000000002")
	previous := []byte("00000000000000000000000000000001")
	rotatedAt := time.Now()

	token, err := signOutbound(current, "peer", "https://hub.local", "rid", nil)
	require.NoError(t, err)

	env := &rotateEnvStub{
		clearErr: errors.New("database is locked"),
		row: db.FederationSecret{
			ID:              7,
			Kid:             "peer",
			SecretCrypt:     current,
			PrevSecretCrypt: previous,
			RotatedAt:       &rotatedAt,
		},
	}

	auth, err := verifyInbound(context.Background(), env, token, nil)

	require.NoError(t, err)
	require.Equal(t, "peer", auth.KID)
}
