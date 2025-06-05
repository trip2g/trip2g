package hidenotes

import (
	"context"
	"database/sql"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

type Env interface {
	HideNotePath(ctx context.Context, params db.HideNotePathParams) error
}

func Resolve(ctx context.Context, env Env, input model.HideNotesInput) (model.HideNotesOrErrorPayload, error) {
	for _, path := range input.Paths {
		params := db.HideNotePathParams{
			HiddenBy: sql.NullInt64{Valid: true, Int64: int64(input.ApiKey.CreatedBy)},
			Value:    path,
		}

		err := env.HideNotePath(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to hide note path %s: %w", path, err)
		}
	}

	response := model.HideNotesPayload{
		Success: true,
	}

	return &response, nil
}
