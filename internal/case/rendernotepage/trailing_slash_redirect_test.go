package rendernotepage_test

import (
	"net/http"
	"testing"

	"trip2g/internal/appreq"
	"trip2g/internal/case/rendernotepage"
	"trip2g/internal/mdloader"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func trailingSlashEnv(t *testing.T) *EnvMock {
	t.Helper()

	views := loadTestVault(t, []mdloader.SourceFile{
		{Path: "blog.md", Content: []byte("---\nfree: true\n---\nBlog")},
		{Path: "index.md", Content: []byte("---\nfree: true\n---\nHome")},
		{Path: "docs.md", Content: []byte("---\nfree: true\nroute: /docs/\n---\nDocs served at a slash route")},
	})

	env, _, _ := cacheTestEnv(views, nil)

	return env
}

func handleIgnoringRender404(env *EnvMock, ctx *fasthttp.RequestCtx) {
	req := &appreq.Request{Req: ctx, Env: env}
	req.SetUserToken(nil)
	_, _ = rendernotepage.Endpoint{}.Handle(req)
}

func TestTrailingSlash_RedirectsToNoteWithoutSlash(t *testing.T) {
	env := trailingSlashEnv(t)

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		t.Run(method, func(t *testing.T) {
			ctx := newReqCtx(reqOpts{path: "/blog/"})
			ctx.Request.Header.SetMethod(method)
			runHandle(t, env, ctx, nil)

			require.Equal(t, http.StatusMovedPermanently, ctx.Response.StatusCode())
			require.Equal(t, "/blog", string(ctx.Response.Header.Peek("Location")))
		})
	}
}

func TestTrailingSlash_RedirectKeepsQueryString(t *testing.T) {
	env := trailingSlashEnv(t)

	ctx := newReqCtx(reqOpts{path: "/blog/", query: "a=1&b=two"})
	runHandle(t, env, ctx, nil)

	require.Equal(t, http.StatusMovedPermanently, ctx.Response.StatusCode())
	require.Equal(t, "/blog?a=1&b=two", string(ctx.Response.Header.Peek("Location")))
}

func TestTrailingSlash_NoNoteWithoutSlashStays404(t *testing.T) {
	env := trailingSlashEnv(t)

	ctx := newReqCtx(reqOpts{path: "/missing/"})
	handleIgnoringRender404(env, ctx)

	require.Equal(t, http.StatusNotFound, ctx.Response.StatusCode())
	require.Empty(t, ctx.Response.Header.Peek("Location"))
}

func TestTrailingSlash_RootIsNotRedirected(t *testing.T) {
	env := trailingSlashEnv(t)

	ctx := newReqCtx(reqOpts{path: "/"})
	runHandle(t, env, ctx, nil)

	require.Equal(t, http.StatusOK, ctx.Response.StatusCode())
	require.Empty(t, ctx.Response.Header.Peek("Location"))
}

func TestTrailingSlash_SystemPathIsNotRedirected(t *testing.T) {
	env := trailingSlashEnv(t)

	ctx := newReqCtx(reqOpts{path: "/_system/blog/"})
	handleIgnoringRender404(env, ctx)

	require.Equal(t, http.StatusNotFound, ctx.Response.StatusCode())
	require.Empty(t, ctx.Response.Header.Peek("Location"))
}

func TestTrailingSlash_PostIsNotRedirected(t *testing.T) {
	env := trailingSlashEnv(t)

	ctx := newReqCtx(reqOpts{path: "/blog/"})
	ctx.Request.Header.SetMethod(http.MethodPost)
	handleIgnoringRender404(env, ctx)

	require.Equal(t, http.StatusNotFound, ctx.Response.StatusCode())
	require.Empty(t, ctx.Response.Header.Peek("Location"))
}

func TestTrailingSlash_NoteServedAtSlashPathIsNotRedirected(t *testing.T) {
	env := trailingSlashEnv(t)

	ctx := newReqCtx(reqOpts{path: "/docs/"})
	runHandle(t, env, ctx, nil)

	require.Equal(t, http.StatusOK, ctx.Response.StatusCode())
	require.Empty(t, ctx.Response.Header.Peek("Location"))
}
