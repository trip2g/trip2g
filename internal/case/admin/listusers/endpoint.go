package listusers

import (
	"context"
	"net/http"
	"trip2g/internal/appreq"
)

type Endpoint struct{}

func (e Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	return Resolve(context.Background(), req.Env.(Env), Request{})
}

func (Endpoint) Path() string {
	return "/api/admin/listusers"
}

func (Endpoint) Method() string {
	return http.MethodGet
}
