package renderadminpage

import (
	"net/http"
	"trip2g/internal/appreq"
	"trip2g/internal/case/render404"
)

//go:generate go run github.com/valyala/quicktemplate/qtc -dir=.

type Endpoint struct{}

func (e Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	token, err := req.UserToken()
	if err != nil {
		return nil, err
	}

	request := Request{
		UserToken: token,
	}

	ctx := req.Req

	resp, err := Resolve(ctx, req.Env.(Env), request)
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return render404.Handle(req)
	}

	ctx.SetContentType("text/html; charset=utf-8")
	ctx.SetStatusCode(http.StatusOK)

	WritePage(ctx, resp)

	return nil, nil
}

func (Endpoint) Path() string {
	return "/admin"
}

func (Endpoint) Method() string {
	return http.MethodGet
}
