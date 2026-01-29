package rendernotepage

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"trip2g/internal/appreq"
	"trip2g/internal/case/render404"
	"trip2g/internal/case/renderlayout"
	"trip2g/internal/templateviews"

	"github.com/CloudyKit/jet/v6"
	"github.com/valyala/fasthttp"
)

//go:generate go tool github.com/valyala/quicktemplate/qtc -dir=. -ext=html

type Endpoint struct{}

func (e Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	token, err := req.UserToken()
	if err != nil {
		return nil, err
	}

	request := Request{
		Path:     string(req.Req.URI().Path()),
		Version:  string(req.Req.QueryArgs().Peek("version")),
		Referrer: string(req.Req.Request.Header.Peek("Referer")),

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

		layoutParams.OGTags = map[string]string{
			"og:url":  env.PublicURL() + resp.Note.Permalink,
			"og:type": "article",
		}

		if resp.Note.FirstImage != nil {
			assetReplace, ok := resp.Note.AssetReplaces[*resp.Note.FirstImage]
			if ok && assetReplace != nil {
				layoutParams.OGTags["og:image"] = assetReplace.URL
			}
		}
	}

	if resp.Note != nil && resp.Note.Redirect != nil {
		ctx.Response.Header.Set("Location", *resp.Note.Redirect)
		ctx.SetStatusCode(http.StatusFound)
		return nil, nil
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
	vars["note"] = reflect.ValueOf(templateviews.NewNote(resp.Note))
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
