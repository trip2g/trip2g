package codellm

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
)

// channelTokenScheme is the Authorization scheme carrying the fleet-lane channel
// token. codellm impersonates an OpenAI endpoint and fleet's OpenAILLM client
// already sends the api_key as `Authorization: Bearer <key>`, so the channel
// token IS that api_key — no extra header for fleet to set.
const channelTokenScheme = "Bearer "

var (
	// errChannelLaneDisabled is returned when no channel token is configured, so
	// the fleet lane is off and every request must pass the browser admin gate.
	errChannelLaneDisabled = errors.New("codellm: channel-token lane disabled (no token configured)")
	// errChannelTokenMismatch is returned when the presented bearer token is
	// absent or does not exactly match the configured token.
	errChannelTokenMismatch = errors.New("codellm: channel token absent or mismatched")
)

// ChannelTokenCheck builds the fleet-lane TokenCheck for BrowserAuth's seam: it
// authenticates a server-to-server caller by a shared channel token presented as
// `Authorization: Bearer <token>`, compared in constant time.
//
// Fail-safe: an empty configured token DISABLES the fleet lane — the returned
// check always fails, so no request (with or without a header, even an empty
// one) can authenticate this way, and callers fall through to the browser admin
// gate. Only a non-empty configured token enables the lane, and only an exact
// constant-time match passes. The empty-configured guard runs BEFORE the compare
// precisely so an empty token can never match an empty/absent header (that would
// be an auth bypass, since a constant-time compare of two empty slices returns
// equal).
func ChannelTokenCheck(token string) func(*http.Request) error {
	want := []byte(token)
	return func(r *http.Request) error {
		if len(want) == 0 {
			return errChannelLaneDisabled
		}
		got := []byte(bearerToken(r))
		if subtle.ConstantTimeCompare(got, want) != 1 {
			return errChannelTokenMismatch
		}
		return nil
	}
}

// bearerToken extracts the token from an `Authorization: Bearer <token>` header,
// or "" when the header is absent or not a Bearer credential. The scheme name is
// case-insensitive per RFC 7235.
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) < len(channelTokenScheme) || !strings.EqualFold(h[:len(channelTokenScheme)], channelTokenScheme) {
		return ""
	}
	return h[len(channelTokenScheme):]
}
