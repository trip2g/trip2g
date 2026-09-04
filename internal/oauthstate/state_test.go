package oauthstate

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestSafeRedirect(t *testing.T) {
	const host = "app.example.com"

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", "/"},
		{"root", "/", "/"},
		{"path", "/dashboard", "/dashboard"},
		{"nested path", "/settings/profile", "/settings/profile"},
		// an absolute URL on our own host is reduced to its path: the frontend
		// sends location.href, and losing the path there sends every sign-in
		// back to the root instead of the page it started on
		{"own host", "https://app.example.com/boards/x", "/boards/x"},
		{"own host over http", "http://app.example.com/boards/x", "/boards/x"},
		{"own host with query", "https://app.example.com/boards/x?tab=2", "/boards/x?tab=2"},
		{"own host with fragment", "https://app.example.com/boards/x#!userspace=open", "/boards/x#!userspace=open"},
		{"own host bare", "https://app.example.com", "/"},
		{"own host with port", "https://app.example.com:8443/boards/x", "/"},
		// absolute URLs elsewhere must be rejected
		{"other host", "https://evil.com", "/"},
		{"other host with path", "http://evil.com/steal", "/"},
		// The host matches and the path is protocol-relative, so reducing the
		// URL to that path hands back "//evil.com" — a Location the browser
		// reads as another origin. The same shape the relative branch has
		// always refused.
		{"own host, protocol-relative path", "https://app.example.com//evil.com", "/"},
		{"own host, protocol-relative path with more", "https://app.example.com//evil.com/steal?a=1", "/"},
		{"own host, backslash path", "https://app.example.com/\\evil.com", "/%5Cevil.com"},
		{"userinfo disguise", "https://app.example.com@evil.com/steal", "/"},
		{"host as prefix", "https://app.example.com.evil.com/steal", "/"},
		// protocol-relative
		{"protocol relative", "//evil.com", "/"},
		{"protocol relative with path", "//evil.com/path", "/"},
		// backslash variants some browsers normalise
		{"backslash", "/\\evil.com", "/"},
		// no leading slash
		{"bare host", "evil.com", "/"},
		{"relative", "relative/path", "/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, safeRedirect(host, tc.in))
		})
	}
}

func TestSafeRedirectWithoutHostRejectsAbsolute(t *testing.T) {
	require.Equal(t, "/", safeRedirect("", "https://app.example.com/boards/x"))
}

func TestGenerateKeepsOwnOriginPath(t *testing.T) {
	var ctx fasthttp.RequestCtx
	ctx.Request.SetHost("app.example.com")

	encoded, err := Generate(&ctx, "https://app.example.com/boards/x#!userspace=open", true)
	require.NoError(t, err)

	raw, err := base64.URLEncoding.DecodeString(encoded)
	require.NoError(t, err)

	var s State
	require.NoError(t, json.Unmarshal(raw, &s))
	require.Equal(t, "/boards/x#!userspace=open", s.Redirect)
}

// csrfNonceFromState pulls the CSRF nonce out of an encoded state blob so the
// validate step can present a matching cookie (mirrors what the browser does).
func csrfNonceFromState(t *testing.T, encoded string) string {
	t.Helper()
	raw, err := base64.URLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	var s State
	require.NoError(t, json.Unmarshal(raw, &s))
	return s.Nonce
}

func TestOIDCNonceRoundTrip(t *testing.T) {
	var genCtx fasthttp.RequestCtx
	encoded, err := GenerateWithOIDCNonce(&genCtx, "/dashboard", "oidc-nonce-abc", true)
	require.NoError(t, err)

	// The blob must carry the OIDC nonce so the callback can bind the id_token.
	raw, err := base64.URLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	var s State
	require.NoError(t, json.Unmarshal(raw, &s))
	require.Equal(t, "oidc-nonce-abc", s.OIDCNonce)

	var valCtx fasthttp.RequestCtx
	valCtx.Request.Header.SetCookie(CookieName, csrfNonceFromState(t, encoded))

	redirect, nonce, err := ValidateWithOIDCNonce(&valCtx, encoded, true)
	require.NoError(t, err)
	require.Equal(t, "/dashboard", redirect)
	require.Equal(t, "oidc-nonce-abc", nonce)
}

func TestValidateWithOIDCNonceRejectsMismatchedCookie(t *testing.T) {
	var genCtx fasthttp.RequestCtx
	encoded, err := GenerateWithOIDCNonce(&genCtx, "/", "oidc-nonce-abc", true)
	require.NoError(t, err)

	var valCtx fasthttp.RequestCtx
	valCtx.Request.Header.SetCookie(CookieName, "tampered")

	_, _, err = ValidateWithOIDCNonce(&valCtx, encoded, true)
	require.ErrorIs(t, err, ErrInvalidState)
}

func TestGenerateOmitsOIDCNonceWhenEmpty(t *testing.T) {
	var ctx fasthttp.RequestCtx
	encoded, err := Generate(&ctx, "/", true)
	require.NoError(t, err)

	raw, err := base64.URLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "\"on\"", "plain Generate must not emit an OIDC nonce field")
}
