package signinbyhat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"trip2g/internal/appreq"
	"trip2g/internal/case/signinbyhat"
	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"
)

// mockEnvWithSetupFn extends mockEnv with a SetupUserToken override so tests
// can exercise the real cookie-setting path (usertoken.Manager.Store) instead
// of the fixed "mock-session-token" stub.
type mockEnvWithSetupFn struct {
	mockEnv
	setupTokenFn func(ctx context.Context, userID int64) (string, error)
}

func (m *mockEnvWithSetupFn) SetupUserToken(ctx context.Context, userID int64) (string, error) {
	if m.setupTokenFn != nil {
		return m.setupTokenFn(ctx, userID)
	}
	return m.mockEnv.SetupUserToken(ctx, userID)
}

func TestGetEndpoint_ValidToken_SetsCookieAndRedirectsToAdmin(t *testing.T) {
	tokenManager := usertoken.NewManager(usertoken.DefaultConfig())
	email := "owner@example.com"

	env := &mockEnvWithSetupFn{
		mockEnv: mockEnv{
			hotAuthToken: &model.HotAuthToken{Email: email},
			user:         db.User{ID: 1, Email: &email},
		},
		setupTokenFn: func(ctx context.Context, userID int64) (string, error) {
			rc, ok := ctx.(*fasthttp.RequestCtx)
			require.True(t, ok)
			data, err := tokenManager.Store(rc, usertoken.Data{ID: int(userID), Role: "user"})
			if err != nil {
				return "", err
			}
			return data.JWT, nil
		},
	}

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/_system/hat?token=valid-token")

	req := &appreq.Request{Env: env, Req: reqCtx}

	endpoint := &signinbyhat.GetEndpoint{}
	_, err := endpoint.Handle(req)
	require.NoError(t, err)

	require.Equal(t, fasthttp.StatusFound, reqCtx.Response.StatusCode())
	require.Equal(t, "/admin", string(reqCtx.Response.Header.Peek("Location")))
	require.NotEmpty(t, reqCtx.Response.Header.Peek("Set-Cookie"))
}

func TestGetEndpoint_InvalidToken_Rejected(t *testing.T) {
	env := &mockEnv{
		parseErr: errors.New("invalid signature"),
	}

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/_system/hat?token=bad-token")

	req := &appreq.Request{Env: env, Req: reqCtx}

	endpoint := &signinbyhat.GetEndpoint{}
	_, err := endpoint.Handle(req)
	require.NoError(t, err)

	require.Equal(t, fasthttp.StatusUnauthorized, reqCtx.Response.StatusCode())
	require.Empty(t, reqCtx.Response.Header.Peek("Set-Cookie"))
}

func TestGetEndpoint_MissingToken_BadRequest(t *testing.T) {
	env := &mockEnv{}

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/_system/hat")

	req := &appreq.Request{Env: env, Req: reqCtx}

	endpoint := &signinbyhat.GetEndpoint{}
	_, err := endpoint.Handle(req)
	require.NoError(t, err)

	require.Equal(t, fasthttp.StatusBadRequest, reqCtx.Response.StatusCode())
}

func TestPostEndpoint_ValidToken_StillRedirectsToRoot(t *testing.T) {
	email := "owner@example.com"
	env := &mockEnv{
		hotAuthToken: &model.HotAuthToken{Email: email},
		user:         db.User{ID: 1, Email: &email},
	}

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("POST")
	reqCtx.Request.SetRequestURI("/_system/hat")
	reqCtx.Request.Header.SetContentType("application/x-www-form-urlencoded")
	reqCtx.Request.SetBodyString("token=valid-token")

	req := &appreq.Request{Env: env, Req: reqCtx}

	endpoint := &signinbyhat.Endpoint{}
	_, err := endpoint.Handle(req)
	require.NoError(t, err)

	require.Equal(t, fasthttp.StatusFound, reqCtx.Response.StatusCode())
	require.Equal(t, "/", string(reqCtx.Response.Header.Peek("Location")))
}
