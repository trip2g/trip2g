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
	ID     int64
	Color  string
	Hidden bool
}

type Input = Request
type Payload = model.UpdateSubgraphOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	return input.Resolve(ctx, env)
}

func (input *Request) Resolve(ctx context.Context, env Env) (Payload, error) {
	params := db.UpdateAdminSubgraphParams{
		ID: input.ID,

		Hidden: input.Hidden,
	}

	if input.Color != "" {
		params.Color = db.ToNullableString(&input.Color)
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
