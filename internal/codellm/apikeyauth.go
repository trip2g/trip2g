package codellm

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
)

// bearerScheme is the Authorization scheme carrying codellm's API key. codellm
// is a plain OpenAI-compatible endpoint — like any OpenAI server it has an
// api_key — and any OpenAI client (fleet's OpenAILLM included) already sends it
// as `Authorization: Bearer <api_key>`. No special header, no coupling to fleet.
const bearerScheme = "Bearer "

var (
	// errAPIKeyAuthDisabled is returned when no API key is configured, so key
	// auth is off and every request must pass the browser admin-cookie gate.
	errAPIKeyAuthDisabled = errors.New("codellm: api-key auth disabled (no key configured)")
	// errAPIKeyMismatch is returned when the presented bearer token is absent or
	// does not exactly match the configured key.
	errAPIKeyMismatch = errors.New("codellm: api key absent or mismatched")
)

// APIKeyCheck builds the TokenCheck for BrowserAuth's seam: it authenticates a
// caller by codellm's own OpenAI-standard API key, presented as
// `Authorization: Bearer <api_key>` and compared in constant time. Any client
// speaking the OpenAI protocol — fleet included — authenticates this way; there
// is no fleet-specific credential, which keeps codellm interchangeable with a
// real LLM endpoint (both are just "base_url + api_key" to a caller).
//
// Fail-safe: an empty configured key DISABLES key auth — the returned check
// always fails, so no request (with or without a header, even an empty one) can
// authenticate this way, and callers fall through to the browser admin-cookie
// gate. Only a non-empty configured key enables it, and only an exact
// constant-time match passes. The empty-configured guard runs BEFORE the
// compare precisely so an empty key can never match an empty/absent header
// (that would be an auth bypass, since a constant-time compare of two empty
// slices returns equal).
func APIKeyCheck(apiKey string) func(*http.Request) error {
	want := []byte(apiKey)
	return func(r *http.Request) error {
		if len(want) == 0 {
			return errAPIKeyAuthDisabled
		}
		got := []byte(bearerToken(r))
		if subtle.ConstantTimeCompare(got, want) != 1 {
			return errAPIKeyMismatch
		}
		return nil
	}
}

// bearerToken extracts the token from an `Authorization: Bearer <token>` header,
// or "" when the header is absent or not a Bearer credential. The scheme name is
// case-insensitive per RFC 7235.
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) < len(bearerScheme) || !strings.EqualFold(h[:len(bearerScheme)], bearerScheme) {
		return ""
	}
	return h[len(bearerScheme):]
}
