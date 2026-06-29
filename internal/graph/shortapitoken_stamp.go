package graph

import (
	"strings"

	"trip2g/internal/appreq"
	"trip2g/internal/personaltoken"
	"trip2g/internal/shortapitoken"
)

// stampShortAPIToken inspects req for a Bearer token that is a shortapitoken
// (JWT — not a personal t2g_* token) and, if valid, stamps the appreq with
// the scoped delivery claims (WebhookReadPatterns, WebhookWritePatterns,
// WebhookDepth, WebhookDeliveryKind, WebhookDeliveryID).
//
// It is intentionally a no-op when:
//   - there is no Authorization header
//   - the token is a personal token (starts with "t2g_")
//   - the token is already stamped (WebhookReadPatterns already set)
//   - the JWT signature/expiry is invalid (graceful degradation: the request
//     proceeds anonymously and read_patterns stay empty, so scoped private
//     notes are invisible — the same as an unknown caller)
//
// It must be called once per GraphQL operation, before resolvers run.
func stampShortAPIToken(req *appreq.Request, secret string) {
	if req == nil || req.Req == nil {
		return
	}

	// If already stamped by an earlier path (e.g. a mutation that called
	// checkapikey.Resolve), skip to avoid double-parsing.
	if req.WebhookScoped || len(req.WebhookReadPatterns) > 0 || len(req.WebhookWritePatterns) > 0 || req.WebhookDepth != 0 {
		return
	}

	authHeader := string(req.Req.Request.Header.Peek("Authorization"))
	if authHeader == "" {
		return
	}

	const bearerPrefix = "Bearer "
	tokenStr, ok := strings.CutPrefix(authHeader, bearerPrefix)
	if !ok || tokenStr == "" {
		return
	}

	// Personal tokens (t2g_*) and admin/session JWTs are handled elsewhere;
	// only shortapitokens need stamping here.
	if personaltoken.IsPersonal(tokenStr) {
		return
	}

	data, err := shortapitoken.Parse(tokenStr, secret)
	if err != nil {
		// Invalid or expired token — leave request anonymous.
		return
	}

	req.WebhookDepth = data.Depth
	req.WebhookReadPatterns = data.ReadPatterns
	req.WebhookWritePatterns = data.WritePatterns
	req.WebhookScoped = true
	req.WebhookDeliveryKind = data.DeliveryKind
	req.WebhookDeliveryID = data.DeliveryID
}
