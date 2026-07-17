package webhookutil

import (
	"context"

	"trip2g/internal/appreq"
)

// WriteScopeDenied reports whether writing path is denied by the request's
// webhook write scope. Scoped-token requests: empty write_patterns means
// deny-all, non-empty denies on no-match. Keyed off Scoped — the flag every
// shortapitoken carries — not off DeliveryKind, which is an optional claim: a
// scoped token minted without it must not fall through to the legacy allow-all.
// Unscoped/admin requests keep the legacy behaviour: empty means allow-all.
// path must be the canonical slash-less note path.
func WriteScopeDenied(ctx context.Context, path string) bool {
	wp := appreq.WebhookWritePatterns(ctx)
	if appreq.Scoped(ctx) || appreq.WebhookDeliveryKind(ctx) != "" {
		return len(wp) == 0 || !MatchesAny(path, wp)
	}
	return len(wp) > 0 && !MatchesAny(path, wp)
}
