package oauthstate

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestSafeRedirect(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "/"},
		{"/", "/"},
		{"/dashboard", "/dashboard"},
		{"/settings/profile", "/settings/profile"},
		// absolute URLs must be rejected
		{"https://evil.com", "/"},
		{"http://evil.com/steal", "/"},
		// protocol-relative
		{"//evil.com", "/"},
		{"//evil.com/path", "/"},
		// backslash variants some browsers normalise
		{"/\\evil.com", "/"},
		// no leading slash
		{"evil.com", "/"},
		{"relative/path", "/"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			require.Equal(t, tc.want, safeRedirect(tc.in))
		})
	}
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
