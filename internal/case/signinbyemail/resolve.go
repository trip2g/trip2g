package signinbyemail

import (
	"context"
	"trip2g/internal/usertoken"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

type Env interface{}

type Request struct {
	Email string
}

type Response struct {
	tokenData *usertoken.Data

	Token string
}

func Resolve(ctx context.Context, env Env, _ Request) (*Response, error) {
	response := &Response{
		tokenData: &usertoken.Data{ID: 1, Opened: []string{"secondbrain"}},
	}

	return response, nil
}
