package hermesllm

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// bearerScheme is the Authorization scheme carrying hermesllm's API key.
// hermesllm is a plain OpenAI-compatible endpoint — like any OpenAI server it
// has an api_key — and any OpenAI client (fleet's OpenAILLM included) already
// sends it as `Authorization: Bearer <api_key>`.
const bearerScheme = "Bearer "

// APIKeyAuth gates a handler on hermesllm's own OpenAI-standard API key,
// presented as `Authorization: Bearer <api_key>` and compared in constant time.
//
// An empty configured key DISABLES the check and admits every request. This is
// the opposite of codellm's fail-safe, and deliberately so: codellm has a second
// lane (the browser delegated-admin cookie gate) to fall through to, while
// hermesllm has none — an always-failing check would make the service unusable.
// The operator turns auth on by setting a key.
func APIKeyAuth(apiKey string) func(http.Handler) http.Handler {
	want := []byte(apiKey)
	return func(next http.Handler) http.Handler {
		if len(want) == 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := []byte(bearerToken(r))
			if subtle.ConstantTimeCompare(got, want) != 1 {
				writeError(w, http.StatusUnauthorized, "invalid_request_error", "invalid api key")
				return
			}
			next.ServeHTTP(w, r)
		})
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
