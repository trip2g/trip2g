// Package rotatefederationsecret replaces the key a pairing shares with a peer.
//
// One primitive with two callers: installing an outbound secret runs it once so
// the key that arrived out of band — through a chat, usually, relayed by whoever
// was asked to pass it on — stops being the live credential; an operator runs it
// again whenever they want. Both go through Exchange, which is the part that
// talks to the peer, so there is no second notion of what a rotation is.
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

// ErrPeerRefused is what a caller shows an operator: the peer did not take the
// new key, so nothing changed on either side and the same call can be repeated.
var ErrPeerRefused = errors.New("the peer did not accept the new key")

// ErrInsecureURL is separate because the fix is different in kind — not "try
// again" but "this deployment has decided it federates over the open internet,
// so the address has to be https".
var ErrInsecureURL = errors.New("rotation needs an https peer address")

type Env interface {
	OutboundFederationSecretByKID(ctx context.Context, kid string) (db.FederationSecret, error)
	RotateFederationSecret(ctx context.Context, arg db.RotateFederationSecretParams) error
	ClearFederationSecretPrev(ctx context.Context, arg db.ClearFederationSecretPrevParams) error
	FederationPeerClient(peer model.FederationPeer) model.Federation
	EncryptData([]byte) ([]byte, error)
	DecryptData([]byte) ([]byte, error)
	FederationAllowsPlainHTTP() bool
	PublicURL() string
	AuditLogger() logger.Logger
}

// Exchange mints a key and asks the peer to adopt it, returning what the peer
// now holds. It writes nothing: a caller decides what to store, which is what
// lets the install path record the result before any row exists and the operator
// path record it against one that does.
//
// The new key is generated here rather than by the peer, and that is what makes
// a lost response survivable: the caller keeps what it proposed, and a retry
// carrying the same value reaches a peer that already applied it as a no-op.
func Exchange(ctx context.Context, env Env, peer model.FederationPeer) ([]byte, error) {
	if !env.FederationAllowsPlainHTTP() && !strings.HasPrefix(peer.KBURL, "https://") {
		return nil, fmt.Errorf("%w: %s", ErrInsecureURL, peer.KBURL)
	}

	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	if err != nil {
		return nil, fmt.Errorf("generate federation secret: %w", err)
	}

	peer.Issuer = env.PublicURL()
	params := model.MCPRotateSecretParams{SecretHex: hex.EncodeToString(secret)}

	result, err := env.FederationPeerClient(peer).RotateSecret(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrPeerRefused, err)
	}
	if result.IsError {
		return nil, fmt.Errorf("%w: %s", ErrPeerRefused, firstText(result))
	}

	return secret, nil
}

// Resolve rotates an outbound pairing that is already installed.
//
// A peer that refuses and an address that cannot carry a key are answers an
// operator acts on, not faults: they come back as an ErrorPayload with the
// reason, and nothing on either side has changed.
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

	secret, err := Exchange(ctx, env, peer)
	if errors.Is(err, ErrPeerRefused) || errors.Is(err, ErrInsecureURL) {
		return &graphmodel.ErrorPayload{Message: err.Error()}, nil
	}
	if err != nil {
		return nil, err
	}

	encrypted, err := env.EncryptData(secret)
	if err != nil {
		return nil, fmt.Errorf("encrypt federation secret: %w", err)
	}

	rotateParams := db.RotateFederationSecretParams{
		SecretCrypt: encrypted,
		ID:          row.ID,
	}

	err = env.RotateFederationSecret(ctx, rotateParams)
	if err != nil {
		return nil, fmt.Errorf("store rotated federation secret: %w", err)
	}

	env.AuditLogger().Info("federation secret rotated", "kid", kid, "secretID", row.ID, "kbURL", *row.KbUrl)

	// row.SecretCrypt is what the rotation just moved into prev_secret_crypt —
	// the stored ciphertext, not a re-encryption of the same key, which would
	// not compare equal if encryption is not deterministic.
	Confirm(ctx, env, row.ID, row.SecretCrypt, model.FederationPeer{
		KBID:   peer.KBID,
		KBURL:  peer.KBURL,
		KID:    peer.KID,
		Secret: secret,
	})

	return &graphmodel.RotateFederationSecretPayload{Kid: kid}, nil
}

// Confirm asks the peer one cheap question with the new key and, if it answers,
// drops the key the pairing rotated away from.
//
// The probe carries the new key ALONE. An ordinary client falls back to the
// previous key on an auth refusal, which is right for traffic and wrong here:
// the fallback would make a peer that never applied the rotation answer
// successfully, and this would then retire the only key that peer still accepts
// — leaving a link that cannot heal. What is being proved is specifically that
// the new key verifies.
//
// Best effort otherwise, and never a decision the other way: a probe that does
// not come back leaves both keys in place, because it cannot tell a peer that
// refused from one that was busy for a second. The grace window retires the old
// key on its own; this only makes the usual case immediate.
func Confirm(ctx context.Context, env Env, secretID int64, retired []byte, peer model.FederationPeer) {
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

	// A rotation whose response was lost left the peer holding the new key while
	// this side still calls the old one current. Carrying both is what lets the
	// call that starts the next rotation reach it at all.
	if len(row.PrevSecretCrypt) > 0 && row.RotatedAt != nil && time.Since(*row.RotatedAt) <= model.RotationGrace {
		previous, prevErr := env.DecryptData(row.PrevSecretCrypt)
		if prevErr != nil {
			return model.FederationPeer{}, fmt.Errorf("decrypt previous federation secret: %w", prevErr)
		}
		peer.PrevSecret = previous
	}

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
