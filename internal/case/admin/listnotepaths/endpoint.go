package listnotepaths

import (
	"net/http"
	"trip2g/internal/appreq"
)

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	return Resolve(req.Req, req.Env.(Env), Request{})
}

func (*Endpoint) Path() string {
	return "/api/admin/listnotepaths"
}

func (*Endpoint) Method() string {
	return http.MethodGet
}
