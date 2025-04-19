package me

import (
	"context"
	"trip2g/internal/db"
)

//go:generate easyjson -all -snake_case -no_std_marshalers ./resolve.go

type Env interface {
	GetUserByID(ctx context.Context, id int64) (db.User, error)
}

type Request struct {
	UserToken *usertoken.Data
}

type Response struct {
	User db.User
}

func Resolve(ctx context.Context, env Env, req Request) (*Response, error) {
	user, err := env.GetUserByID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	return &Response{User: user}, nil
}
