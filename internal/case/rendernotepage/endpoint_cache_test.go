package rendernotepage_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"testing"

	"trip2g/internal/appreq"
	"trip2g/internal/case/rendernotepage"
	"trip2g/internal/db"
	"trip2g/internal/defaulttemplate"
	"trip2g/internal/features"
	"trip2g/internal/layoutloader"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/pagecache"
	"trip2g/internal/templateviews"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestMain(m *testing.M) {
	// The default template's paywall / sign-in / onboarding render paths call
	// ctx.T(), which needs the i18n bundle loaded.
	if err := defaulttemplate.Init(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// --- helpers ---------------------------------------------------------------

// cacheLoaderEnv satisfies layoutloader.Env for building test layouts.
type cacheLoaderEnv struct{}

func (cacheLoaderEnv) Logger() logger.Logger { return &logger.DummyLogger{} }
func (cacheLoaderEnv) IsDevMode() bool       { return false }

func cacheTestNote() (*model.NoteView, *model.NoteViews) {
	note := &model.NoteView{
		Path:          "/test-note",
		Title:         "Test Note",
		PathID:        1,
		VersionID:     42,
		Content:       []byte("# Test"),
		HTML:          "<h1>Test</h1>",
		Permalink:     "/test-note",
		Free:          true,
		InLinks:       map[string]struct{}{},
		Assets:        map[string]struct{}{},
		AssetReplaces: map[string]*model.NoteAssetReplace{},
	}
	views := &model.NoteViews{
		Map:     map[string]*model.NoteView{"/test-note": note},
		List:    []*model.NoteView{note},
		Version: "live",
	}
	return note, views
}

// cacheTestEnv wires an EnvMock for an anonymous, free, default-template note
// render. The page cache and a mutable config epoch are returned so tests can
// observe fills, force reloads, and bump the epoch.
func cacheTestEnv(views *model.NoteViews, layouts *model.Layouts) (*EnvMock, *pagecache.PageCache, *uint64) {
	pc := pagecache.New()
	epoch := new(uint64)
	if layouts == nil {
		layouts = &model.Layouts{Map: map[string]model.Layout{}}
	}
	env := &EnvMock{
		LoggerFunc:            func() logger.Logger { return &logger.DummyLogger{} },
		SiteConfigFunc:        func(ctx context.Context) model.SiteConfig { return model.SiteConfig{} },
		SiteTitleTemplateFunc: func() string { return "%s" },
		LiveNoteViewsFunc:     func() *model.NoteViews { return views },
		LatestNoteViewsFunc:   func() *model.NoteViews { return views },
		LayoutsFunc:           func() *model.Layouts { return layouts },
		CanReadNoteFunc:       func(ctx context.Context, note *model.NoteView) (bool, error) { return true, nil },
		GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
			return nil, nil
		},
		PublicURLFunc: func() string { return "https://example.com" },
		AssetURLFunc:  func(path string) string { return path },
		FeaturesFunc:  func() features.Features { return features.Features{} },
		ActiveHTMLInjectionsFunc: func(ctx context.Context) ([]db.HtmlInjection, error) {
			return nil, nil
		},
		NoteVersionEditorFunc: func(ctx context.Context, versionID int64) (*templateviews.NoteEditor, error) {
			return nil, nil
		},
		UserJSURLsFunc:       func() []string { return nil },
		UserCSSURLsFunc:      func() []string { return nil },
		UserInlineCSSFunc:    func() string { return "" },
		UserLocaleHashesFunc: func() map[string]string { return nil },
		IsDevModeFunc:        func() bool { return false },
		// Authenticated-viewer path (handleUserToken) — only exercised by the
		// authenticated-bypass test; harmless stubs otherwise.
		ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
			return nil, nil
		},
		RecordUserNoteViewFunc: func(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
		},
		LastUserNoteViewFunc: func(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
			return db.LastUserNoteViewRow{}, nil
		},
		ReaderMovesActiveFunc: func() bool { return false },
		ConfigEpochFunc:       func() uint64 { return *epoch },
		CachedPageFunc:        pc.Get,
		StoreCachedPageFunc:   pc.Set,
	}
	return env, pc, epoch
}

type reqOpts struct {
	path           string
	host           string
	acceptEncoding string
	acceptLang     string
	langCookie     string
	query          string
}

