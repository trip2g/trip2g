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
	"slices"
	"strings"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

var (
	ErrFedAuthUnknownKid  = errors.New("federation auth unknown kid")
	ErrFedAuthBadSig      = errors.New("federation auth bad signature")
	ErrFedAuthExpired     = errors.New("federation auth expired")
	ErrFedAuthFutureIAT   = errors.New("federation auth future iat")
	ErrFedAuthBodyChanged = errors.New("federation auth body does not match the signature")
)

const federationJWTSkew = 5 * time.Second

const contentTypeText = "text"

// FederationVerifyEnv is what verifying a peer's bearer needs. Exported because
// the pairing-description endpoint lives in its own package — it answers a
// different question from the tool surface — and must accept exactly the same
// credential rather than a second reading of it.
type FederationVerifyEnv = federationVerifyEnv

// VerifyFederationBearer reports the pairing a peer's JWT authenticated as, or
// why it did not. The body is passed so a signature that covers one is checked
// against it; a request without a body passes nil, and a token that binds
// nothing is accepted as before.
func VerifyFederationBearer(ctx context.Context, env FederationVerifyEnv, token string, body []byte) (string, error) {
	auth, err := verifyInbound(ctx, env, token, body)
	if err != nil {
		return "", err
	}
	return auth.KID, nil
}

type federationVerifyEnv interface {
	FederationSecretByKID(ctx context.Context, kid string) (db.FederationSecret, bool, error)
	ListFederationSecretSubgraphsByKID(ctx context.Context, kid string) ([]string, error)
	DecryptData([]byte) ([]byte, error)
	ClearFederationSecretPrev(ctx context.Context, arg db.ClearFederationSecretPrevParams) error
	Logger() logger.Logger
}

// federationAuth is what a verified inbound call carries into the handlers.
//
// SecretID and BodyBound exist for one caller each and are not decoration.
// The rotate handler writes to the row it authenticated against rather than
// looking one up again, and it refuses a request whose body the signature does
// not cover — a key in unsigned arguments is a key anything on the connection
// can replace inside the 30-second window.
type federationAuth struct {
	KID              string
	AllowedSubgraphs []string
	SecretID         int64
	BodyBound        bool
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
	Bh  string `json:"bh,omitempty"`
}

func signOutbound(secret []byte, kid, iss, rid string, body []byte) (string, error) {
	return signOutboundAt(secret, kid, iss, rid, time.Now(), 30*time.Second, body)
}

