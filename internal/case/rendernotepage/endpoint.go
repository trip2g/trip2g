package rendernotepage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"trip2g/internal/appreq"
	"trip2g/internal/case/render404"
	"trip2g/internal/case/renderlayout"
	"trip2g/internal/case/similarnotes"
	"trip2g/internal/db"
	"trip2g/internal/defaulttemplate"
	graphmodel "trip2g/internal/graph/model"
	"trip2g/internal/langdetect"
	"trip2g/internal/model"
	"trip2g/internal/ptr"
	"trip2g/internal/templateviews"

	"github.com/CloudyKit/jet/v6"
	"github.com/valyala/fasthttp"
)

//go:generate go tool github.com/valyala/quicktemplate/qtc -dir=. -ext=html

type Endpoint struct{}

//nolint:gocognit,funlen // high branch count is inherent to HTTP response dispatch; extracting sub-branches would obscure the flow
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

	// 301 redirect for non-canonical URL variants (alternate transliteration methods).
	if resp.Note != nil && resp.Note.AlternatePermalinks != nil && request.Path != resp.Note.Permalink {
		redirectURL := resp.Note.PermalinkEncoded()
		if qs := req.Req.URI().QueryString(); len(qs) > 0 {
			redirectURL += "?" + string(qs)
		}
		ctx.Response.Header.Set("Location", redirectURL)
		ctx.SetStatusCode(http.StatusMovedPermanently)
		return nil, nil
	}

	if resp.Note != nil && resp.Note.Redirect != nil {
		ctx.Response.Header.Set("Location", *resp.Note.Redirect)
		ctx.SetStatusCode(http.StatusFound)
		return nil, nil
	}

	if handleSetLang(ctx, resp) {
		return nil, nil
	}

	if redirectToRightLang(ctx, resp) {
		return nil, nil
	}

	// Short-circuit for raw file types (.canvas, .base, .excalidraw): these are
	// ingested infrastructure files, not content — paywall / signin checks do
	// not apply. Render the "not supported yet" placeholder regardless of err.
	if resp != nil && resp.Note != nil {
		if ext := unsupportedFileExt(resp.Note.Path); ext != "" {
			dtCtx := buildDefaultTemplateCtx(req, layoutParams, resp, env)
			dtCtx.UnsupportedFileExt = ext
			defaulttemplate.WriteRender(ctx, dtCtx)
			return nil, nil
		}
	}

	if resp.OnboardingMode {
		layoutParams.MetaRobots = "noindex"
		ctx.Response.Header.Set("Cache-Control", "no-store")

		dtCtx := buildDefaultTemplateCtx(req, layoutParams, resp, env)
		dtCtx.OnboardingMode = true
		defaulttemplate.WriteRender(ctx, dtCtx)
		return nil, nil
	}

	if err != nil {
		var signinWallErr *SigninWallError
		if errors.As(err, &signinWallErr) {
			// Sign-in wall: render auth form instead of content
			layoutParams.MetaRobots = "noindex, nofollow"
			ctx.Response.Header.Set("Cache-Control", "no-store")

			dtCtx := buildDefaultTemplateCtx(req, layoutParams, resp, env)
			dtCtx.SigninWallError = &defaulttemplate.SigninWallError{
				Note: resp.NoteView,
			}
			defaulttemplate.WriteRender(ctx, dtCtx)
			return nil, nil
		}

		var paywallErr *PaywallError
		if errors.As(err, &paywallErr) {
			layoutParams.MetaRobots = "noindex, nofollow"

			dtCtx := buildDefaultTemplateCtx(req, layoutParams, resp, env)
			dtCtx.PaywallError = &defaulttemplate.PaywallError{
				Note:          resp.NoteView,
				SubgraphsJSON: resp.NoteSubgraphsJSON(),
			}
			defaulttemplate.WriteRender(ctx, dtCtx)
			return nil, nil
		}

		if errors.Is(err, ErrNotFound) {
			ctx.SetStatusCode(http.StatusNotFound)

			return render404.Handle(req)
		}

		return nil, err
	}

	var layout string
	if resp.Note != nil {
		layout = resp.Note.Layout
	}
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

	dtCtx := buildDefaultTemplateCtx(req, layoutParams, resp, env)
	defaulttemplate.WriteRender(ctx, dtCtx)
	return nil, nil
}

func unsupportedFileExt(path string) string {
	switch {
	case strings.HasSuffix(path, ".canvas"):
		return ".canvas"
	case strings.HasSuffix(path, ".base"):
		return ".base"
	case strings.HasSuffix(path, ".excalidraw"):
		return ".excalidraw"
	}
	return ""
}

func (Endpoint) Path() string {
	return "" // means the default path that also resolves 404
}

func (Endpoint) Method() string {
	return http.MethodGet
}

const langCookieName = "trip2g_lang"
const langCookieMaxAge = 365 * 24 * 60 * 60 // 1 year in seconds

