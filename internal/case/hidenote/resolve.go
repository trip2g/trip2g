package hidenote

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

func Resolve(ctx context.Context, env Env, input model.HideNoteInput) (model.HideNoteOrErrorPayload, error) {
	params := db.HideNotePathParams{
		HiddenBy: sql.NullInt64{Valid: true, Int64: input.ApiKey.CreatedBy},
		Value:    input.Path,
	}

	err := env.HideNotePath(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to hide note: %w", err)
	}

	response := model.HideNotePayload{
		Success: true,
	}

	return &response, nil
}
