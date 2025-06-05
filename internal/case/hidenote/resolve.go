package hidenote

import (
	"context"
	"database/sql"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	HideNotePath(ctx context.Context, params db.HideNotePathParams) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

func Resolve(ctx context.Context, env Env, input model.HideNoteInput) (model.HideNoteOrErrorPayload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	params := db.HideNotePathParams{
		HiddenBy: sql.NullInt64{Valid: true, Int64: int64(token.ID)},
		Value:    input.Path,
	}

	err = env.HideNotePath(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to hide note: %w", err)
	}

	response := model.HideNotePayload{
		Success: true,
	}

	return &response, nil
}
