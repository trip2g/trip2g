package graph

import (
	"context"
	"database/sql"
	"trip2g/internal/db"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	Queries *db.Queries
	Conn    *sql.DB
	Env     Env
}

type Env interface {
	PrepareNotes(ctx context.Context, queries *db.Queries) error
}
