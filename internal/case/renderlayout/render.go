package renderlayout

import (
	"errors"
	"trip2g/internal/appreq"
)

//go:generate go tool github.com/valyala/quicktemplate/qtc -dir=. -ext=html

type Params struct {
	Title   string
	JSURLs  []string
	CSSURLs []string
	DevMode string

	MetaDescription *string
	MetaRobots      string

	Client string
}

type Env interface {
	UserJSURLs() []string
	UserCSSURLs() []string
	IsDevMode() bool
}

var ErrMissingEnv = errors.New("missing env")

func Handle(req *appreq.Request, params Params, renderContent func()) (interface{}, error) {
	env, ok := req.Env.(Env)
	if !ok {
		return nil, ErrMissingEnv
	}

	if env.IsDevMode() {
		params.DevMode = "true"
	} else {
		params.DevMode = "false"
	}

	if len(params.JSURLs) == 0 {
		params.JSURLs = env.UserJSURLs()
	}

	if len(params.CSSURLs) == 0 {
		params.CSSURLs = env.UserCSSURLs()
	}

	req.Req.SetContentType("text/html; charset=utf-8")

	switch params.Client {
	case "tg":
		WriteBeginTGLayout(req.Req, &params)
		renderContent()
		WriteFinishTGLayout(req.Req, &params)

	default:
		WriteBeginLayout(req.Req, &params)
		renderContent()
		WriteFinishLayout(req.Req, &params)
	}

	return nil, nil
}
