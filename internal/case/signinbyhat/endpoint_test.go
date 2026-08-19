package signinbyhat_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"trip2g/internal/appreq"
	"trip2g/internal/case/signinbyhat"
	"trip2g/internal/db"
	"trip2g/internal/hotauthtoken"
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

// get runs the GET endpoint and returns the system message it asked the router
// for, failing the test if it did not ask for one.
func get(t *testing.T, env signinbyhat.Env, uri string) (*appreq.SystemMessageError, *fasthttp.RequestCtx) {
	t.Helper()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI(uri)

	_, err := (&signinbyhat.GetEndpoint{}).Handle(&appreq.Request{Env: env, Req: reqCtx})

	var sysErr *appreq.SystemMessageError
	require.ErrorAs(t, err, &sysErr)

	return sysErr, reqCtx
}

// Each failure a visitor can land in names its own message, so the page can say
// something they can act on instead of echoing the parser.
func TestGetEndpoint_FailuresAskForTheMatchingMessage(t *testing.T) {
	email := "user@example.com"

	tests := []struct {
		name string
		env  *mockEnv
		msg  string
		code int
	}{
		{
			name: "expired link",
			env:  &mockEnv{parseErr: fmt.Errorf("%w: token is expired", hotauthtoken.ErrExpiredToken)},
			msg:  "hat_expired",
			code: fasthttp.StatusUnauthorized,
		},
		{
			name: "broken link",
			env:  &mockEnv{parseErr: errors.New("invalid signature")},
			msg:  "hat_invalid",
			code: fasthttp.StatusUnauthorized,
		},
		{
			name: "nobody with that address",
			env: &mockEnv{
				hotAuthToken: &model.HotAuthToken{Email: email},
				userErr:      sql.ErrNoRows,
			},
			msg:  "hat_no_account",
			code: fasthttp.StatusUnauthorized,
		},
		{
			name: "our own failure",
			env: &mockEnv{
				hotAuthToken:  &model.HotAuthToken{Email: email},
				user:          db.User{ID: 1, Email: &email},
				setupTokenErr: errors.New("cookie error"),
			},
			msg:  "hat_failed",
			code: fasthttp.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sysErr, reqCtx := get(t, tt.env, "/_system/hat?token=some-token")

			require.Equal(t, tt.msg, sysErr.Msg)
			require.Equal(t, tt.code, sysErr.Code)
			require.Empty(t, reqCtx.Response.Header.Peek("Set-Cookie"))
		})
	}
}

// The cause is what makes the failure diagnosable in the log, so it has to
// survive on the error even though it never reaches the page.
func TestGetEndpoint_KeepsTheCauseForTheLog(t *testing.T) {
	cause := errors.New("invalid signature")

	sysErr, _ := get(t, &mockEnv{parseErr: cause}, "/_system/hat?token=bad-token")

	require.ErrorIs(t, sysErr, cause)
}

func TestGetEndpoint_MissingToken_BadRequest(t *testing.T) {
	sysErr, _ := get(t, &mockEnv{}, "/_system/hat")

	require.Equal(t, fasthttp.StatusBadRequest, sysErr.Code)
	require.Equal(t, "hat_invalid", sysErr.Msg)
}

func TestPostEndpoint_MissingToken_BadRequest(t *testing.T) {
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("POST")
	reqCtx.Request.SetRequestURI("/_system/hat")

	_, err := (&signinbyhat.Endpoint{}).Handle(&appreq.Request{Env: &mockEnv{}, Req: reqCtx})

	var sysErr *appreq.SystemMessageError
	require.ErrorAs(t, err, &sysErr)
	require.Equal(t, fasthttp.StatusBadRequest, sysErr.Code)
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
