package federationtopology

import (
	"context"
	"net/http"

	"trip2g/internal/appreq"
)

type Endpoint struct{}

func (*Endpoint) Path() string   { return "/_system/federation/admin" }
func (*Endpoint) Method() string { return http.MethodGet }

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	token, err := req.UserToken()
	if err != nil {
		return nil, err
	}
	if token == nil || !token.IsAdmin() {
		req.Req.SetStatusCode(http.StatusUnauthorized)
		return nil, nil
	}

	env := req.Env.(Env)
	resp, err := Resolve(context.Context(req.Req), env)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
