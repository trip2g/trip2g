package unbanuser

import (
	"context"
	"fmt"
	"trip2g/internal/graph/model"
)

type Env interface {
	UnbanUser(ctx context.Context, userID int64) error
	ResetBanCache(ctx context.Context) error
}

type Input = model.UnbanUserInput
type Payload = model.UnbanUserOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	err := env.UnbanUser(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to unban user: %w", err)
	}

	err = env.ResetBanCache(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to reset ban cache: %w", err)
	}

	response := model.UnbanUserPayload{
		UserID: input.UserID,
	}

	return &response, nil
}
