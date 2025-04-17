package listadminsubgraphs

import (
	"context"
	"net/http"
	"trip2g/internal/appreq"
)

type Endpoint struct{}

func (e Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	var request Request
	return Resolve(context.Background(), req.Env.(Env), request)
}

func (Endpoint) Path() string {
	return "/api/listadminsubgraphs"
}

func (Endpoint) Method() string {
	return http.MethodGet
}
