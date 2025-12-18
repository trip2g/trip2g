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
	ListUncommittedPaths(ctx context.Context) ([]int64, error)
	ClearUncommittedPaths(ctx context.Context) error
}

type Payload = model.CommitNotesOrErrorPayload

func Resolve(ctx context.Context, env Env) (Payload, error) {
	// Get uncommitted path IDs
	pathIDs, err := env.ListUncommittedPaths(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list uncommitted paths: %w", err)
	}

	_, err = env.PrepareLatestNotes(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare notes: %w", err)
	}

	err = env.HandleLatestNotesAfterSave(ctx, pathIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to handle latest notes after save: %w", err)
	}

	// Clear uncommitted paths after successful processing
	err = env.ClearUncommittedPaths(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to clear uncommitted paths: %w", err)
	}

	return &model.CommitNotesPayload{Success: true}, nil
}
