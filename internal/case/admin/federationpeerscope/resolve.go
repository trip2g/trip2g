// Package federationpeerscope asks a peer what this pairing is allowed to see
// there.
//
// Only the peer knows: scope is granted on its side, against the kid, and
// nothing about it is recorded here. Without asking, the asking side is left
// guessing — and the guess it makes wrong is the expensive one, because a link
// scoped to nothing authenticates, answers, and returns an empty result set that
// looks exactly like a query nothing matched.
package federationpeerscope

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"trip2g/internal/db"
	graphmodel "trip2g/internal/graph/model"
	"trip2g/internal/model"
)

type Payload = graphmodel.FederationPeerScopeOrErrorPayload

type Env interface {
	OutboundFederationSecretByKID(ctx context.Context, kid string) (db.FederationSecret, error)
	FederationPeerClient(peer model.FederationPeer) model.Federation
	DecryptData([]byte) ([]byte, error)
	PublicURL() string
}

// Resolve reports what the peer says, or why it could not be asked. Both are
// answers an operator acts on, so neither is an error.
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

	result, err := env.FederationPeerClient(peer).GrantedScope(ctx)
	if err != nil {
		//nolint:nilerr // intentional: an unreachable peer is what an operator came here to find out
		return &graphmodel.ErrorPayload{Message: "the peer did not answer: " + err.Error()}, nil
	}
	if result.IsError {
		return &graphmodel.ErrorPayload{Message: "the peer does not report its scope: " + firstText(result)}, nil
	}

	subgraphs, err := readSubgraphs(result)
	if err != nil {
		//nolint:nilerr // intentional: an answer this side cannot read is a fact about the peer, not a fault here
		return &graphmodel.ErrorPayload{Message: err.Error()}, nil
	}

	return &graphmodel.FederationPeerScopePayload{Kid: kid, Subgraphs: subgraphs}, nil
}

// readSubgraphs takes the structured answer and nothing else. The text beside it
// is for a human reading a tool call; parsing that instead would make this agree
// with the peer only for as long as its phrasing holds.
func readSubgraphs(result model.FederationResult) ([]string, error) {
	if len(result.StructuredContent) == 0 {
		return nil, errors.New("the peer answered without a structured scope; it is probably an older trip2g")
	}

	var payload struct {
		Subgraphs []string `json:"subgraphs"`
	}

	err := json.Unmarshal(result.StructuredContent, &payload)
	if err != nil {
		return nil, fmt.Errorf("the peer's answer could not be read: %w", err)
	}
	if payload.Subgraphs == nil {
		return []string{}, nil
	}

	return payload.Subgraphs, nil
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
		Issuer: env.PublicURL(),
	}

	// A rotation the peer has not confirmed leaves it holding the previous key,
	// and a question asked with the wrong one comes back as an auth failure that
	// reads like a broken pairing rather than one mid-rotation.
	if len(row.PrevSecretCrypt) > 0 {
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
