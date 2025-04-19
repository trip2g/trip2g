package me

import (
	"net/http"
	"trip2g/internal/appreq"
)

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	token, err := req.UserToken()
	if err != nil {
		return nil, err
	}

	request := Request{
		UserToken: token,
	}

	return Resolve(req.Req, req.Env.(Env), request)
}

func (*Endpoint) Path() string {
	return "/api/me"
}

func (*Endpoint) Method() string {
	return http.MethodGet
}
