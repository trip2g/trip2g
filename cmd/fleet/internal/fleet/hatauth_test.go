package fleet

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"trip2g/internal/hotauthtoken"

	"github.com/stretchr/testify/require"
)

// TestHATAuthenticator_MintsValidAdminToken asserts the helper mints an HS256
// HAT JWT signed with the shared secret carrying the admin email and the
// AdminEnter flag, so /_system/hat self-provisions the admin user.
func TestHATAuthenticator_MintsValidAdminToken(t *testing.T) {
	const secret = "shared-jwt-secret"
	a := newHATAuthenticator("http://example.invalid", secret, "fleet@local", nil)

	token, err := a.mintToken()
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// A peer holding the same secret can verify it and read the claims.
	parsed, err := hotauthtoken.NewManager(hotauthtoken.Config{
		Secret:    secret,
		ExpiresIn: 5 * time.Minute,
	}).ParseToken(token)
	require.NoError(t, err)
	require.Equal(t, "fleet@local", parsed.Email)
	require.True(t, parsed.AdminEnter, "admin HAT must set AdminEnter so the user is elevated to admin")
}

// TestHATAuthenticator_ExchangeCapturesCookie asserts authenticate() POSTs the
// minted token to /_system/hat as a form field, does NOT follow the 302
// redirect, and captures the admin session cookie from the Set-Cookie header.
func TestHATAuthenticator_ExchangeCapturesCookie(t *testing.T) {
	var gotPath, gotCT, gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(b))
		gotToken = form.Get("token")
		http.SetCookie(w, &http.Cookie{Name: "trip2g_token", Value: "sess123", Path: "/"})
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	a := newHATAuthenticator(srv.URL, "secret", "fleet@local", srv.Client())
	require.NoError(t, a.authenticate(context.Background()))

	require.Equal(t, "/_system/hat", gotPath)
	require.Contains(t, gotCT, "application/x-www-form-urlencoded")
	require.NotEmpty(t, gotToken, "the HAT must be sent as the token form field")

	cookies := a.cached()
	var sess string
	for _, ck := range cookies {
		if ck.Name == "trip2g_token" {
			sess = ck.Value
		}
	}
	require.Equal(t, "sess123", sess, "the admin session cookie must be captured for reuse")
}

// TestHATAuthenticator_ExchangeNoCookieErrors asserts a HAT exchange that sets
// no session cookie (e.g. an auth failure) surfaces as an error rather than
// silently caching nothing.
func TestHATAuthenticator_ExchangeNoCookieErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("authentication failed"))
	}))
	defer srv.Close()

	a := newHATAuthenticator(srv.URL, "secret", "fleet@local", srv.Client())
	require.Error(t, a.authenticate(context.Background()))
	require.Empty(t, a.cached())
}
