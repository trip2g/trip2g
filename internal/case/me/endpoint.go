package me

import (
	"net/http"
	"trip2g/internal/appreq"
)

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	userID, err := req.UserToken()
	if err != nil {
		return nil, err
	}
	return Resolve(req.Req, req.Env.(Env), Request{UserID: int64(userID.ID)})
}

func (*Endpoint) Path() string {
	return "/api/me"
}

func (*Endpoint) Method() string {
	return http.MethodGet
}
