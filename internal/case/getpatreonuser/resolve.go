package getpatreonuser

import (
	"context"
	"trip2g/internal/db"
)

type Env interface {
}

func Resolve(ctx context.Context, env Env, email string) (*db.User, error) {
	return nil, nil
}
