package requestemailsignin

import (
	"context"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

type Env interface{}

type Request struct {
	Email string
}

type Response struct {
	Success bool
}

func Resolve(ctx context.Context, env Env, req Request) (*Response, error) {
	// Dummy implementation
	response := &Response{
		Success: true,
	}

	return response, nil
}
