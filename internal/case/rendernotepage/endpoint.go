package rendernotepage

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"trip2g/internal/appreq"
	"trip2g/internal/case/render404"
	"trip2g/internal/case/renderlayout"
	"trip2g/internal/langdetect"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/CloudyKit/jet/v6"
	"github.com/valyala/fasthttp"
)

//go:generate go tool github.com/valyala/quicktemplate/qtc -dir=. -ext=html

type Endpoint struct{}

func setLangCookie(ctx *fasthttp.RequestCtx, lang string) {
	c := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(c)
	c.SetKey("lang")
	c.SetValue(lang)
	c.SetPath("/")
	c.SetMaxAge(365 * 24 * 60 * 60) // 1 year.
	c.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	ctx.Response.Header.SetCookie(c)
}

func (e Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	token, err := req.UserToken()
	if err != nil {
		return nil, err
	}

	request := Request{
		Path:      string(req.Req.URI().Path()),
		Version:   string(req.Req.QueryArgs().Peek("version")),
		Referrer:  string(req.Req.Request.Header.Peek("Referer")),
		Host:      string(req.Req.Host()),
		UserToken: token,
	}

	ctx := req.Req
	ctx.SetContentType("text/html; charset=utf-8")
	ctx.SetStatusCode(http.StatusOK)

	layoutParams := renderlayout.Params{
		Client: string(req.Req.QueryArgs().Peek("client")),
	}

	env := req.Env.(Env)

	resp, err := Resolve(ctx, env, request)
	if resp != nil && resp.Note != nil {
		layoutParams.Title = resp.Title
		layoutParams.MetaDescription = resp.Note.Description
		layoutParams.OGTags = buildOGTags(req, env, resp)

		if resp.Note.Lang != "" {
			layoutParams.HTMLLang = resp.Note.Lang
		}

		layoutParams.HrefLangs = buildHrefLangs(env, resp.Note)
	}

	if resp.Note != nil && resp.Note.Redirect != nil {
		ctx.Response.Header.Set("Location", *resp.Note.Redirect)
		ctx.SetStatusCode(http.StatusFound)
		return nil, nil
	}

	nolang := ctx.QueryArgs().Has("nolang")

	// Language redirect: if note has lang_redirect targets, check user's preferred language.
	// ?nolang param (any value) disables redirect for this request (for authors/SEO tools).
	if resp.Note != nil && len(resp.Note.LangRedirects) > 0 && !nolang {

		cookieLang := string(ctx.Request.Header.Cookie("lang"))
		acceptLang := string(ctx.Request.Header.Peek("Accept-Language"))
		preferred := langdetect.DetectPreferred(cookieLang, acceptLang)

		if preferred != "" {
			for _, lr := range resp.Note.LangRedirects {
				if lr.Note == resp.Note {
					continue
				}
				if lr.Lang == preferred {
					setLangCookie(ctx, preferred)
					ctx.Response.Header.Set("Location", lr.URL)
					ctx.SetStatusCode(http.StatusFound)
					return nil, nil
				}
			}
		}
	}

	// Set lang cookie on any page that declares a language.
	// Skipped when ?nolang is set (debug/SEO mode).
	if resp.Note != nil && resp.Note.Lang != "" && !nolang {
		setLangCookie(ctx, resp.Note.Lang)
	}

	if resp.OnboardingMode {
		layoutParams.MetaRobots = "noindex"
		ctx.Response.Header.Set("Cache-Control", "no-store")

		return renderlayout.Handle(req, layoutParams, func() {
			WriteOnboarding(ctx, resp)
		})
	}

	if err != nil {
		var paywallErr *PaywallError
		if errors.As(err, &paywallErr) {
			layoutParams.MetaRobots = "noindex, nofollow"

			return renderlayout.Handle(req, layoutParams, func() {
				WritePayWall(ctx, resp, paywallErr)
			})
		}

		if errors.Is(err, ErrNotFound) {
			ctx.SetStatusCode(http.StatusNotFound)

			return render404.Handle(req)
		}

		return nil, err
	}

	turbo := len(ctx.Request.Header.Peek("X-Turbo")) > 0
	if turbo {
		ctx.Response.Header.Set("X-Turbo-Response", "true")
		WriteTurboNote(ctx, resp)
		return nil, nil
	}

	layout := resp.Note.Layout
	if layout == "" && resp.Config.DefaultLayout != "" {
		layout = resp.Config.DefaultLayout
	}

	if layout != "" {
		processed, layoutErr := renderLayout(ctx, env, resp, layout)
		if layoutErr != nil {
			return nil, layoutErr
		}

		if processed {
			return nil, nil
		}
	}

	return renderlayout.Handle(req, layoutParams, func() {
		WriteNote(ctx, resp)
	})
}

func (Endpoint) Path() string {
	return "" // means the default path that also resolves 404
}

func (Endpoint) Method() string {
	return http.MethodGet
}

