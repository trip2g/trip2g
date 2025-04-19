package updatesubgraph

import (
	"context"
	"fmt"
	"trip2g/internal/appresp"
	"trip2g/internal/db"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

type Env interface {
	UpdateAdminSubgraph(ctx context.Context, arg db.UpdateAdminSubgraphParams) (db.Subgraph, error)
}

type Request struct {
	ID    int64
	Color *string
}

type Response struct {
	appresp.Response

	Row *db.Subgraph
}

func Resolve(ctx context.Context, env Env, req Request) (*Response, error) {
	response := Response{}
	response.Success = true
	response.Errors = make([]string, 0)

	params := db.UpdateAdminSubgraphParams{
		ID:    req.ID,
		Color: db.ToNullableString(req.Color),
	}

	subgraph, err := env.UpdateAdminSubgraph(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update subgraph: %w", err)
	}

	response.Row = &subgraph

	return &response, nil
}
