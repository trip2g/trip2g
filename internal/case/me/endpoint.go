package me

import (
	"net/http"
	"trip2g/internal/appreq"
)

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	userID := req.UserID // Assuming UserID is part of the request context
	return Resolve(req.Req, req.Env.(Env), Request{UserID: userID})
}

func (*Endpoint) Path() string {
	return "/api/me"
}

func (*Endpoint) Method() string {
	return http.MethodGet
}
