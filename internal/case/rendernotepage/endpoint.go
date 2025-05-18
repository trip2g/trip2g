package rendernotepage

import (
	"context"
	"errors"
	"net/http"
	"trip2g/internal/appreq"
)

//go:generate go run github.com/valyala/quicktemplate/qtc -dir=.

type Endpoint struct{}

func (e Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	token, err := req.UserToken()
	if err != nil {
		return nil, err
	}

	request := Request{
		Path: string(req.Req.URI().Path()),

		UserToken: token,
	}

	ctx := req.Req
	ctx.SetContentType("text/html; charset=utf-8")
	ctx.SetStatusCode(http.StatusOK)

	resp, err := Resolve(context.Background(), req.Env.(Env), request)
	if err != nil {
		paywallErr, paywallOk := err.(*PaywallError)
		if paywallOk {
			WriteLayoutHeader(ctx, resp)
			WritePayWall(ctx, resp, paywallErr)
			WriteLayoutFooter(ctx, resp)

			return nil, nil
		}

		if errors.Is(err, ErrNotFound) {
			ctx.SetStatusCode(http.StatusNotFound)

			WriteLayoutHeader(ctx, resp)
			WriteNotFound(ctx)
			WriteLayoutFooter(ctx, resp)

			return nil, nil
		}

		return nil, err
	}

	WriteLayoutHeader(ctx, resp)
	WriteNote(ctx, resp)
	WriteLayoutFooter(ctx, resp)

	return nil, nil
}

func (Endpoint) Path() string {
	return ""
}

func (Endpoint) Method() string {
	return http.MethodGet
}
