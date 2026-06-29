package rendernotepage_test

import (
	"context"
	"net/http"
	"testing"

	"trip2g/internal/appreq"
	"trip2g/internal/case/rendernotepage"
	"trip2g/internal/layoutloader"
	"trip2g/internal/model"
	"trip2g/internal/pagecache"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// These tests pin the "check page cache before Resolve" reorder: on an anonymous,
// gzip-accepting, cacheable request the page-cache HIT must be served BEFORE
// Resolve runs, skipping Resolve's per-request DB enrichment (telegram links).
// They reuse the helpers in endpoint_cache_test.go (cacheTestNote, cacheTestEnv,
// newReqCtx, runHandle, body, reqOpts, cacheLoaderEnv).

// Test a (the RED test): a cache HIT must NOT re-run Resolve's DB enrichment.
// GetTelegramPostLinksByNoteVersionID is the proxy for "Resolve ran": today the
// cache is consulted only AFTER Resolve, so the hit still pays that query — this
// assertion fails until the early fast-path short-circuits before Resolve.
func TestPageCache_EarlyHitSkipsResolveDBWork(t *testing.T) {
	_, views := cacheTestNote()
	env, pc, _ := cacheTestEnv(views, nil)

	// Prime the cache via the full path (Resolve runs, telegram links queried).
	runHandle(t, env, newReqCtx(reqOpts{acceptEncoding: "gzip"}), nil)
	require.Equal(t, 1, pc.Len())
	tgBefore := len(env.GetTelegramPostLinksByNoteVersionIDCalls())
	require.Equal(t, 1, tgBefore, "priming request runs full Resolve incl. telegram links")

	// Cache HIT: serve cached bytes WITHOUT re-running Resolve's DB enrichment.
	ctx := newReqCtx(reqOpts{acceptEncoding: "gzip"})
	runHandle(t, env, ctx, nil)
	require.Equal(t, "gzip", string(ctx.Response.Header.ContentEncoding()))
	require.Len(t, env.StoreCachedPageCalls(), 1, "hit must not re-fill the cache")
	require.Equal(t, tgBefore, len(env.GetTelegramPostLinksByNoteVersionIDCalls()),
		"cache HIT must skip GetTelegramPostLinksByNoteVersionID (Resolve DB enrichment)")
}

// Test b: the early-served bytes are byte-identical to the filling render.
func TestPageCache_EarlyHitBytesMatchRender(t *testing.T) {
	_, views := cacheTestNote()
	env, pc, _ := cacheTestEnv(views, nil)

	ctx1 := newReqCtx(reqOpts{acceptEncoding: "gzip"})
	runHandle(t, env, ctx1, nil) // fill
	ctx2 := newReqCtx(reqOpts{acceptEncoding: "gzip"})
	runHandle(t, env, ctx2, nil) // early hit

	require.Equal(t, ctx1.Response.Body(), ctx2.Response.Body(),
		"early cache hit serves byte-identical gzip to the filling render")
	require.Equal(t, body(t, ctx1), body(t, ctx2),
		"decompressed early-hit body equals the filling render")
	require.Equal(t, 1, pc.Len())
}

// Test c: a cache miss falls through to the full path (Resolve runs, telegram
// links queried, page rendered and cache filled).
func TestPageCache_MissFallsThrough(t *testing.T) {
	_, views := cacheTestNote()
	env, pc, _ := cacheTestEnv(views, nil)
	require.Equal(t, 0, pc.Len())

	ctx := newReqCtx(reqOpts{acceptEncoding: "gzip"})
	runHandle(t, env, ctx, nil)
	require.GreaterOrEqual(t, len(env.GetTelegramPostLinksByNoteVersionIDCalls()), 1,
		"a cache miss runs full Resolve incl. telegram links")
	require.Equal(t, 1, pc.Len(), "miss renders and fills the cache")
	require.Contains(t, string(body(t, ctx)), "<h1>Test</h1>")
}

// Test d: every bypass gate must take the full (Resolve) path — the early
// fast-path must never short-circuit them, even when a normal anon entry is
// already cached.
func TestPageCache_EarlyBypassTakesFullPath(t *testing.T) {
	// runAndAssertFull proves the request ran Resolve by observing a new
	// telegram-links DB query (which the early fast-path would have skipped).
	runAndAssertFull := func(t *testing.T, env *EnvMock, ctx *fasthttp.RequestCtx, token *usertoken.Data) {
		t.Helper()
		before := len(env.GetTelegramPostLinksByNoteVersionIDCalls())
		runHandle(t, env, ctx, token)
		require.Greater(t, len(env.GetTelegramPostLinksByNoteVersionIDCalls()), before,
			"bypass request must run full Resolve, not the early cache path")
	}

	t.Run("authenticated token", func(t *testing.T) {
		_, views := cacheTestNote()
		env, _, _ := cacheTestEnv(views, nil)
		runHandle(t, env, newReqCtx(reqOpts{acceptEncoding: "gzip"}), nil) // prime anon entry
		runAndAssertFull(t, env, newReqCtx(reqOpts{acceptEncoding: "gzip"}), &usertoken.Data{ID: 1, Role: "user"})
	})

	t.Run("version param", func(t *testing.T) {
		_, views := cacheTestNote()
		env, _, _ := cacheTestEnv(views, nil)
		runHandle(t, env, newReqCtx(reqOpts{acceptEncoding: "gzip"}), nil)
		runAndAssertFull(t, env, newReqCtx(reqOpts{acceptEncoding: "gzip", query: "version=latest"}), nil)
	})

	t.Run("non-gzip client", func(t *testing.T) {
		_, views := cacheTestNote()
		env, _, _ := cacheTestEnv(views, nil)
		runHandle(t, env, newReqCtx(reqOpts{acceptEncoding: "gzip"}), nil)
		runAndAssertFull(t, env, newReqCtx(reqOpts{}), nil) // no Accept-Encoding
	})

	t.Run("exotic accept-language", func(t *testing.T) {
		_, views := cacheTestNote()
		env, _, _ := cacheTestEnv(views, nil)
		runHandle(t, env, newReqCtx(reqOpts{acceptEncoding: "gzip"}), nil)
		runAndAssertFull(t, env, newReqCtx(reqOpts{acceptEncoding: "gzip", acceptLang: "de-DE,de;q=0.9"}), nil)
	})

	t.Run("setlang query", func(t *testing.T) {
		_, views := cacheTestNote()
		env, _, _ := cacheTestEnv(views, nil)
		runHandle(t, env, newReqCtx(reqOpts{acceptEncoding: "gzip"}), nil) // prime anon entry
		// ?setlang must hit handleSetLang (cookie + 302), never the early cache,
		// even though it shares the cache-key path with a normal request.
		ctx := newReqCtx(reqOpts{acceptEncoding: "gzip", query: "setlang=en"})
		runHandle(t, env, ctx, nil)
		require.Equal(t, http.StatusFound, ctx.Response.StatusCode(),
			"setlang must redirect (302), not be served from the early cache")
		require.NotEqual(t, "gzip", string(ctx.Response.Header.ContentEncoding()))
	})

	t.Run("personalized layout", func(t *testing.T) {
		layouts, err := layoutloader.Load(cacheLoaderEnv{}, []model.LayoutSourceFile{
			{ID: "/personal", Path: "_layouts/personal.html", Content: `<main>{{ if currentUser.IsAdmin() }}admin{{ end }}body</main>`},
		}, layoutloader.Options{})
		require.NoError(t, err)
		require.True(t, layouts.Map["/personal"].Personalized)

		note, views := cacheTestNote()
		note.Layout = "personal"
		env, pc, _ := cacheTestEnv(views, layouts)
		runAndAssertFull(t, env, newReqCtx(reqOpts{acceptEncoding: "gzip"}), nil)
		require.Equal(t, 0, pc.Len(), "personalized layout is never cached")
	})

	t.Run("note not found", func(t *testing.T) {
		_, views := cacheTestNote()
		env, _, _ := cacheTestEnv(views, nil)
		ctx := newReqCtx(reqOpts{path: "/does-not-exist", acceptEncoding: "gzip"})
		req := &appreq.Request{Req: ctx, Env: env}
		req.SetUserToken(nil)
		// render404 errors on this mock (no TrackNotFound); we only assert the
		// early path did NOT short-circuit: an unresolved path must not even
		// consult the page cache, and the status is 404.
		_, _ = rendernotepage.Endpoint{}.Handle(req)
		require.Empty(t, env.CachedPageCalls(),
			"unresolved path must not consult the page cache in the early path")
		require.Equal(t, http.StatusNotFound, ctx.Response.StatusCode())
	})
}

// Test e (conservative-design safety): the early path must enforce CanReadNote
// before serving a cache hit. Here a primed entry exists under the exact key the
// early path builds, but access is denied — the primed bytes must NOT be served;
// the request falls through to Resolve, which paywalls it.
func TestPageCache_EarlyAccessEnforced(t *testing.T) {
	_, views := cacheTestNote()
	env, pc, _ := cacheTestEnv(views, nil)

	// Prime a cache entry directly under the key the early path will build for a
	// default anon request (path /test-note, host example.com, version 42,
	// epoch 0, ui_lang "").
	primed := []byte("PRIMED-SHOULD-NOT-BE-SERVED")
	gz, err := pagecache.Gzip(primed)
	require.NoError(t, err)
	key := pagecache.Key{Path: "/test-note", Host: "example.com", NoteVersionID: 42, ConfigEpoch: 0, UILang: ""}
	pc.Set(key, gz)
	require.Equal(t, 1, pc.Len())

	// Access now denied for this anonymous viewer.
	env.CanReadNoteFunc = func(ctx context.Context, n *model.NoteView) (bool, error) { return false, nil }

	ctx := newReqCtx(reqOpts{acceptEncoding: "gzip"})
	runHandle(t, env, ctx, nil)
	require.NotContains(t, string(body(t, ctx)), "PRIMED-SHOULD-NOT-BE-SERVED",
		"early path must enforce CanReadNote before serving a cache hit")
}
