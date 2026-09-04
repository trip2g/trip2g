package appresp

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestRedirectKeepsTheLocationVerbatim(t *testing.T) {
	cases := []struct {
		name     string
		location string
	}{
		{"root", "/"},
		{"path with query", "/?berror=invalid_state"},
		{"path with fragment", "/boards/x#!userspace=open"},
		{"absolute elsewhere", "https://oauth.example.com/auth?client_id=x"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ctx fasthttp.RequestCtx
			ctx.Request.SetRequestURI("/_system/auth/oidc/callback")
			ctx.Request.SetHost("app.example.com")

			Redirect(&ctx, tc.location, http.StatusFound)

			require.Equal(t, http.StatusFound, ctx.Response.StatusCode())
			require.Equal(t, tc.location, string(ctx.Response.Header.Peek("Location")))
		})
	}
}

// A relative target must not be absolutised against the request, which behind a
// TLS-terminating proxy always says http — the downgrade that drops every
// Secure cookie the browser was just given.
func TestRedirectDoesNotAdoptTheRequestScheme(t *testing.T) {
	var ctx fasthttp.RequestCtx
	ctx.Request.SetRequestURI("/_system/auth/oidc/callback")
	ctx.Request.SetHost("app.example.com")

	Redirect(&ctx, "/", http.StatusFound)

	require.NotContains(t, string(ctx.Response.Header.Peek("Location")), "http://")
}
