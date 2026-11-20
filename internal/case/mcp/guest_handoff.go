package mcp

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// GuestHandoffTTL is how long a browser has to carry a handoff from the hub's
// page to the peer.
const GuestHandoffTTL = 30 * time.Second

const guestHandoffUse = "guest"

// GuestHandoff is what a hub vouches for when it sends a reader to a peer: the
// page to open there, how many handoffs deep the reader is, the reader id, and
// the origin of the hub's site, whose page alone may post it.
type GuestHandoff struct {
	Path   string
	Hop    int
	Reader string
	Origin string
}

// GuestHandoffClaims is a verified handoff together with the edge it arrived
// over and the id that makes it single-use.
type GuestHandoffClaims struct {
	GuestHandoff

	Caller    FederationCaller
	ID        string
	ExpiresAt time.Time
}

// SignGuestHandoff signs a handoff with the edge secret, in the same JWT the
// tool surface uses, marked with a use the tool surface refuses.
func SignGuestHandoff(secret []byte, kid, iss string, handoff GuestHandoff) (string, error) {
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return "", fmt.Errorf("guest handoff id: %w", err)
	}

	now := time.Now()
	header := federationJWTHeader{Alg: "HS256", Typ: "JWT", Kid: kid}
	claims := federationJWTClaims{
		Iss:  iss,
		Iat:  now.Unix(),
		Exp:  now.Add(GuestHandoffTTL).Unix(),
		Rid:  hex.EncodeToString(id),
		Use:  guestHandoffUse,
		Path: handoff.Path,
		Hop:  handoff.Hop,
		Sub:  handoff.Reader,
		Org:  handoff.Origin,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal jwt header: %w", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal jwt claims: %w", err)
	}

	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." +
		base64.RawURLEncoding.EncodeToString(claimsJSON)
	signature := hmacSHA256(secret, []byte(unsigned))

	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

// VerifyGuestHandoff checks a handoff exactly as a tool call's bearer is
// checked, then insists it was issued as a handoff. Single use is the caller's:
// it holds the set of spent ids.
func VerifyGuestHandoff(ctx context.Context, env FederationVerifyEnv, token string) (GuestHandoffClaims, error) {
	auth, err := verifyInboundFor(ctx, env, token, nil, guestHandoffUse)
	if err != nil {
		return GuestHandoffClaims{}, err
	}

	claims := GuestHandoffClaims{
		GuestHandoff: GuestHandoff{
			Path:   auth.claims.Path,
			Hop:    auth.claims.Hop,
			Reader: auth.claims.Sub,
			Origin: auth.claims.Org,
		},
		Caller:    FederationCaller{KID: auth.KID, AllowedSubgraphs: auth.AllowedSubgraphs},
		ID:        auth.claims.Rid,
		ExpiresAt: time.Unix(auth.claims.Exp, 0),
	}
	return claims, nil
}
