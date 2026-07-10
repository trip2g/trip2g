package gitapi

import (
	"context"
	"encoding/base64"
	"testing"

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
