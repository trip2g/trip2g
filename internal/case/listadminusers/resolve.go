package listadminusers

import (
	"context"
	"fmt"
	"trip2g/internal/appresp"
	"trip2g/internal/db"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

type Env interface {
	ListAllUsers(ctx context.Context) ([]db.User, error)
}

type Request struct {
}

type Response struct {
	appresp.Response

	Rows []db.User
}

func Resolve(ctx context.Context, env Env, _ Request) (*Response, error) {
	response := Response{}
	response.Success = true
	response.Errors = make([]string, 0)

	users, err := env.ListAllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	response.Rows = users

	return &response, nil
}
