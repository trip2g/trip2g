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

func Resolve(_ context.Context, env Env, _ Request) (*Response, error) {
	response := &Response{
		tokenData: nil,
	}

	return response, nil
}
