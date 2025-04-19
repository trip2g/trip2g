package me

import (
	"context"
	"trip2g/internal/db"
	"trip2g/internal/usertoken"
)

//go:generate easyjson -all -snake_case -no_std_marshalers ./resolve.go

type Env interface {
	GetUserByID(ctx context.Context, id int64) (db.User, error)
}

type Request struct {
	UserToken *usertoken.Data
}

type Response struct {
	User *db.User
}

func Resolve(ctx context.Context, env Env, req Request) (*Response, error) {
	response := Response{}

	if req.UserToken == nil {
		return &response, nil
	}

	user, err := env.GetUserByID(ctx, int64(req.UserToken.ID))
	if err != nil {
		return nil, err
	}

	response.User = &user

	return &response, nil
}
