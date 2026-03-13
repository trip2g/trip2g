package render404

import (
	"context"
	"net/http"
	"trip2g/internal/appreq"
	"trip2g/internal/case/renderlayout"
	"trip2g/internal/db"
	"trip2g/internal/defaulttemplate"
	"trip2g/internal/langdetect"
	"trip2g/internal/usertoken"
)

//go:generate go tool github.com/valyala/quicktemplate/qtc -dir=.

type Params struct {
	UserToken *usertoken.Data
}

type Env interface {
	TrackNotFound(path string, ip string)
}

func Handle(req *appreq.Request) (interface{}, error) {
	ctx := req.Req
	ctx.SetStatusCode(http.StatusNotFound)
	ctx.SetContentType("text/html; charset=utf-8")

	env, ok := req.Env.(Env)
	if !ok {
		return nil, appreq.ErrInvalidEnv
	}

	env.TrackNotFound(string(req.Req.Path()), req.Req.RemoteIP().String())

	token, err := req.UserToken()
	if err != nil {
		return nil, err
	}

	var jsURLs, cssURLs []string
	devMode := "false"
	injections := map[string][]db.HtmlInjection{}

	if rlEnv, ok := req.Env.(renderlayout.Env); ok {
		jsURLs = rlEnv.UserJSURLs()
		cssURLs = rlEnv.UserCSSURLs()
		if rlEnv.IsDevMode() {
			devMode = "true"
		}
		if active, err := rlEnv.ActiveHTMLInjections(context.Background()); err == nil {
			for _, inj := range active {
				injections[inj.Placement] = append(injections[inj.Placement], inj)
			}
		}
	}

	uiLang := langdetect.DetectPreferred(
		string(req.Req.Request.Header.Cookie("trip2g_lang")),
		string(req.Req.Request.Header.Peek("Accept-Language")),
	)

	dtCtx := &defaulttemplate.Ctx{
		Title:          "Page not found",
		JSURLs:         jsURLs,
		CSSURLs:        cssURLs,
		DevMode:        devMode,
		HTMLInjections: injections,
		UILang:         uiLang,
		UserToken:      token,
		NotFoundMode:   true,
	}

	defaulttemplate.WriteRender(ctx, dtCtx)
	return nil, nil
}
