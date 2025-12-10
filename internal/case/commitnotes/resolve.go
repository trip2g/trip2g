package commitnotes

import (
	"context"
	"fmt"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

type Env interface {
	PrepareLatestNotes(ctx context.Context, partial bool) (*appmodel.NoteViews, error)
	HandleLatestNotesAfterSave(ctx context.Context, changedPathIDs []int64) error
	InsertSubgraph(ctx context.Context, name string) error
}

type Payload = model.CommitNotesOrErrorPayload

func Resolve(ctx context.Context, env Env) (Payload, error) {
	nvs, err := env.PrepareLatestNotes(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare notes: %w", err)
	}

	for _, subgraph := range nvs.Subgraphs {
		insertErr := env.InsertSubgraph(ctx, subgraph.Name)
		if insertErr != nil {
			return nil, fmt.Errorf("failed to insert subgraph: %w", insertErr)
		}
	}

	err = env.HandleLatestNotesAfterSave(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to handle latest notes after save: %w", err)
	}

	return &model.CommitNotesPayload{Success: true}, nil
}
