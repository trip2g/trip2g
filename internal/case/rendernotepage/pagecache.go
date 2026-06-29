package rendernotepage

import (
	"net/http"
	"time"

	"trip2g/internal/langdetect"
	"trip2g/internal/pagecache"

	"github.com/valyala/fasthttp"
)

// maxCacheableRender bounds how long a render may take and still be cached. It
// is defense-in-depth against fasthttp.TimeoutHandler: on timeout (60s prod /
// 10min dev) the client is handed a fresh ctx while the original handler
// goroutine keeps running on its now-abandoned ctx, which it owns exclusively
// (fasthttp 1.66.0 server.go:2459-2462 — the timed-out ctx is not reused, so
// there is no foreign-ctx contamination). The guard simply refuses to cache a
// render that reaches the fill path far slower than any healthy one: those bytes
// come from an incomplete/degraded render, not a representative page. Healthy
// renders complete in well under a second; this is comfortably below the timeout.
const maxCacheableRender = 5 * time.Second

// normalizeUILang returns (lang, true) only for the small whitelist the page
// cache supports. The rendered page embeds the RAW ui_lang value server-side,
// so the cache key MUST equal that value exactly; we therefore cache only the
// exact values {"en","ru",""} and bypass any other language (e.g. a bot's
// exotic Accept-Language) instead of risking a page whose embedded ui_lang
// differs from the key. This also bounds the key space to three lang buckets so
// exotic Accept-Language headers cannot thrash the cache.
func normalizeUILang(raw string) (string, bool) {
	switch raw {
	case "en", "ru", "":
		return raw, true
	default:
		return "", false
	}
}

// cacheDecision determines whether the current normal-served-note response is
// safe to serve from / store in the anonymous page cache, and builds its key.
//
// It is only reached on the normal note branch: redirects, onboarding, paywall,
// sign-in wall, 404 and unsupported-file responses already returned, so a
// hit/fill here is always a 200 HTML note page. Caching is refused (false) when
// ANY safety gate trips:
//   - the client does not accept gzip (we only cache the gzipped variant);
//   - the viewer is authenticated (resp.UserToken != nil) — could be admin,
//     subscriber, or otherwise personalized / paywall-exempt;
//   - an explicit ?version= switch is present (admin live/latest preview);
//   - the note's custom Jet layout is personalized (viewer/role-dependent);
//   - the UI language is outside the cacheable whitelist.
func cacheDecision(
	ctx *fasthttp.RequestCtx,
	env Env,
	resp *Response,
	request Request,
	layoutName string,
) (pagecache.Key, bool) {
	var zero pagecache.Key

	if resp == nil || resp.Note == nil {
		return zero, false
	}
	if resp.UserToken != nil {
		return zero, false
	}
	if request.Version != "" {
		return zero, false
	}
	if !ctx.Request.Header.HasAcceptEncoding("gzip") {
		return zero, false
	}
	if layoutIsPersonalized(env, layoutName) {
		return zero, false
	}

	uiLang, ok := normalizeUILang(langdetect.DetectPreferred(
		string(ctx.Request.Header.Cookie(langCookieName)),
		string(ctx.Request.Header.Peek("Accept-Language")),
	))
	if !ok {
		return zero, false
	}

	return pagecache.Key{
		Path:          request.Path,
		Host:          request.Host,
		NoteVersionID: resp.Note.VersionID,
		ConfigEpoch:   env.ConfigEpoch(),
		UILang:        uiLang,
	}, true
}

// layoutIsPersonalized reports whether the named custom layout exists, parses,
// and references viewer-dependent helpers. A missing or parse-error layout
// falls back to the role-uniform default template, so it is NOT personalized.
func layoutIsPersonalized(env Env, layoutName string) bool {
	if layoutName == "" {
		return false
	}
	layout, ok := env.Layouts().Map["/"+layoutName]
	if !ok || layout.View == nil {
		return false
	}
	return layout.Personalized
}

// writeCachedPage serves pre-gzipped bytes from the cache. It declares the gzip
// Content-Encoding so fasthttp.CompressHandler skips re-compression.
func writeCachedPage(ctx *fasthttp.RequestCtx, gz []byte) {
	ctx.SetStatusCode(http.StatusOK)
	ctx.SetContentType("text/html; charset=utf-8")
	ctx.Response.Header.SetContentEncoding("gzip")
	// CompressHandler's gzipBody early-returns once Content-Encoding is set, so it
	// never adds the Vary: Accept-Encoding it emits on the uncached path. Add it
	// here so cached responses carry the same Vary header as normal output.
	ctx.Response.Header.Add("Vary", "Accept-Encoding")
	ctx.Response.SetBody(gz)
}

// fillPageCache gzips the freshly rendered body once, stores it under key, and
// replaces the response body with the gzipped bytes + Content-Encoding so the
// filling request is served identically and CompressHandler is skipped. It is a
// no-op when caching was refused, the response is not a clean 200, or the render
// took anomalously long (a likely timed-out handler whose incomplete bytes must
// not enter the shared cache).
func fillPageCache(
	ctx *fasthttp.RequestCtx,
	env Env,
	key pagecache.Key,
	cacheable bool,
	renderStart time.Time,
) {
	if !cacheable {
		return
	}
	if ctx.Response.StatusCode() != http.StatusOK {
		return
	}
	if time.Since(renderStart) >= maxCacheableRender {
		return
	}

	gz, err := pagecache.Gzip(ctx.Response.Body())
	if err != nil {
		env.Logger().Error("page cache gzip failed", "error", err)
		return
	}

	env.StoreCachedPage(key, gz)
	ctx.Response.Header.SetContentEncoding("gzip")
	// See writeCachedPage: presetting Content-Encoding makes CompressHandler skip
	// adding Vary: Accept-Encoding, so add it to match the uncached path.
	ctx.Response.Header.Add("Vary", "Accept-Encoding")
	ctx.Response.SetBody(gz)
}