func signOutboundAt(secret []byte, kid, iss, rid string, issuedAt time.Time, ttl time.Duration, body []byte) (string, error) {
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
		Bh:  bodyDigest(body),
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

func verifyInbound(ctx context.Context, env federationVerifyEnv, token string, body []byte) (federationAuth, error) {
	header, claims, unsigned, signature, err := parseFederationJWT(token)
	if err != nil {
		return federationAuth{}, err
	}
	if header.Alg != "HS256" || header.Kid == "" {
		return federationAuth{}, ErrFedAuthBadSig
	}

	secretRow, ok, err := env.FederationSecretByKID(ctx, header.Kid)
	if err != nil {
		return federationAuth{}, fmt.Errorf("get federation secret by kid: %w", err)
	}
	if !ok {
		return federationAuth{}, ErrFedAuthUnknownKid
	}

	matched, usedPrevious, err := matchFederationSecret(env, secretRow, unsigned, signature)
	if err != nil {
		return federationAuth{}, err
	}
	if !matched {
		return federationAuth{}, ErrFedAuthBadSig
	}

	now := time.Now().Unix()
	if claims.Exp < now-int64(federationJWTSkew.Seconds()) {
		return federationAuth{}, ErrFedAuthExpired
	}
	if claims.Iat > now+int64(federationJWTSkew.Seconds()) {
		return federationAuth{}, ErrFedAuthFutureIAT
	}

	// A peer that predates body binding sends no bh and is authenticated as
	// before; one that sends it has to have sent the body it signed. Only the
	// calls that need the guarantee insist on its presence — see BodyBound.
	if claims.Bh != "" {
		digest := sha256.Sum256(body)
		if subtle.ConstantTimeCompare([]byte(claims.Bh), []byte(base64.RawURLEncoding.EncodeToString(digest[:]))) != 1 {
			return federationAuth{}, ErrFedAuthBodyChanged
		}
	}

	allowedSubgraphs, err := env.ListFederationSecretSubgraphsByKID(ctx, header.Kid)
	if err != nil {
		return federationAuth{}, fmt.Errorf("list federation secret subgraphs by kid: %w", err)
	}

	// The peer signed with the current key, so it holds it, so the one it
	// rotated away from has nothing left to cover. Retiring it here rather than
	// on a timer is what keeps the two-key state to the milliseconds around a
	// rotation instead of the whole grace window.
	//
	// Conditional on the value this request read, and best effort. Both matter,
	// for the same reason: the decision was made from a row that another request
	// may have moved on since. A rotation racing this one leaves a *different*
	// previous key staged, and a clear keyed on the id alone would wipe it —
	// taking with it the key that rotation's own lost-response retry depends on.
	// A stale clear has to be a no-op, and a housekeeping write that fails has
	// no business turning someone's search into an authentication error.
	if !usedPrevious && len(secretRow.PrevSecretCrypt) > 0 {
		clearParams := db.ClearFederationSecretPrevParams{
			ID:              secretRow.ID,
			PrevSecretCrypt: secretRow.PrevSecretCrypt,
		}

		clearErr := env.ClearFederationSecretPrev(ctx, clearParams)
		if clearErr != nil {
			env.Logger().Warn("mcp:federation: failed to retire a rotated key",
				"kid", header.Kid, "secretID", secretRow.ID, "error", clearErr)
		}
	}

	return federationAuth{
		KID:              header.Kid,
		AllowedSubgraphs: allowedSubgraphs,
		SecretID:         secretRow.ID,
		BodyBound:        claims.Bh != "",
	}, nil
}

// matchFederationSecret tries the current key and then, inside the grace window,
// the one the pairing rotated away from. Reporting which matched is what lets
// the caller retire the old key on the first call that proves it is unneeded.
func matchFederationSecret(
	env federationVerifyEnv,
	row db.FederationSecret,
	unsigned string,
	signature []byte,
) (bool, bool, error) {
	secret, err := env.DecryptData(row.SecretCrypt)
	if err != nil {
		return false, false, fmt.Errorf("decrypt federation secret: %w", err)
	}
	if subtle.ConstantTimeCompare(signature, hmacSHA256(secret, []byte(unsigned))) == 1 {
		return true, false, nil
	}

	if len(row.PrevSecretCrypt) == 0 || row.RotatedAt == nil {
		return false, false, nil
	}
	if time.Since(*row.RotatedAt) > model.RotationGrace {
		return false, false, nil
	}

	previous, err := env.DecryptData(row.PrevSecretCrypt)
	if err != nil {
		return false, false, fmt.Errorf("decrypt federation previous secret: %w", err)
	}
	if subtle.ConstantTimeCompare(signature, hmacSHA256(previous, []byte(unsigned))) == 1 {
		return true, true, nil
	}
	return false, false, nil
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
		if ref := items[i].Federation; ref != nil && ref.KBID != "" {
			composed := localSegment + "/" + ref.KBID
			if ref.AgentInstruction == federationAgentInstruction(ref.KBID) {
				ref.AgentInstruction = federationAgentInstruction(composed)
			}
			ref.KBID = composed
		}
	}
}

// prefixPointerHints rewrites the pointer hint lines in a text block into the
// caller's frame. The hint is the one line a text-only client sees a kb_id in,
// so it has to compose across hops like the structured kb_id does. Only a line
// that is exactly what federationPointerHint renders is touched.
func prefixPointerHints(localSegment, text string) string {
	if !strings.Contains(text, pointerHintMark) {
		return text
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		start := strings.Index(line, "kb_id: ")
		mark := strings.Index(line, pointerHintMark)
		if start < 0 || mark < start {
			continue
		}
		kbID := line[start+len("kb_id: ") : mark]
		if line[start:] != federationPointerHint(kbID) {
			continue
		}
		lines[i] = line[:start] + federationPointerHint(localSegment+"/"+kbID)
	}
	return strings.Join(lines, "\n")
}

