package mcp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/model"
)

var (
	ErrFedAuthUnknownKid = errors.New("federation auth unknown kid")
	ErrFedAuthBadSig     = errors.New("federation auth bad signature")
	ErrFedAuthExpired    = errors.New("federation auth expired")
	ErrFedAuthFutureIAT  = errors.New("federation auth future iat")
	ErrFedAuthRevoked    = errors.New("federation auth revoked")
)

const federationJWTSkew = 5 * time.Second

const contentTypeText = "text"

type federationVerifyEnv interface {
	FederationSecretByKID(ctx context.Context, kid string) (db.FederationSecret, bool, error)
	ListFederationSecretSubgraphsByKID(ctx context.Context, kid string) ([]string, error)
	DecryptData([]byte) ([]byte, error)
}

type federationJWTHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid"`
}

type federationJWTClaims struct {
	Iss string `json:"iss"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
	Rid string `json:"rid"`
}

func signOutbound(secret []byte, kid, iss, rid string) (string, error) {
	return signOutboundAt(secret, kid, iss, rid, time.Now(), 30*time.Second)
}

func signOutboundAt(secret []byte, kid, iss, rid string, issuedAt time.Time, ttl time.Duration) (string, error) {
	header := federationJWTHeader{
		Alg: "HS256",
		Typ: "JWT",
		Kid: kid,
	}
	claims := federationJWTClaims{
		Iss: iss,
		Iat: issuedAt.Unix(),
		Exp: issuedAt.Add(ttl).Unix(),
		Rid: rid,
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

func verifyInbound(ctx context.Context, env federationVerifyEnv, token string) (string, []string, error) {
	header, claims, unsigned, signature, err := parseFederationJWT(token)
	if err != nil {
		return "", nil, err
	}
	if header.Alg != "HS256" || header.Kid == "" {
		return "", nil, ErrFedAuthBadSig
	}

	secretRow, ok, err := env.FederationSecretByKID(ctx, header.Kid)
	if err != nil {
		return "", nil, fmt.Errorf("get federation secret by kid: %w", err)
	}
	if !ok {
		return "", nil, ErrFedAuthUnknownKid
	}
	if secretRow.RevokedAt != nil {
		return "", nil, ErrFedAuthRevoked
	}

	secret, err := env.DecryptData(secretRow.SecretCrypt)
	if err != nil {
		return "", nil, fmt.Errorf("decrypt federation secret: %w", err)
	}

	expected := hmacSHA256(secret, []byte(unsigned))
	if subtle.ConstantTimeCompare(signature, expected) != 1 {
		return "", nil, ErrFedAuthBadSig
	}

	now := time.Now().Unix()
	if claims.Exp < now-int64(federationJWTSkew.Seconds()) {
		return "", nil, ErrFedAuthExpired
	}
	if claims.Iat > now+int64(federationJWTSkew.Seconds()) {
		return "", nil, ErrFedAuthFutureIAT
	}

	allowedSubgraphs, err := env.ListFederationSecretSubgraphsByKID(ctx, header.Kid)
	if err != nil {
		return "", nil, fmt.Errorf("list federation secret subgraphs by kid: %w", err)
	}

	return header.Kid, allowedSubgraphs, nil
}

func parseFederationJWT(token string) (federationJWTHeader, federationJWTClaims, string, []byte, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return federationJWTHeader{}, federationJWTClaims{}, "", nil, ErrFedAuthBadSig
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return federationJWTHeader{}, federationJWTClaims{}, "", nil, ErrFedAuthBadSig
	}
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return federationJWTHeader{}, federationJWTClaims{}, "", nil, ErrFedAuthBadSig
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return federationJWTHeader{}, federationJWTClaims{}, "", nil, ErrFedAuthBadSig
	}

	var header federationJWTHeader
	if e := json.Unmarshal(headerBytes, &header); e != nil {
		return federationJWTHeader{}, federationJWTClaims{}, "", nil, ErrFedAuthBadSig
	}
	var claims federationJWTClaims
	if e := json.Unmarshal(claimsBytes, &claims); e != nil {
		return federationJWTHeader{}, federationJWTClaims{}, "", nil, ErrFedAuthBadSig
	}

	return header, claims, parts[0] + "." + parts[1], signature, nil
}

func hmacSHA256(secret, payload []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return mac.Sum(nil)
}

func splitKBID(id string) (string, string) {
	head, rest, _ := strings.Cut(id, "/")
	return head, rest
}

// prefixKBID rewrites every search result into the caller's frame by stamping
// the local peer segment onto its kb_id. A base's id is relative to the peer
// that mounts it, so each hop prefixes its own segment: an item with no kb_id
// yet (a leaf note answered here) becomes localSegment; one that already
// carries a kb_id (reported by a deeper hop) becomes localSegment/<kb_id>.
// The pointer ref's kb_id is kept in sync for back-compat.
func prefixKBID(localSegment string, items []SearchResultItem) {
	for i := range items {
		if items[i].KBID == "" {
			items[i].KBID = localSegment
		} else {
			items[i].KBID = localSegment + "/" + items[i].KBID
		}
		if items[i].Federation != nil && items[i].Federation.KBID != "" {
			items[i].Federation.KBID = localSegment + "/" + items[i].Federation.KBID
		}
	}
}

// rewriteFederatedResponse rewrites a federated response returned by the child
// reached through localSegment into this hub's frame. Search results get their
// kb_id prefixed; a not-configured status carries a prefixable hint so the
// suggested address composes across every hop back to the root.
func rewriteFederatedResponse(localSegment string, result *model.FederationResult) {
	if result == nil || len(result.StructuredContent) == 0 {
		return
	}

	var status FederationStatusPayload
	if err := json.Unmarshal(result.StructuredContent, &status); err == nil &&
		status.Status == "federation_not_configured" && status.KBID != "" {
		status.KBID = localSegment + "/" + status.KBID
		status.Message = notConfiguredMessage(status.KBID)
		if encoded, e := json.Marshal(status); e == nil {
			result.StructuredContent = encoded
			result.Content = []model.FederationContent{{Type: contentTypeText, Text: status.Message}}
		}
		return
	}

	var payload SearchResultPayload
	if err := json.Unmarshal(result.StructuredContent, &payload); err == nil && len(payload.Results) > 0 {
		prefixKBID(localSegment, payload.Results)
		if encoded, e := json.Marshal(payload); e == nil {
			result.StructuredContent = encoded
		}
	}
}

// notConfiguredMessage renders the not-configured hint. A flat kb_id may live
// under a hub, so the hint tells the caller how nested bases are addressed; a
// composed (already nested) kb_id is stated verbatim as the address to use.
func notConfiguredMessage(kbID string) string {
	switch {
	case kbID == "":
		return "Federation is not configured for this hub. No KB-notes were found. To enable federation, create a note with mcp_federation_kb_url frontmatter pointing to another MCP endpoint."
	case strings.Contains(kbID, "/"):
		return fmt.Sprintf(
			"Federation is not configured for kb_id %q. Bases reached through a hub are addressed <hub>/<base> — address this base as %q.",
			kbID, kbID)
	default:
		return fmt.Sprintf(
			"Federation is not configured for kb_id %q. If %q lives under a hub, address it as <hub>/%s (bases reached through a hub are addressed <hub>/<base>).",
			kbID,
			kbID,
			kbID,
		)
	}
}
