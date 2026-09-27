package rendernotepage_test

import (
	"testing"

	"trip2g/internal/layoutloader"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

const noIndexMeta = `<meta name="robots" content="noindex">`

func TestNoIndexNote(t *testing.T) {
	t.Run("default template prints meta robots and drops JSON-LD", func(t *testing.T) {
		note, views := cacheTestNote()
		note.NoIndex = true
		env, _, _ := cacheTestEnv(views, nil)

		ctx := newReqCtx(reqOpts{})
		runHandle(t, env, ctx, nil)

		html := string(body(t, ctx))
		require.Contains(t, html, noIndexMeta)
		require.NotContains(t, html, "application/ld+json")
		require.Equal(t, "noindex", string(ctx.Response.Header.Peek("X-Robots-Tag")))
	})

	t.Run("an indexable note gets neither", func(t *testing.T) {
		_, views := cacheTestNote()
		env, _, _ := cacheTestEnv(views, nil)

		ctx := newReqCtx(reqOpts{})
		runHandle(t, env, ctx, nil)

		html := string(body(t, ctx))
		require.NotContains(t, html, `name="robots"`)
		require.Contains(t, html, "application/ld+json")
		require.Empty(t, ctx.Response.Header.Peek("X-Robots-Tag"))
	})

	t.Run("cached response keeps the header", func(t *testing.T) {
		note, views := cacheTestNote()
		note.NoIndex = true
		env, pc, _ := cacheTestEnv(views, nil)

		fill := newReqCtx(reqOpts{acceptEncoding: "gzip"})
		runHandle(t, env, fill, nil)
		require.Equal(t, 1, pc.Len())

		hit := newReqCtx(reqOpts{acceptEncoding: "gzip"})
		runHandle(t, env, hit, nil)
		require.Len(t, env.StoreCachedPageCalls(), 1, "second request is served from the cache")
		require.Equal(t, "noindex", string(hit.Response.Header.Peek("X-Robots-Tag")))
		require.Contains(t, string(body(t, hit)), noIndexMeta)
	})

	t.Run("custom layout gets the header", func(t *testing.T) {
		layouts, err := layoutloader.Load(cacheLoaderEnv{}, []model.LayoutSourceFile{
			{ID: "/plain", Path: "_layouts/plain.html", Content: `<main>plain layout</main>`},
		}, layoutloader.Options{})
		require.NoError(t, err)

		note, views := cacheTestNote()
		note.NoIndex = true
		note.Layout = "plain"
		env, _, _ := cacheTestEnv(views, layouts)

		ctx := newReqCtx(reqOpts{})
		runHandle(t, env, ctx, nil)

		require.Contains(t, string(body(t, ctx)), "plain layout")
		require.Equal(t, "noindex", string(ctx.Response.Header.Peek("X-Robots-Tag")))
	})

	t.Run("paywall keeps its stricter meta robots", func(t *testing.T) {
		note, views := cacheTestNote()
		note.NoIndex = true
		note.Free = false
		env, _, _ := cacheTestEnv(views, nil)

		ctx := newReqCtx(reqOpts{})
		runHandle(t, env, ctx, nil)

		require.Contains(t, string(body(t, ctx)), `<meta name="robots" content="noindex, nofollow">`)
	})
}
