package rendernotepage

import (
	"context"
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

	resp, err := Resolve(context.Background(), req.Env.(Env), request)
	if err != nil {
		return nil, err
	}

	WriteLayoutHeader(req.Req, resp)
	WriteNote(req.Req, resp)
	WriteLayoutFooter(req.Req)

	return nil, nil
}

func (Endpoint) Path() string {
	return ""
}

func (Endpoint) Method() string {
	return http.MethodPost
}
