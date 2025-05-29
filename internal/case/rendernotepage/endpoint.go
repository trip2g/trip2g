package rendernotepage

import (
	"context"
	"errors"
	"net/http"
	"trip2g/internal/appreq"
	"trip2g/internal/case/render404"
	"trip2g/internal/case/renderlayout"
)

//go:generate go run github.com/valyala/quicktemplate/qtc -dir=.

type Endpoint struct{}

func (e Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	token, err := req.UserToken()
	if err != nil {
		return nil, err
	}

	request := Request{
		Path:    string(req.Req.URI().Path()),
		Version: string(req.Req.QueryArgs().Peek("version")),

		UserToken: token,
	}

	ctx := req.Req
	ctx.SetContentType("text/html; charset=utf-8")
	ctx.SetStatusCode(http.StatusOK)

	layoutParams := renderlayout.Params{}

	resp, err := Resolve(context.Background(), req.Env.(Env), request)
	if resp != nil && resp.Note != nil {
		layoutParams.Title = resp.Note.Title
	}

	if err != nil {
		paywallErr, paywallOk := err.(*PaywallError)
		if paywallOk {
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
