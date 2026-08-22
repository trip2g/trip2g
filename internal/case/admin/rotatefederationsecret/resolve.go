// Package rotatefederationsecret replaces the key a pairing shares with a peer.
//
// One primitive with two callers: installing an outbound secret runs it once so
// the key that arrived out of band — through a chat, usually, relayed by whoever
// was asked to pass it on — stops being the live credential; an operator runs it
// again whenever they want.
//
// See docs/dev/federation_key_rotation.md.
package rotatefederationsecret

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"trip2g/internal/db"
	graphmodel "trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

type Payload = graphmodel.RotateFederationSecretOrErrorPayload

// SecretBytes is the size of a federation key.
const SecretBytes = 32

// ErrPeerRefused means the peer answered and said no — most often a peer too old
// to know the call, or an adapter that never will. It definitively does not hold
// the new key, so nothing is recorded here: recording it would leave this side
// signing with a key the peer cannot verify, and the link would die when the
// grace on the old one closes.
var ErrPeerRefused = errors.New("the peer refused the new key")

// ErrPeerSilent means no answer came back. That is NOT the same as a refusal: a
// response lost after the peer committed looks exactly like a peer that never
// heard the request. The proposal is recorded on this outcome precisely because
// the peer may be holding it, and a retry re-proposes the same key — which a
// peer that already applied it answers as a no-op.
var ErrPeerSilent = errors.New("the peer did not answer")

// ErrInsecureURL is separate because the fix is different in kind — not "try
// again" but "this deployment has decided it federates over the open internet,
// so the address has to be https".
var ErrInsecureURL = errors.New("rotation needs an https peer address")

// ErrRotationInFlight means the row moved between being read and being written,
// which is another rotation of the same pairing running at the same time.
// Retrying is safe; racing it would leave the two sides holding different pairs.
var ErrRotationInFlight = errors.New("another rotation of this pairing is in flight")

type Env interface {
	OutboundFederationSecretByKID(ctx context.Context, kid string) (db.FederationSecret, error)
	RotateFederationSecret(ctx context.Context, arg db.RotateFederationSecretParams) (int64, error)
	ClearFederationSecretPrev(ctx context.Context, arg db.ClearFederationSecretPrevParams) error
	FederationPeerClient(peer model.FederationPeer) model.Federation
	EncryptData([]byte) ([]byte, error)
	DecryptData([]byte) ([]byte, error)
	FederationAllowsPlainHTTP() bool
	PublicURL() string
	AuditLogger() logger.Logger
}

// NewSecret mints a key. Random, never derived from the one it replaces: a
// derived key would be computable by anyone holding the key that travelled out
// of band, which is what rotation exists to defeat.
func NewSecret() ([]byte, error) {
	secret := make([]byte, SecretBytes)

	_, err := rand.Read(secret)
	if err != nil {
		return nil, fmt.Errorf("generate federation secret: %w", err)
	}

	return secret, nil
}

// Propose asks the peer to adopt a key, signed as the peer currently expects.
//
// It writes nothing, and it distinguishes the two ways of not succeeding: a peer
// that answered no still holds the old key, a peer that said nothing may hold
// either. What the caller records depends on which, so the two cannot share an
// error.
func Propose(ctx context.Context, env Env, peer model.FederationPeer, secret []byte) error {
	if !env.FederationAllowsPlainHTTP() && !strings.HasPrefix(peer.KBURL, "https://") {
		return fmt.Errorf("%w: %s", ErrInsecureURL, peer.KBURL)
	}

	peer.Issuer = env.PublicURL()
	params := model.MCPRotateSecretParams{SecretHex: hex.EncodeToString(secret)}

	result, err := env.FederationPeerClient(peer).RotateSecret(ctx, params)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrPeerSilent, err)
	}
	if result.IsError {
		return fmt.Errorf("%w: %s", ErrPeerRefused, firstText(result))
	}

	return nil
}

