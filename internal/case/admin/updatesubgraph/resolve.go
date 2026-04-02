package updatesubgraph

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

type Env interface {
	UpdateAdminSubgraph(ctx context.Context, arg db.UpdateAdminSubgraphParams) (db.Subgraph, error)
	PrepareLatestNotes(ctx context.Context, partial bool) (*appmodel.NoteViews, error)
}

type Input = model.UpdateSubgraphInput
type Payload = model.UpdateSubgraphOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	params := db.UpdateAdminSubgraphParams{
		ID: input.ID,

		Hidden:        input.Hidden,
		RequireSignin: input.RequireSignin,
	}

	if input.Color != "" {
		params.Color = &input.Color
	}

	subgraph, err := env.UpdateAdminSubgraph(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update subgraph: %w", err)
	}

	// Reload notes so noteloader re-enriches NoteSubgraph.RequireSignin from DB.
	if _, err = env.PrepareLatestNotes(ctx, true); err != nil {
		return nil, fmt.Errorf("failed to reload notes after subgraph update: %w", err)
	}

	response := model.UpdateSubgraphPayload{
		Subgraph: &subgraph,
	}

	return &response, nil
}
