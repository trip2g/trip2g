package main

import (
	"context"
	"net/http"
	"testing"

	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func adminGraphQL(t *testing.T, s *store, setAuth func(*fasthttp.Request)) string {
	t.Helper()

	fctx := &fasthttp.RequestCtx{}
	fctx.Init(&fasthttp.Request{}, nil, nil)
	fctx.Request.Header.SetMethod(http.MethodPost)
	fctx.Request.SetRequestURI("/_system/graphql")
	fctx.Request.Header.SetHost("consumer.test")
	fctx.Request.Header.SetContentType("application/json")
	fctx.Request.SetBodyString(`{"query":"{admin{__typename}}"}`)
	setAuth(&fctx.Request)

	s.handler(fctx)

	return string(fctx.Response.Body())
}

// A session token minted while the user was an admin must stop granting admin
// once the admins row is gone, without waiting for the JWT to expire.
func TestRemovedAdminSessionLosesAdmin(t *testing.T) {
	s := newStore(t, "https://consumer.test")
	s.load(t)

	stored, err := s.app.tokenManager.Store(&fasthttp.RequestCtx{}, usertoken.Data{ID: int(s.adminID), Role: "admin"})
	require.NoError(t, err)

	auths := map[string]func(*fasthttp.Request){
		"cookie": func(r *fasthttp.Request) { r.Header.SetCookie("trip2g_token", stored.JWT) },
		"bearer": func(r *fasthttp.Request) { r.Header.Set("Authorization", "Bearer "+stored.JWT) },
	}

	for name, setAuth := range auths {
		require.Contains(t, adminGraphQL(t, s, setAuth), `"__typename":"AdminQuery"`, name+": admin before removal")
	}

	err = s.app.WriteQueries.DeleteAdmin(context.Background(), s.adminID)
	require.NoError(t, err)

	for name, setAuth := range auths {
		body := adminGraphQL(t, s, setAuth)
		require.NotContains(t, body, `"__typename":"AdminQuery"`, name+": admin after removal")
		require.Contains(t, body, "unauthorized", name)
	}
}
