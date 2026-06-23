package signinbyhat_test

import (
	"net/http"
	"testing"
	"trip2g/internal/appreq"
	"trip2g/internal/case/signinbyhat"
	"trip2g/internal/db"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// newGetRequest builds an appreq.Request wrapping a GET to the given URI with
// the provided Env, mirroring how the router invokes endpoint handlers.
func newGetRequest(uri string, env signinbyhat.Env) *appreq.Request {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI(uri)
	return &appreq.Request{Env: env, Req: ctx}
}

func TestGetEndpoint_MissingToken(t *testing.T) {
	env := &mockEnv{}
	ep := &signinbyhat.GetEndpoint{}

	req := newGetRequest("/_system/hat", env)
	_, err := ep.Handle(req)

	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, req.Req.Response.StatusCode())
	require.Equal(t, "missing token", string(req.Req.Response.Body()))
}

func TestGetEndpoint_ReadsTokenFromQuery_DefaultRedirect(t *testing.T) {
	email := "user@example.com"
	env := &mockEnv{
		hotAuthToken: &model.HotAuthToken{Email: email, AdminEnter: false},
		user:         db.User{ID: 123, Email: &email},
	}
	ep := &signinbyhat.GetEndpoint{}

	req := newGetRequest("/_system/hat?token=valid-token", env)
	_, err := ep.Handle(req)

	require.NoError(t, err)
	require.Equal(t, http.StatusFound, req.Req.Response.StatusCode())
	require.Equal(t, "/", string(req.Req.Response.Header.Peek("Location")))
}

func TestGetEndpoint_HonorsLocalNext(t *testing.T) {
	email := "user@example.com"
	env := &mockEnv{
		hotAuthToken: &model.HotAuthToken{Email: email, AdminEnter: false},
		user:         db.User{ID: 123, Email: &email},
	}
	ep := &signinbyhat.GetEndpoint{}

	req := newGetRequest("/_system/hat?token=valid-token&next=/admin", env)
	_, err := ep.Handle(req)

	require.NoError(t, err)
	require.Equal(t, http.StatusFound, req.Req.Response.StatusCode())
	require.Equal(t, "/admin", string(req.Req.Response.Header.Peek("Location")))
}

func TestGetEndpoint_SanitizesExternalNext(t *testing.T) {
	email := "user@example.com"

	cases := map[string]string{
		"absolute URL":     "/_system/hat?token=valid-token&next=https://evil.example.com",
		"protocol-relative": "/_system/hat?token=valid-token&next=//evil.example.com",
		"non-slash path":   "/_system/hat?token=valid-token&next=admin",
	}

	for name, uri := range cases {
		t.Run(name, func(t *testing.T) {
			env := &mockEnv{
				hotAuthToken: &model.HotAuthToken{Email: email, AdminEnter: false},
				user:         db.User{ID: 123, Email: &email},
			}
			ep := &signinbyhat.GetEndpoint{}

			req := newGetRequest(uri, env)
			_, err := ep.Handle(req)

			require.NoError(t, err)
			require.Equal(t, http.StatusFound, req.Req.Response.StatusCode())
			require.Equal(t, "/", string(req.Req.Response.Header.Peek("Location")))
		})
	}
}