// rewriteFederatedResponse rewrites a federated response returned by the child
// reached through localSegment into this hub's frame. Search results get their
// kb_id prefixed, the pointer hint lines in the text with them; a not-configured
// status gets the hub, its bases and the kb_id prefixed so the message names
// what exists in the caller's frame, and localKBIDs — the bases this hub
// connects directly, as the caller may see them — added as the second level.
func rewriteFederatedResponse(localSegment string, localKBIDs []string, result *model.FederationResult) {
	if result == nil {
		return
	}
	for i := range result.Content {
		result.Content[i].Text = prefixPointerHints(localSegment, result.Content[i].Text)
	}
	if len(result.StructuredContent) == 0 {
		return
	}

	var status FederationStatusPayload
	if err := json.Unmarshal(result.StructuredContent, &status); err == nil &&
		status.Status == "federation_not_configured" && status.KBID != "" {
		status.KBID = localSegment + "/" + status.KBID
		if status.Hub == "" {
			status.Hub = localSegment
		} else {
			status.Hub = localSegment + "/" + status.Hub
		}
		for i := range status.ConnectedKBIDs {
			status.ConnectedKBIDs[i] = localSegment + "/" + status.ConnectedKBIDs[i]
		}
		status.LocalKBIDs = localKBIDs
		status.Message = notConfiguredMessage(status)
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

// notConfiguredMessage explains a kb_id that resolved to nothing: which
// segment is unknown, which bases exist at that point in the caller's frame,
// the likeliest address when a later segment names one of them, and — for a
// miss reported by a peer — which bases this hub connects directly, so the
// peer's list is not mistaken for everything there is.
func notConfiguredMessage(status FederationStatusPayload) string {
	connected := status.ConnectedKBIDs
	if status.KBID == "" {
		if len(connected) == 0 {
			return "Federation is not configured for this hub. No KB-notes were found. To enable federation, create a note with mcp_federation_kb_url frontmatter pointing to another MCP endpoint."
		}
		return fmt.Sprintf("None of the requested kb_ids names a connected base on this hub. Connected bases: %s.",
			strings.Join(connected, ", "))
	}

	prefix := ""
	if status.Hub != "" {
		prefix = status.Hub + "/"
	}
	unknown, rest := splitKBID(strings.TrimPrefix(status.KBID, prefix))

	var sb strings.Builder
	fmt.Fprintf(&sb, "Federation is not configured for kb_id %q: ", status.KBID)
	switch {
	case status.Hub != "":
		fmt.Fprintf(&sb, "hub %q has no base %q.", status.Hub, unknown)
	case len(connected) == 0:
		sb.WriteString("this hub has no connected bases.")
		return sb.String()
	default:
		fmt.Fprintf(&sb, "no connected base on this hub is named %q.", unknown)
	}
	if len(connected) > 0 {
		if status.Hub != "" {
			fmt.Fprintf(&sb, " Bases connected under %q: %s.", status.Hub, strings.Join(connected, ", "))
		} else {
			fmt.Fprintf(&sb, " Connected bases: %s.", strings.Join(connected, ", "))
		}
	}
	if status.Hub != "" && len(status.LocalKBIDs) > 0 {
		fmt.Fprintf(&sb, " Bases connected directly to this hub: %s.", strings.Join(status.LocalKBIDs, ", "))
	}

	// A guessed prefix in front of a real base is the common mistake; name it.
	for _, seg := range strings.Split(rest, "/") {
		if seg != "" && slices.Contains(connected, prefix+seg) {
			fmt.Fprintf(&sb, " %q is a connected base — address it as kb_id=%q.", prefix+seg, prefix+seg)
			return sb.String()
		}
	}
	switch {
	case status.Hub == "" && rest == "":
		fmt.Fprintf(&sb, " If %q lives under one of them, address it as <hub>/%s (bases reached through a hub are addressed <hub>/<base>).",
			unknown, unknown)
	case status.Hub != "" && len(connected) == 0:
		fmt.Fprintf(&sb, " To find its kb_id, search the hub for it: federated_search(kb_id=%q, query=%q) — the pointer card prints the kb_id to use.",
			status.Hub, unknown)
	}
	return sb.String()
}

// bodyDigest is what a signature has to cover for the body to be authenticated.
// Empty for no body, so a caller with nothing to bind sends no bh and a verifier
// has nothing to check — which is also how a peer that predates body binding
// looks.
func bodyDigest(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	digest := sha256.Sum256(body)
	return base64.RawURLEncoding.EncodeToString(digest[:])
}