// handleSetLang processes the ?setlang=xxx query parameter.
// Sets the trip2g_lang cookie and redirects:
//   - to the language alternative if the current note has one for that lang
//   - otherwise to the current note's permalink (strips query params)
//
// Returns true if a redirect was sent.
func handleSetLang(ctx *fasthttp.RequestCtx, resp *Response) bool {
	setLang := strings.ToLower(strings.TrimSpace(string(ctx.QueryArgs().Peek("setlang"))))
	if setLang == "" {
		return false
	}

	c := fasthttp.AcquireCookie()
	c.SetKey(langCookieName)
	c.SetValue(setLang)
	c.SetPath("/")
	c.SetMaxAge(langCookieMaxAge)
	c.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	ctx.Response.Header.SetCookie(c)
	fasthttp.ReleaseCookie(c)

	redirectTo := ""
	if resp.Note != nil && resp.Note.Lang != setLang {
		if alt, ok := resp.Note.LangAlternatives[setLang]; ok && alt != nil {
			redirectTo = alt.Permalink
		}
	}
	if redirectTo == "" && resp.Note != nil {
		redirectTo = resp.Note.Permalink
	}
	if redirectTo == "" {
		redirectTo = "/"
	}

	ctx.Response.Header.Set("Location", redirectTo)
	ctx.SetStatusCode(http.StatusFound)
	return true
}

