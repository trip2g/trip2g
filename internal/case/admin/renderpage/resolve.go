package renderpage

import (
	"context"
	"trip2g/internal/usertoken"
)

type Env interface{}

type Request struct {
	UserToken *usertoken.Data
}

type Response struct{}

func Resolve(ctx context.Context, env Env, request Request) (Response, error) {
	return Response{}, nil
}
