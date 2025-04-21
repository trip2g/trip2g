package graph

import (
	"context"
	"trip2g/internal/db"
)

type Resolver struct {
	Env Env
}

type Env interface {
	ListAllUsers(ctx context.Context) ([]db.User, error)
}
