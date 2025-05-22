package renderadminpage

import (
	"context"
	"trip2g/internal/usertoken"
)

type Env interface {
	AdminJSURL() string
}

type Request struct {
	UserToken *usertoken.Data
}

type Response struct {
	JSURL string
}

func Resolve(ctx context.Context, env Env, request Request) (*Response, error) {
	if !request.UserToken.IsAdmin() {
		return nil, nil
	}

	jsURL := env.AdminJSURL()

	return &Response{JSURL: jsURL}, nil
}
