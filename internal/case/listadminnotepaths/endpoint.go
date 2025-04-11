package listadminnotepaths

import (
	"net/http"
	"trip2g/internal/appreq"
)

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	return Resolve(req.Req, req.Env.(Env), Request{})
}

func (*Endpoint) Path() string {
	return "listadminnotepaths"
}

func (*Endpoint) Method() string {
	return http.MethodGet
}