//nolint:nonamedreturns // named returns required for defer/recover to set return values
func renderLayout(
	ctx *fasthttp.RequestCtx,
	env Env,
	resp *Response,
	layoutName string,
) (processed bool, err error) {
	layout, layoutExists := env.Layouts().Map["/"+layoutName]
	if !layoutExists {
		layoutNames := []string{}

		for name := range env.Layouts().Map {
			layoutNames = append(layoutNames, name)
		}

		env.Logger().Warn(
			"layout not found",
			"name", resp.Note.Layout,
			"available_layouts", layoutNames,
		)

		return false, nil
	}

	// Layout has parse error - show error to admin, fallback to default for others
	if layout.View == nil && len(layout.Warnings) > 0 {
		env.Logger().Error("layout has parse error", "name", layoutName, "warnings", layout.Warnings)
		if resp.IsAdmin {
			WriteLayoutError(ctx, resp, layoutName, layout.Warnings)
			return true, nil
		}
		// Non-admin: fallback to default rendering
		return false, nil
	}

	// Recover from template panics (e.g., type conversion errors in Jet)
	defer func() {
		if r := recover(); r != nil {
			env.Logger().Error("template panic", "layout", layoutName, "error", r)
			if resp.IsAdmin {
				_, _ = fmt.Fprintf(ctx, "Template error: %v", r)
				processed = true
				err = nil
			} else {
				processed = false
				err = fmt.Errorf("template panic: %v", r)
			}
		}
	}()

	vars := make(jet.VarMap)
	vars["note"] = reflect.ValueOf(resp.NoteView)
	vars["nvs"] = reflect.ValueOf(templateviews.NewNVS(resp.Notes, resp.DefaultVersion))
	vars["title"] = reflect.ValueOf(resp.Title)

	viewErr := layout.View.Execute(ctx, vars, resp)
	if viewErr != nil {
		if resp.IsAdmin {
			_, _ = ctx.WriteString(viewErr.Error())
			return true, nil
		}
		return false, fmt.Errorf("failed to execute view: %w", viewErr)
	}

	return true, nil
}

// buildHrefLangs builds the list of hreflang alternate links for a note.
// Returns nil if the note has no language group.
func buildHrefLangs(env Env, note *model.NoteView) []renderlayout.HrefLang {
	if note.LangGroup == nil {
		return nil
	}

	publicURL := env.PublicURL()
	group := note.LangGroup
	var hrefLangs []renderlayout.HrefLang

	// Add x-default (and lang tag if hub has a lang) for the hub page.
	hubURL := publicURL + group.Hub.Permalink
	if group.Hub.Lang == "" {
		hrefLangs = append(hrefLangs, renderlayout.HrefLang{
			Lang: "x-default",
			Href: hubURL,
		})
	} else {
		hrefLangs = append(hrefLangs, renderlayout.HrefLang{
			Lang: group.Hub.Lang,
			Href: hubURL,
		})
		hrefLangs = append(hrefLangs, renderlayout.HrefLang{
			Lang: "x-default",
			Href: hubURL,
		})
	}

	// Add each language version.
	for _, lr := range group.Versions {
		if lr.Note == group.Hub {
			continue
		}
		hrefLangs = append(hrefLangs, renderlayout.HrefLang{
			Lang: lr.Lang,
			Href: publicURL + lr.URL,
		})
	}

	return hrefLangs
}

// buildOGTags constructs Open Graph metadata for a rendered note.
// On custom domains it uses the domain-specific route URL rather than the canonical permalink.
func buildOGTags(req *appreq.Request, env Env, resp *Response) map[string]string {
	ogBaseURL, ogPath := ogURLForNote(req, env, resp.Note)

	tags := map[string]string{
		"og:url":  ogBaseURL + ogPath,
		"og:type": "article",

		// https://bureau.ru/soviet/20221027/
		"twitter:card": "summary_large_image",
	}

	// TODO: use a first paragraph as description
	// if this note is free.
	if resp.Note.Description != nil {
		tags["og:description"] = *resp.Note.Description
	}

	if resp.Note.FirstImage != nil {
		assetReplace, ok := resp.Note.AssetReplaces[*resp.Note.FirstImage]
		if ok && assetReplace != nil {
			tags["og:image"] = assetReplace.URL
		}
	}

	return tags
}

// ogURLForNote returns the base URL and path to use for og:url.
// On custom domains the domain-specific route URL is preferred over the canonical permalink.
func ogURLForNote(req *appreq.Request, env Env, note *model.NoteView) (string, string) {
	publicURL := env.PublicURL()
	permalink := note.Permalink

	requestHost := model.NormalizeDomain(string(req.Req.Host()))
	mainHost := model.NormalizeDomain(model.ExtractHost(publicURL))

	if requestHost == mainHost || requestHost == "" {
		return publicURL, permalink
	}

	// Custom domain request: find the best matching route.
	r := findRouteForHost(note.Routes, requestHost, string(req.Req.URI().Path()))
	if r == nil {
		return publicURL, permalink
	}

	scheme := "https"
	if strings.HasPrefix(publicURL, "http://") {
		scheme = "http"
	}

	routePath := permalink
	if r.Path != "" {
		routePath = r.Path
	}

	return scheme + "://" + string(req.Req.Host()), routePath
}

// findRouteForHost finds the best ParsedRoute for a given host and request path.
// Prefers an exact host+path match; falls back to the first host-only match.
func findRouteForHost(routes []model.ParsedRoute, host, requestPath string) *model.ParsedRoute {
	var firstMatch *model.ParsedRoute

	for i := range routes {
		r := &routes[i]
		if r.Host != host {
			continue
		}
		if firstMatch == nil {
			firstMatch = r
		}
		if r.Path == requestPath {
			return r
		}
	}

	return firstMatch
}
