package gitapi

import (
	"context"
	"encoding/base64"
	"testing"

	"trip2g/internal/db"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestHandleRequestAuthenticatedUnknownPathReturnsClientError(t *testing.T) {
	env := &fakeEnv{}
	cfg := DefaultConfig()
	cfg.RepoPath = t.TempDir()

	api, err := New(context.Background(), cfg, env)
	require.NoError(t, err)
	require.NotNil(t, api)

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/_system/git/not-a-git-endpoint")
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	credentials := base64.StdEncoding.EncodeToString([]byte("user:valid-test-token"))
	ctx.Request.Header.Set("Authorization", "Basic "+credentials)

	var panicValue any
	var handled bool
	func() {
		defer func() {
			panicValue = recover()
		}()
		handled = api.HandleRequest(ctx)
	}()

	require.Nil(t, panicValue, "an authenticated unknown Git path must not call a nil handler")
	require.True(t, handled)
	require.Contains(t, []int{fasthttp.StatusNotFound, fasthttp.StatusMethodNotAllowed}, ctx.Response.StatusCode())
}

func boolPtr(b bool) *bool { return &b }

// TestTokenScopeDecision is the decision matrix behind scope enforcement:
// a token must hold can_pull for upload-pack (fetch/clone) and can_push for
// receive-pack (push); info/refs inherits the scope of its ?service=.
func TestTokenScopeDecision(t *testing.T) {
	cases := []struct {
		name    string
		relPath string
		service string
		canPull *bool
		canPush *bool
		allowed bool
	}{
		// can_pull only
		{"pull-only fetch ok", "/git-upload-pack", "", boolPtr(true), boolPtr(false), true},
		{"pull-only push denied", "/git-receive-pack", "", boolPtr(true), boolPtr(false), false},
		{"pull-only refs upload ok", "/info/refs", "git-upload-pack", boolPtr(true), boolPtr(false), true},
		{"pull-only refs receive denied", "/info/refs", "git-receive-pack", boolPtr(true), boolPtr(false), false},

		// can_push only
		{"push-only push ok", "/git-receive-pack", "", boolPtr(false), boolPtr(true), true},
		{"push-only fetch denied", "/git-upload-pack", "", boolPtr(false), boolPtr(true), false},
		{"push-only refs receive ok", "/info/refs", "git-receive-pack", boolPtr(false), boolPtr(true), true},
		{"push-only refs upload denied", "/info/refs", "git-upload-pack", boolPtr(false), boolPtr(true), false},

		// both scopes
		{"both fetch ok", "/git-upload-pack", "", boolPtr(true), boolPtr(true), true},
		{"both push ok", "/git-receive-pack", "", boolPtr(true), boolPtr(true), true},

		// neither scope (including nil columns = never granted)
		{"neither fetch denied", "/git-upload-pack", "", boolPtr(false), boolPtr(false), false},
		{"neither push denied", "/git-receive-pack", "", boolPtr(false), boolPtr(false), false},
		{"nil-scopes fetch denied", "/git-upload-pack", "", nil, nil, false},
		{"nil-scopes push denied", "/git-receive-pack", "", nil, nil, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token := db.GitToken{CanPull: tc.canPull, CanPush: tc.canPush}
			scope := requiredScope(tc.relPath, tc.service)
			require.NotEqual(t, scopeNone, scope, "endpoint must resolve to a scope")
			require.Equal(t, tc.allowed, tokenHasScope(token, scope))
		})
	}
}

// TestHandleRequestEnforcesTokenScope drives the full request path and asserts
// the HTTP status: allowed operations advertise refs (200), scoped-out
// operations are rejected with 403, and disabled/invalid tokens with 401.
func TestHandleRequestEnforcesTokenScope(t *testing.T) {
	cases := []struct {
		name       string
		service    string
		canPull    *bool
		canPush    *bool
		tokenErr   bool
		wantStatus int
	}{
		{"pull-only can fetch", "git-upload-pack", boolPtr(true), boolPtr(false), false, fasthttp.StatusOK},
		{"pull-only cannot enumerate push", "git-receive-pack", boolPtr(true), boolPtr(false), false, fasthttp.StatusForbidden},
		{"push-only can push", "git-receive-pack", boolPtr(false), boolPtr(true), false, fasthttp.StatusOK},
		{"push-only cannot enumerate fetch", "git-upload-pack", boolPtr(false), boolPtr(true), false, fasthttp.StatusForbidden},
		{"both fetch", "git-upload-pack", boolPtr(true), boolPtr(true), false, fasthttp.StatusOK},
		{"both push", "git-receive-pack", boolPtr(true), boolPtr(true), false, fasthttp.StatusOK},
		{"neither fetch", "git-upload-pack", boolPtr(false), boolPtr(false), false, fasthttp.StatusForbidden},
		{"neither push", "git-receive-pack", boolPtr(false), boolPtr(false), false, fasthttp.StatusForbidden},
		{"nil-scopes fetch", "git-upload-pack", nil, nil, false, fasthttp.StatusForbidden},
		{"disabled token fetch", "git-upload-pack", boolPtr(true), boolPtr(true), true, fasthttp.StatusUnauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := &fakeEnv{gitToken: db.GitToken{CanPull: tc.canPull, CanPush: tc.canPush}}
			if tc.tokenErr {
				env.gitTokenErr = context.DeadlineExceeded
			}
			api := newTestAPI(t, env)

			ctx := &fasthttp.RequestCtx{}
			ctx.Request.SetRequestURI("/_system/git/info/refs?service=" + tc.service)
			ctx.Request.Header.SetMethod(fasthttp.MethodGet)
			creds := base64.StdEncoding.EncodeToString([]byte("user:tok"))
			ctx.Request.Header.Set("Authorization", "Basic "+creds)

			require.True(t, api.HandleRequest(ctx))
			require.Equal(t, tc.wantStatus, ctx.Response.StatusCode())
		})
	}
}

// TestHandleRequestScopeDeniedOnPackEndpoints covers the POST pack endpoints
// directly (not just info/refs) for the denial directions.
func TestHandleRequestScopeDeniedOnPackEndpoints(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		canPull *bool
		canPush *bool
	}{
		{"pull-only denied on receive-pack", "/git-receive-pack", boolPtr(true), boolPtr(false)},
		{"push-only denied on upload-pack", "/git-upload-pack", boolPtr(false), boolPtr(true)},
		{"neither denied on upload-pack", "/git-upload-pack", boolPtr(false), boolPtr(false)},
		{"neither denied on receive-pack", "/git-receive-pack", boolPtr(false), boolPtr(false)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := &fakeEnv{gitToken: db.GitToken{CanPull: tc.canPull, CanPush: tc.canPush}}
			api := newTestAPI(t, env)

			ctx := &fasthttp.RequestCtx{}
			ctx.Request.SetRequestURI("/_system/git" + tc.path)
			ctx.Request.Header.SetMethod(fasthttp.MethodPost)
			creds := base64.StdEncoding.EncodeToString([]byte("user:tok"))
			ctx.Request.Header.Set("Authorization", "Basic "+creds)

			require.True(t, api.HandleRequest(ctx))
			require.Equal(t, fasthttp.StatusForbidden, ctx.Response.StatusCode())
		})
	}
}
