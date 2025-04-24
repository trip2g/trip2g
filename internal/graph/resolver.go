package graph

import (
	"context"
	"trip2g/internal/case/requestemailsignin"
	"trip2g/internal/case/signinbyemail"
	"trip2g/internal/case/signout"
	"trip2g/internal/db"
)

type Resolver struct {
	Env Env
}

type Env interface {
	ListAllUsers(ctx context.Context) ([]db.User, error)
	ListAllUserSubgraphAccesses(ctx context.Context) ([]db.UserSubgraphAccess, error)
	UserByID(ctx context.Context, id int64) (db.User, error)

	requestemailsignin.Env
	signinbyemail.Env
	signout.Env
}
