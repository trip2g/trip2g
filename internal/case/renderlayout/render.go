package renderlayout

import (
	"context"
	"errors"
	"fmt"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
)

//go:generate go tool github.com/valyala/quicktemplate/qtc -dir=. -ext=html

// HrefLang represents a single hreflang alternate link.
type HrefLang struct {
	Lang string // "en", "ru", "x-default"
	Href string // full URL including scheme+host
}

type Params struct {
	Title   string
	JSURLs  []string
	CSSURLs []string
	DevMode string

	MetaDescription *string
	MetaRobots      string

	OGTags map[string]string

	Client string

	HTMLInjections map[string][]db.HtmlInjection

	HrefLangs []HrefLang
	HTMLLang  string // for <html lang="xx">, defaults to "ru" in template
}

func htmlLang(params *Params) string {
	if params.HTMLLang != "" {
		return params.HTMLLang
	}
	return "ru"
}

type Env interface {
	UserJSURLs() []string
	UserCSSURLs() []string
	IsDevMode() bool
	ActiveHTMLInjections(ctx context.Context) ([]db.HtmlInjection, error)
}

var ErrMissingEnv = errors.New("missing env")

const (
	injectionPlaceholderHead    = "head"
	injectionPlaceholderBodyEnd = "body_end"
)

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

	htmlInjections, err := env.ActiveHTMLInjections(req.Req)
	if err != nil {
		return nil, fmt.Errorf("failed to get active HTML injections: %w", err)
	}

	// TODO: cache it
	params.HTMLInjections = make(map[string][]db.HtmlInjection)

	for _, injection := range htmlInjections {
		placement := injection.Placement
		params.HTMLInjections[placement] = append(params.HTMLInjections[placement], injection)
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
