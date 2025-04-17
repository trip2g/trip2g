package listadminsubgraphs

import (
	"context"
	"fmt"
	"trip2g/internal/db"
)

//go:generate easyjson -all -snake_case -no_std_marshalers ./resolve.go

type Env interface {
	ListAdminSubgraphs(ctx context.Context) ([]db.Subgraph, error)
}

type Request struct {
}

type Response struct {
	Rows []db.Subgraph
}

func Resolve(ctx context.Context, env Env, _ Request) (*Response, error) {
	response := Response{
		Rows: make([]db.Subgraph, 0),
	}

	subgraphs, err := env.ListAdminSubgraphs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all subgraphs: %w", err)
	}

	response.Rows = subgraphs

	return &response, nil
}
