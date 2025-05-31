package updatenotegraphposition

import (
	"context"
	"database/sql"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	UpdateNoteGraphPositionByPathID(ctx context.Context, arg db.UpdateNoteGraphPositionByPathIDParams) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

func Resolve(ctx context.Context, env Env, input model.UpdateNoteGraphPositionInput) (model.UpdateNoteGraphPositionOrErrorPayload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	params := db.UpdateNoteGraphPositionByPathIDParams{
		GraphPositionX: sql.NullFloat64{Valid: true, Float64: input.X},
		GraphPositionY: sql.NullFloat64{Valid: true, Float64: input.Y},
		ID:             int64(input.PathID),
	}

	err = env.UpdateNoteGraphPositionByPathID(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update note graph position: %w", err)
	}

	payload := model.UpdateNoteGraphPositionPayload{
		PathID: int64(input.PathID),
	}

	return &payload, nil
}