func redirectToRightLang(ctx *fasthttp.RequestCtx, resp *Response) bool {
	if resp.Note == nil || len(resp.Note.LangRedirects) == 0 || len(resp.Note.Lang) > 0 {
		return false
	}

	cookieVal := string(ctx.Request.Header.Cookie(langCookieName))
	acceptLang := string(ctx.Request.Header.Peek("Accept-Language"))
	preferred := langdetect.DetectPreferred(cookieVal, acceptLang)

	if preferred != "" && preferred != resp.Note.Lang {
		for _, lr := range resp.Note.LangRedirects {
			if lr.Note == resp.Note {
				continue
			}
			if lr.Lang == preferred {
				ctx.Response.Header.Set("Location", lr.URL)
				ctx.SetStatusCode(http.StatusFound)
				return true
			}
		}
	}

	return false
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
		if resp.UserToken.IsAdmin() {
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
			if resp.UserToken.IsAdmin() {
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

	headInjections := []db.HtmlInjection{}
	bodyEndInjections := []db.HtmlInjection{}
	if active, iErr := env.ActiveHTMLInjections(context.Background()); iErr == nil {
		for _, inj := range active {
			switch inj.Placement {
			case "head":
				headInjections = append(headInjections, inj)
			case "body_end":
				bodyEndInjections = append(bodyEndInjections, inj)
			}
		}
	}
	vars["htmlInjectionsHead"] = reflect.ValueOf(headInjections)
	vars["htmlInjectionsBodyEnd"] = reflect.ValueOf(bodyEndInjections)

	// Build the shared helper and expose two Jet namespaces:
	//   defaultTemplate — template chrome: {{ defaultTemplate.UserSpaceScripts() }}
	//   currentUser     — viewer role:     {{ currentUser.IsAdmin() }}
	usHelper := buildUserSpaceHelper(ctx, env, resp)
	defaultTemplateNS := reflect.ValueOf(usHelper.jetMap())
	currentUserNS := reflect.ValueOf(usHelper.currentUserJetMap())
	vars["defaultTemplate"] = defaultTemplateNS
	vars["currentUser"] = currentUserNS

	// Attach the admin-only "last edited by" resolver so layouts can render the
	// version author (gated on currentUser.IsAdmin()).
	injectLastEditedByResolver(env, resp)

	viewErr := layout.View.Execute(ctx, vars, resp)
	if viewErr != nil {
		if resp.UserToken.IsAdmin() {
			_, _ = ctx.WriteString(viewErr.Error())
			return true, nil
		}
		return false, fmt.Errorf("failed to execute view: %w", viewErr)
	}

	return true, nil
}

// injectLastEditedByResolver wires the admin-only "who pushed this version"
// resolver onto the template note. The resolver is lazy: the read query only
// runs when a layout actually calls note.LastEditedBy(), so it is safe to attach
// unconditionally.
//
// SECURITY: admin/editor-only data. Layouts MUST gate display on
// currentUser.IsAdmin() and never render it on public pages.
func injectLastEditedByResolver(env Env, resp *Response) {
	if resp == nil || resp.NoteView == nil || resp.Note == nil {
		return
	}
	versionID := resp.Note.VersionID
	if versionID <= 0 {
		return
	}
	resp.NoteView.SetLastEditedByResolver(func() *templateviews.NoteEditor {
		editor, err := env.NoteVersionEditor(context.Background(), versionID)
		if err != nil {
			env.Logger().Error("failed to resolve note version editor", "version_id", versionID, "error", err)
			return nil
		}
		return editor
	})
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

	// Explicit frontmatter og_image/cover wins; otherwise fall back to the first
	// image in the note body.
	if ogImage := resp.Note.OGImageURL(); ogImage != "" {
		tags["og:image"] = ogImage
	} else if resp.Note.FirstImage != nil {
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

// buildDefaultTemplateCtx constructs a *defaulttemplate.Ctx from the request, layout params, and response.
func buildDefaultTemplateCtx( //nolint:gocognit // template context assembly requires many optional fields
	req *appreq.Request, layoutParams renderlayout.Params, resp *Response, env Env,
) *defaulttemplate.Ctx {
	// Attach the admin-only "last edited by" resolver so the default template can
	// render the version author for normal notes (gated on admin in the template).
	injectLastEditedByResolver(env, resp)

	// Fetch JS/CSS URLs and dev mode from the renderlayout.Env interface.
	rlEnv, ok := req.Env.(renderlayout.Env)

	jsURLs := layoutParams.JSURLs
	cssURLs := layoutParams.CSSURLs
	inlineCSS := ""
	devMode := "false"

	if ok {
		if len(jsURLs) == 0 {
			jsURLs = rlEnv.UserJSURLs()
		}
		if len(cssURLs) == 0 {
			inlineCSS = rlEnv.UserInlineCSS()
		}
		if rlEnv.IsDevMode() {
			devMode = "true"
		}
	}

	// toc.js handles both TOC scrollspy and sidebar active-link highlighting;
	// load it on every note page.
	jsURLs = append(jsURLs, env.AssetURL("/assets/toc/toc.js"))

	// Append per-note widget glue conditionally, so a page only downloads the
	// script for widgets it actually uses. Emitted server-side (not via a client
	// loader) so the browser's preload scanner fetches them immediately.
	if note := resp.NoteView; note != nil {
		if note.HasCharts() {
			jsURLs = append(jsURLs, env.AssetURL("/assets/chart.js"))
		}
		if note.HasCodeLanguage("mermaid") {
			jsURLs = append(jsURLs, env.AssetURL("/assets/mermaid.js"))
		}
		if note.HasAnyCodeBlock() {
			jsURLs = append(jsURLs, env.AssetURL("/assets/codeblock.js"))
		}
	}

	// Build HTML injections map.
	injections := map[string][]db.HtmlInjection{}
	if ok {
		active, err := rlEnv.ActiveHTMLInjections(context.Background())
		if err == nil {
			for _, inj := range active {
				injections[inj.Placement] = append(injections[inj.Placement], inj)
			}
		} else {
			env.Logger().Error("failed to get active HTML injections", "error", err)
		}
	}

	// Convert hreflang slice.
	var hrefLangs []defaulttemplate.HrefLang
	for _, hl := range layoutParams.HrefLangs {
		hrefLangs = append(hrefLangs, defaulttemplate.HrefLang{
			Lang: hl.Lang,
			Href: hl.Href,
		})
	}

	dtCtx := &defaulttemplate.Ctx{
		Note:   resp.NoteView,
		Notes:  templateviews.NewNVS(resp.Notes, resp.DefaultVersion),
		Title:  layoutParams.Title,
		JSURLs: jsURLs,
		LocaleHashes: func() map[string]string {
			if ok {
				return rlEnv.UserLocaleHashes()
			}
			return nil
		}(),
		CSSURLs:         cssURLs,
		InlineCSS:       inlineCSS,
		DevMode:         devMode,
		MetaDescription: layoutParams.MetaDescription,
		MetaRobots:      layoutParams.MetaRobots,
		OGTags:          layoutParams.OGTags,
		HTMLInjections:  injections,
		HrefLangs:       hrefLangs,
		HTMLLang:        layoutParams.HTMLLang,
		PublicURL:       env.PublicURL(),
		SiteName:        defaulttemplate.DeriveSiteName(env.SiteConfig(context.Background()).SiteTitleTemplate, env.PublicURL()),
		UILang: langdetect.DetectPreferred(
			string(req.Req.Request.Header.Cookie(langCookieName)),
			string(req.Req.Request.Header.Peek("Accept-Language")),
		),
		EnableRSS:     env.SiteConfig(context.Background()).EnableRSS,
		UserToken:     resp.UserToken,
		TelegramLinks: resp.TelegramLinks,
		LayoutSections: func() []model.LayoutSectionEntry {
			if resp.Notes != nil {
				return resp.Notes.LayoutSections
			}
			return nil
		}(),
	}

	if resp.Note != nil && env.Features().VectorSearch.Enabled {
		simResults, err := similarnotes.Resolve(req.Req, env, graphmodel.SimilarNotesInput{
			Path:  resp.Note.Path,
			Limit: ptr.To(int32(10)),
		})
		if err != nil {
			env.Logger().Error("similar notes failed", "error", err)
		} else {
			for _, sn := range simResults {
				if sn.Note != nil && sn.Note.NoteView != nil {
					dtCtx.SimilarNotes = append(dtCtx.SimilarNotes, sn.Note.NoteView)
				}
			}
		}
	}

	return dtCtx
}
