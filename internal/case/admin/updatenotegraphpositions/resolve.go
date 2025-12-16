package updatenotegraphpositions

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
)

type Env interface {
	UpdateNoteGraphPositionByPathID(ctx context.Context, arg db.UpdateNoteGraphPositionByPathIDParams) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Input = model.UpdateNoteGraphPositionsInput
type Payload = model.UpdateNoteGraphPositionsOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	pathsID := make([]int64, 0, len(input.Positions))

	for _, position := range input.Positions {
		params := db.UpdateNoteGraphPositionByPathIDParams{
			GraphPositionX: ptr.To(position.X),
			GraphPositionY: ptr.To(position.Y),
			ID:             position.PathID,
		}

		err = env.UpdateNoteGraphPositionByPathID(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to update note graph position for pathId %d: %w", position.PathID, err)
		}

		pathsID = append(pathsID, position.PathID)
	}

	payload := model.UpdateNoteGraphPositionsPayload{
		Success: true,
		PathsID: pathsID,
	}

	return &payload, nil
}
