package updatesubgraph

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

type Env interface {
	UpdateAdminSubgraph(ctx context.Context, arg db.UpdateAdminSubgraphParams) (db.Subgraph, error)
}

type Request struct {
	ID    int64
	Color string
}

func (req *Request) Resolve(ctx context.Context, env Env) (model.UpdateSubgraphOrErrorPayload, error) {
	params := db.UpdateAdminSubgraphParams{
		ID: req.ID,
	}

	if req.Color != "" {
		params.Color = db.ToNullableString(&req.Color)
	}

	subgraph, err := env.UpdateAdminSubgraph(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update subgraph: %w", err)
	}

	response := model.UpdateSubgraphPayload{
		Subgraph: &subgraph,
	}

	return &response, nil
}