func newReqCtx(o reqOpts) *fasthttp.RequestCtx {
	if o.path == "" {
		o.path = "/test-note"
	}
	if o.host == "" {
		o.host = "example.com"
	}
	uri := o.path
	if o.query != "" {
		uri += "?" + o.query
	}
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI(uri)
	ctx.Request.Header.SetHost(o.host)
	if o.acceptEncoding != "" {
		ctx.Request.Header.Set("Accept-Encoding", o.acceptEncoding)
	}
	if o.acceptLang != "" {
		ctx.Request.Header.Set("Accept-Language", o.acceptLang)
	}
	if o.langCookie != "" {
		ctx.Request.Header.SetCookie("trip2g_lang", o.langCookie)
	}
	return ctx
}

func runHandle(t *testing.T, env *EnvMock, ctx *fasthttp.RequestCtx, token *usertoken.Data) {
	t.Helper()
	req := &appreq.Request{Req: ctx, Env: env}
	req.SetUserToken(token)
	_, err := rendernotepage.Endpoint{}.Handle(req)
	require.NoError(t, err)
}

func body(t *testing.T, ctx *fasthttp.RequestCtx) []byte {
	t.Helper()
	raw := ctx.Response.Body()
	if string(ctx.Response.Header.ContentEncoding()) != "gzip" {
		return raw
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	require.NoError(t, err)
	out, err := io.ReadAll(zr)
	require.NoError(t, err)
	return out
}

// --- tests -----------------------------------------------------------------

// Test 1: anonymous gzip GET of a free default-template note caches on first
// request and serves byte-identical (decompressed) bytes from cache on the
// second, with Content-Encoding: gzip.
func TestPageCache_AnonGzipHit(t *testing.T) {
	_, views := cacheTestNote()
	env, pc, _ := cacheTestEnv(views, nil)

	ctx1 := newReqCtx(reqOpts{acceptEncoding: "gzip"})
	runHandle(t, env, ctx1, nil)
	require.Equal(t, "gzip", string(ctx1.Response.Header.ContentEncoding()))
	// The fill path presets Content-Encoding (so CompressHandler is skipped) and
	// must re-add the Vary: Accept-Encoding that CompressHandler emits otherwise.
	require.Equal(t, "Accept-Encoding", string(ctx1.Response.Header.Peek("Vary")),
		"fill response must carry Vary: Accept-Encoding like the uncached path")
	require.Len(t, env.StoreCachedPageCalls(), 1, "first request fills the cache")
	require.Equal(t, 1, pc.Len())
	html1 := body(t, ctx1)
	require.Contains(t, string(html1), "<h1>Test</h1>")

	ctx2 := newReqCtx(reqOpts{acceptEncoding: "gzip"})
	runHandle(t, env, ctx2, nil)
	require.Equal(t, "gzip", string(ctx2.Response.Header.ContentEncoding()))
	require.Equal(t, "Accept-Encoding", string(ctx2.Response.Header.Peek("Vary")),
		"cache-hit response must carry Vary: Accept-Encoding like the uncached path")
	require.Len(t, env.StoreCachedPageCalls(), 1, "second request must NOT re-fill (served from cache)")
	require.Equal(t, html1, body(t, ctx2), "cached page decompresses identical to a fresh render")
	// The cached gzipped bytes are served verbatim.
	require.Equal(t, ctx1.Response.Body(), ctx2.Response.Body())
}

// Test 2: a request carrying an authenticated token is never served from cache
// and never populates it (admin/subscriber must not poison the anon cache).
func TestPageCache_AuthenticatedBypass(t *testing.T) {
	_, views := cacheTestNote()
	env, pc, _ := cacheTestEnv(views, nil)

	// Authenticated viewer (e.g. admin) — gzip-accepting all the same.
	ctx := newReqCtx(reqOpts{acceptEncoding: "gzip"})
	runHandle(t, env, ctx, &usertoken.Data{ID: 1, Role: "admin"})

	require.Empty(t, env.CachedPageCalls(), "authenticated request must not consult the cache")
	require.Empty(t, env.StoreCachedPageCalls(), "authenticated request must not populate the cache")
	require.Equal(t, 0, pc.Len())
	require.NotEqual(t, "gzip", string(ctx.Response.Header.ContentEncoding()),
		"authenticated render is left for CompressHandler, not pre-gzipped from cache")

	// And a subsequent anonymous request still gets a fresh fill (cache was not
	// poisoned by the admin render).
	ctxAnon := newReqCtx(reqOpts{acceptEncoding: "gzip"})
	runHandle(t, env, ctxAnon, nil)
	require.Equal(t, 1, pc.Len())
}

// Test 6a: ?version= (admin live/latest preview) bypasses the cache.
func TestPageCache_VersionParamBypass(t *testing.T) {
	_, views := cacheTestNote()
	env, pc, _ := cacheTestEnv(views, nil)

	ctx := newReqCtx(reqOpts{acceptEncoding: "gzip", query: "version=latest"})
	runHandle(t, env, ctx, nil)
	require.Empty(t, env.StoreCachedPageCalls())
	require.Equal(t, 0, pc.Len())
}

// Non-gzip clients take the normal (uncached) path.
func TestPageCache_NoGzipBypass(t *testing.T) {
	_, views := cacheTestNote()
	env, pc, _ := cacheTestEnv(views, nil)

	ctx := newReqCtx(reqOpts{}) // no Accept-Encoding
	runHandle(t, env, ctx, nil)
	require.Empty(t, env.StoreCachedPageCalls())
	require.Equal(t, 0, pc.Len())
	require.NotEqual(t, "gzip", string(ctx.Response.Header.ContentEncoding()))
}

// Test 4: a note push (reload) invalidates the cache; the next anon request
// re-renders. Here the version id changes (new content) so the key changes; a
// real reload also Clears the map. Either way the old entry is not served.
func TestPageCache_PushInvalidation(t *testing.T) {
	note, views := cacheTestNote()
	env, pc, _ := cacheTestEnv(views, nil)

	ctx1 := newReqCtx(reqOpts{acceptEncoding: "gzip"})
	runHandle(t, env, ctx1, nil)
	require.Len(t, env.StoreCachedPageCalls(), 1)

	// Simulate a push: content + version change, and the reload hook Clears.
	note.VersionID = 43
	note.HTML = "<h1>Updated</h1>"
	pc.Clear()
	require.Equal(t, 0, pc.Len())

	ctx2 := newReqCtx(reqOpts{acceptEncoding: "gzip"})
	runHandle(t, env, ctx2, nil)
	require.Len(t, env.StoreCachedPageCalls(), 2, "post-push request re-renders + re-fills")
	require.Contains(t, string(body(t, ctx2)), "<h1>Updated</h1>")
}

// Test 4b: the whole-cache Clear (map-swap) is load-bearing ON ITS OWN. Unlike
// TestPageCache_PushInvalidation, NO key field changes here — same NoteVersionID,
// same ConfigEpoch, same path/host/lang — so the key stays identical and the only
// thing that can force a re-render is Clear(). This pins the invalidation path for
// changes that do NOT move NoteVersionID (custom-layout edits whose layout version
// is not in the key; subgraph access changes at the same content version). Remove
// the pc.Clear() below and the second request hits the stale entry: the
// StoreCachedPageCalls()==2 assertion then fails.
func TestPageCache_ClearOnlyInvalidation(t *testing.T) {
	_, views := cacheTestNote()
	env, pc, _ := cacheTestEnv(views, nil)

	ctx1 := newReqCtx(reqOpts{acceptEncoding: "gzip"})
	runHandle(t, env, ctx1, nil)
	require.Len(t, env.StoreCachedPageCalls(), 1, "first request fills the cache")
	require.Equal(t, 1, pc.Len())

	// A reload that does NOT bump NoteVersionID still Clears the map.
	pc.Clear()
	require.Equal(t, 0, pc.Len())

	ctx2 := newReqCtx(reqOpts{acceptEncoding: "gzip"})
	runHandle(t, env, ctx2, nil)
	require.Len(t, env.StoreCachedPageCalls(), 2,
		"Clear() alone (no key change) must force a re-render + re-fill")
	require.Equal(t, 1, pc.Len())
}

// Test 5: a config change bumps ConfigEpoch so old entries are unreachable.
func TestPageCache_ConfigEpochBump(t *testing.T) {
	_, views := cacheTestNote()
	env, pc, epoch := cacheTestEnv(views, nil)

	ctx1 := newReqCtx(reqOpts{acceptEncoding: "gzip"})
	runHandle(t, env, ctx1, nil)
	require.Len(t, env.StoreCachedPageCalls(), 1)
	require.Equal(t, 1, pc.Len())

	*epoch++ // config change

	ctx2 := newReqCtx(reqOpts{acceptEncoding: "gzip"})
	runHandle(t, env, ctx2, nil)
	// Same path/note/lang but a new epoch => a different key => a miss + new fill.
	require.Len(t, env.StoreCachedPageCalls(), 2)
	require.Equal(t, 2, pc.Len(), "old-epoch entry remains but is unreachable; new entry added")
}

// Test 3: a note whose custom Jet layout is personalized is NOT cached; a note
// with a non-personalized custom layout IS cached.
func TestPageCache_PersonalizedLayoutBypass(t *testing.T) {
	layouts, err := layoutloader.Load(cacheLoaderEnv{}, []model.LayoutSourceFile{
		{ID: "/plain", Path: "_layouts/plain.html", Content: `<main>plain layout</main>`},
		{ID: "/personal", Path: "_layouts/personal.html", Content: `<main>{{ if currentUser.IsAdmin() }}admin{{ end }}body</main>`},
	}, layoutloader.Options{})
	require.NoError(t, err)
	require.False(t, layouts.Map["/plain"].Personalized)
	require.True(t, layouts.Map["/personal"].Personalized)

	t.Run("non-personalized custom layout is cached", func(t *testing.T) {
		note, views := cacheTestNote()
		note.Layout = "plain"
		env, pc, _ := cacheTestEnv(views, layouts)

		ctx := newReqCtx(reqOpts{acceptEncoding: "gzip"})
		runHandle(t, env, ctx, nil)
		require.Equal(t, 1, pc.Len(), "non-personalized layout fills the cache")
		require.Equal(t, "gzip", string(ctx.Response.Header.ContentEncoding()))
		require.Contains(t, string(body(t, ctx)), "plain layout")
	})

	t.Run("personalized custom layout is bypassed", func(t *testing.T) {
		note, views := cacheTestNote()
		note.Layout = "personal"
		env, pc, _ := cacheTestEnv(views, layouts)

		ctx := newReqCtx(reqOpts{acceptEncoding: "gzip"})
		runHandle(t, env, ctx, nil)
		require.Empty(t, env.StoreCachedPageCalls(), "personalized layout must not be cached")
		require.Equal(t, 0, pc.Len())
	})
}

// Test 6: paywall, sign-in wall and onboarding responses are bypassed — they
// return before the cacheable branch, so the cache is never consulted or filled.
func TestPageCache_NonNoteBranchesBypass(t *testing.T) {
	t.Run("paywall (non-free note, anon)", func(t *testing.T) {
		note, views := cacheTestNote()
		note.Free = false
		env, pc, _ := cacheTestEnv(views, nil)

		ctx := newReqCtx(reqOpts{acceptEncoding: "gzip"})
		runHandle(t, env, ctx, nil)
		require.Empty(t, env.StoreCachedPageCalls())
		require.Empty(t, env.CachedPageCalls())
		require.Equal(t, 0, pc.Len())
	})

	t.Run("sign-in wall (RequireSignin subgraph, anon)", func(t *testing.T) {
		note, views := cacheTestNote()
		note.Subgraphs = map[string]*model.NoteSubgraph{
			"members": {Name: "members", RequireSignin: true},
		}
		env, pc, _ := cacheTestEnv(views, nil)

		ctx := newReqCtx(reqOpts{acceptEncoding: "gzip"})
		runHandle(t, env, ctx, nil)
		require.Empty(t, env.StoreCachedPageCalls())
		require.Equal(t, 0, pc.Len())
	})

	t.Run("onboarding (no notes)", func(t *testing.T) {
		empty := &model.NoteViews{Map: map[string]*model.NoteView{}, Version: "live"}
		env, pc, _ := cacheTestEnv(empty, nil)

		ctx := newReqCtx(reqOpts{acceptEncoding: "gzip"})
		runHandle(t, env, ctx, nil)
		require.Empty(t, env.StoreCachedPageCalls())
		require.Equal(t, 0, pc.Len())
	})
}

// Test 7: UI-language keying — en, ru and "" are distinct cache entries, and an
// exotic Accept-Language bypasses the cache (no unbounded growth).
func TestPageCache_UILangKeying(t *testing.T) {
	_, views := cacheTestNote()
	env, pc, _ := cacheTestEnv(views, nil)

	// "" (no cookie / no Accept-Language)
	runHandle(t, env, newReqCtx(reqOpts{acceptEncoding: "gzip"}), nil)
	// en
	runHandle(t, env, newReqCtx(reqOpts{acceptEncoding: "gzip", acceptLang: "en-US,en;q=0.9"}), nil)
	// ru
	runHandle(t, env, newReqCtx(reqOpts{acceptEncoding: "gzip", langCookie: "ru"}), nil)
	require.Equal(t, 3, pc.Len(), "en, ru and \"\" are separate entries")

	storesBefore := len(env.StoreCachedPageCalls())

	// Exotic languages must not be cached (would otherwise thrash the cache).
	for _, lang := range []string{"de-DE,de;q=0.9", "fr-FR", "zh-CN", "es", "it"} {
		runHandle(t, env, newReqCtx(reqOpts{acceptEncoding: "gzip", acceptLang: lang}), nil)
	}
	require.Equal(t, 3, pc.Len(), "exotic languages do not create cache entries")
	require.Len(t, env.StoreCachedPageCalls(), storesBefore, "exotic languages are never stored")
}
