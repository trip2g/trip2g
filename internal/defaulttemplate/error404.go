package defaulttemplate

import (
	"net/http"

	"trip2g/internal/db"
	"trip2g/internal/usertoken"

	"github.com/valyala/fasthttp"
)

// WriteNotFound sets status 404 and renders the generic not-found page.
func WriteNotFound(ctx *fasthttp.RequestCtx, env Env, token *usertoken.Data) {
	ctx.SetStatusCode(http.StatusNotFound)
	ctx.SetContentType("text/html; charset=utf-8")

	injections := map[string][]db.HtmlInjection{}
	if active, err := env.ActiveHTMLInjections(ctx); err == nil {
		for _, inj := range active {
			injections[inj.Placement] = append(injections[inj.Placement], inj)
		}
	}

	uiLang := uiLangFromCtx(ctx)
	dtCtx := &Ctx{
		Title:              T(uiLang, "not_found_title"),
		JSURLs:             env.UserJSURLs(),
		LocaleHashes:       env.UserLocaleHashes(),
		EditorJSURL:        env.EditorJSURL(),
		EditorLocaleHashes: env.EditorLocaleHashes(),
		CSSURLs:            env.UserCSSURLs(),
		DevMode:            devModeString(env.IsDevMode()),
		HTMLInjections:     injections,
		UILang:             uiLang,
		UserToken:          token,
		NotFoundMode:       true,
	}

	WriteRender(ctx, dtCtx)
}
