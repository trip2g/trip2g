package rendernotepage

import (
	"context"
	"errors"
	"net/http"
	"trip2g/internal/appreq"
	"trip2g/internal/case/render404"
	"trip2g/internal/case/renderlayout"
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

	resp, err := Resolve(context.Background(), req.Env.(Env), request)
	if resp != nil && resp.Note != nil {
		layoutParams.Title = resp.Note.Title
		layoutParams.MetaDescription = resp.Note.Description

		layoutParams.OGTags = map[string]string{
			"og:url":  "https://demo.trip2g.com" + resp.Note.Permalink,
			"og:type": "article",
		}

		if resp.Note.FirstImage != nil {
			assetReplace, ok := resp.Note.AssetReplaces[*resp.Note.FirstImage]
			if ok && assetReplace != nil {
				layoutParams.OGTags["og:image"] = assetReplace.URL
			}
		}
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

	if resp.Note.Redirect != nil {
		ctx.Response.Header.Set("Location", *resp.Note.Redirect)
		ctx.SetStatusCode(http.StatusFound)
		return nil, nil
	}

	turbo := len(ctx.Request.Header.Peek("X-Turbo")) > 0
	if turbo {
		ctx.Response.Header.Set("X-Turbo-Response", "true")
		WriteTurboNote(ctx, resp)
		return nil, nil
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
