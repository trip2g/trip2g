package getnotehashes

import (
	"net/http"
	"trip2g/internal/appreq"
)

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	env, ok := req.Env.(Env)
	if !ok {
		panic("invalid env")
	}

	return Resolve(req.Req, env, Request{})
}

func (*Endpoint) Path() string {
	return "getnotehashes"
}

func (*Endpoint) Method() string {
	return http.MethodGet
}
