package defaulttemplate

import (
	"bytes"
	"context"
	"html"
	"net/http"

	"trip2g/internal/db"
	"trip2g/internal/langdetect"

	"github.com/valyala/fasthttp"
)

const langCookieName = "trip2g_lang"

// Env declares the app dependencies WriteServerError and WriteNotFound need
// to build the shared page chrome (JS/CSS bundles, locale hashes, HTML
// injections, dev mode). Case packages already satisfy it via their own
// (larger) Env interfaces.
type Env interface {
	UserJSURLs() []string
	UserLocaleHashes() map[string]string
	UserCSSURLs() []string
	IsDevMode() bool
	ActiveHTMLInjections(ctx context.Context) ([]db.HtmlInjection, error)
}

// ServerErrorParams configures the 500 page rendered by WriteServerError.
type ServerErrorParams struct {
	Admin  bool
	Detail string // Jet/layout error text (with layout id + line context); shown to admins only
}

// WriteServerError sets status 500 and renders the generic error page.
// Admins see Detail (HTML-escaped) in a <pre> block on the same styled chrome
// as the public page; everyone else gets the generic page with no internal
// detail. If rendering through the template chrome itself fails (e.g. the
// chrome is what is broken), it falls back to a minimal self-contained page
// so the error response is never lost.
func WriteServerError(ctx *fasthttp.RequestCtx, env Env, params ServerErrorParams) {
	ctx.ResetBody()
	ctx.SetStatusCode(http.StatusInternalServerError)
	ctx.SetContentType("text/html; charset=utf-8")

	if !writeServerErrorChrome(ctx, env, params) {
		writeServerErrorFallback(ctx, params)
	}
}

// writeServerErrorChrome renders the 500 page through the default template
// (site header/footer/styling). It renders into a buffer first so a panic
// mid-render never leaks partial output; on panic it reports failure so the
// caller can fall back.
//
//nolint:nonamedreturns // named return required for defer/recover to report failure
func writeServerErrorChrome(ctx *fasthttp.RequestCtx, env Env, params ServerErrorParams) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()

	injections := map[string][]db.HtmlInjection{}
	if active, err := env.ActiveHTMLInjections(ctx); err == nil {
		for _, inj := range active {
			injections[inj.Placement] = append(injections[inj.Placement], inj)
		}
	}

	dtCtx := &Ctx{
		Title:           "Server error",
		JSURLs:          env.UserJSURLs(),
		LocaleHashes:    env.UserLocaleHashes(),
		CSSURLs:         env.UserCSSURLs(),
		DevMode:         devModeString(env.IsDevMode()),
		HTMLInjections:  injections,
		UILang:          uiLangFromCtx(ctx),
		ServerErrorMode: true,
	}
	if params.Admin {
		dtCtx.ServerErrorDetail = params.Detail
	}

	var buf bytes.Buffer
	WriteRender(&buf, dtCtx)
	_, _ = ctx.Write(buf.Bytes())
	return true
}

// writeServerErrorFallback is the last-resort path when the site's own template
// chrome itself fails to render (a critical, near-impossible case). It stays
// dependency-free: a tiny string, no template machinery. Admins get the
// HTML-escaped detail; everyone else just "Critical server error".
func writeServerErrorFallback(ctx *fasthttp.RequestCtx, params ServerErrorParams) {
	if params.Admin && params.Detail != "" {
		_, _ = ctx.WriteString("<h1>Critical server error</h1><pre>" + html.EscapeString(params.Detail) + "</pre>")
		return
	}
	_, _ = ctx.WriteString("Critical server error")
}

func devModeString(dev bool) string {
	if dev {
		return "true"
	}
	return "false"
}

func uiLangFromCtx(ctx *fasthttp.RequestCtx) string {
	return langdetect.DetectPreferred(
		string(ctx.Request.Header.Cookie(langCookieName)),
		string(ctx.Request.Header.Peek("Accept-Language")),
	)
}