// Resolve rotates a pairing that is already installed.
//
// What is recorded turns on what the peer said, because the three answers carry
// different information. It took the key: record it. It answered and refused:
// record nothing, because it demonstrably still holds the old key and moving
// this side off it would kill the link when the grace closes. It said nothing:
// record the proposal, because the peer may be holding it — and a retry then
// re-proposes the SAME key, which a peer that already applied it answers as a
// no-op. Minting a fresh key per attempt instead would mean one lost response
// leaves nobody holding what the peer has.
func Resolve(ctx context.Context, env Env, kid string) (Payload, error) {
	row, err := env.OutboundFederationSecretByKID(ctx, kid)
	if db.IsNoFound(err) {
		return &graphmodel.ErrorPayload{Message: fmt.Sprintf("no live outbound federation secret with kid %q", kid)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get outbound federation secret: %w", err)
	}

	peer, err := peerFromRow(env, row)
	if err != nil {
		return nil, err
	}

	// A staged proposal is one whose previous key is still held: nothing has
	// confirmed the peer took it. Re-proposing that same key is what turns a
	// lost response into a retry instead of a second, different rotation.
	proposal := peer.Secret
	if !staged(row) {
		proposal, err = NewSecret()
		if err != nil {
			return nil, err
		}
	}

	err = Propose(ctx, env, peer, proposal)
	if errors.Is(err, ErrPeerRefused) || errors.Is(err, ErrInsecureURL) {
		return &graphmodel.ErrorPayload{Message: err.Error()}, nil
	}

	silent := errors.Is(err, ErrPeerSilent)
	if err != nil && !silent {
		return nil, err
	}

	if !staged(row) {
		peer, err = record(ctx, env, row, peer, proposal)
		if errors.Is(err, ErrRotationInFlight) {
			return &graphmodel.ErrorPayload{Message: err.Error()}, nil
		}
		if err != nil {
			return nil, err
		}
	}

	if silent {
		return &graphmodel.ErrorPayload{Message: ErrPeerSilent.Error() + "; the key is kept so a retry re-proposes it"}, nil
	}

	env.AuditLogger().Info("federation secret rotated", "kid", kid, "secretID", row.ID, "kbURL", *row.KbUrl)

	Confirm(ctx, env, row.ID, retiredCrypt(row), peer)

	return &graphmodel.RotateFederationSecretPayload{Kid: kid}, nil
}

// record makes the proposal this side's current key, keeping the one the peer
// may still hold as the previous. Returns the peer to speak to afterwards.
func record(
	ctx context.Context,
	env Env,
	row db.FederationSecret,
	peer model.FederationPeer,
	secret []byte,
) (model.FederationPeer, error) {
	encrypted, err := env.EncryptData(secret)
	if err != nil {
		return peer, fmt.Errorf("encrypt federation secret: %w", err)
	}

	rotateParams := db.RotateFederationSecretParams{
		NewSecretCrypt:      encrypted,
		ID:                  row.ID,
		ExpectedSecretCrypt: row.SecretCrypt,
	}

	// Conditional on the row this call read. A concurrent rotation of the same
	// pairing has already moved it, and writing over that blind would leave the
	// two sides holding different pairs of keys with no way back.
	affected, err := env.RotateFederationSecret(ctx, rotateParams)
	if err != nil {
		return peer, fmt.Errorf("stage rotated federation secret: %w", err)
	}
	if affected == 0 {
		return peer, ErrRotationInFlight
	}

	peer.PrevSecret = peer.Secret
	peer.Secret = secret

	return peer, nil
}

// Confirm asks the peer one cheap question with the new key and, if it answers,
// drops the key the pairing rotated away from.
//
// The probe carries the new key ALONE. An ordinary client falls back to the
// previous key on an auth refusal, which is right for traffic and wrong here:
// the fallback would let a peer that never applied the rotation answer
// successfully, and this would then retire the only key that peer still accepts.
// What is being proved is specifically that the NEW key verifies.
//
// Best effort otherwise, and never a decision the other way: a probe that does
// not come back leaves both keys in place, because it cannot tell a peer that
// refused from one that was busy for a second. The grace window retires the old
// key on its own; this only makes the usual case immediate.
func Confirm(ctx context.Context, env Env, secretID int64, retired []byte, peer model.FederationPeer) {
	if len(retired) == 0 {
		return
	}

	peer.Issuer = env.PublicURL()
	peer.PrevSecret = nil

	result, err := env.FederationPeerClient(peer).Search(ctx, model.MCPSearchParams{Query: "ok", Limit: 1})
	if err != nil || result.IsError {
		return
	}

	clearParams := db.ClearFederationSecretPrevParams{ID: secretID, PrevSecretCrypt: retired}

	clearErr := env.ClearFederationSecretPrev(ctx, clearParams)
	if clearErr != nil {
		env.AuditLogger().Error("failed to retire the rotated federation key", "secretID", secretID, "error", clearErr)
	}
}

// staged reports whether this side holds a proposal the peer has not been
// confirmed to hold. The previous key still being there is that signal: a
// confirmed rotation retires it, and the grace window retires it anyway.
func staged(row db.FederationSecret) bool {
	return len(row.PrevSecretCrypt) > 0 && row.RotatedAt != nil &&
		time.Since(*row.RotatedAt) <= model.RotationGrace
}

// retiredCrypt is the stored ciphertext a successful probe may retire: the
// previous key when a proposal was already staged, and the one this call just
// moved aside otherwise.
func retiredCrypt(row db.FederationSecret) []byte {
	if staged(row) {
		return row.PrevSecretCrypt
	}
	return row.SecretCrypt
}

func peerFromRow(env Env, row db.FederationSecret) (model.FederationPeer, error) {
	secret, err := env.DecryptData(row.SecretCrypt)
	if err != nil {
		return model.FederationPeer{}, fmt.Errorf("decrypt federation secret: %w", err)
	}

	peer := model.FederationPeer{
		KBURL:  *row.KbUrl,
		KID:    row.Kid,
		Secret: secret,
	}

	if !staged(row) {
		return peer, nil
	}

	previous, err := env.DecryptData(row.PrevSecretCrypt)
	if err != nil {
		return model.FederationPeer{}, fmt.Errorf("decrypt previous federation secret: %w", err)
	}
	peer.PrevSecret = previous

	return peer, nil
}

func firstText(result model.FederationResult) string {
	for _, item := range result.Content {
		if item.Text != "" {
			return item.Text
		}
	}
	return "no reason given"
}
