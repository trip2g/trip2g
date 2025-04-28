package unbanuser

import (
	"context"
	"fmt"
	"trip2g/internal/graph/model"
)

type Env interface {
	UnbanUser(ctx context.Context, userID int64) error
}

func Resolve(ctx context.Context, env Env, req model.UnbanUserInput) (model.UnbanUserOrErrorPayload, error) {
	err := env.UnbanUser(ctx, int64(req.UserID))
	if err != nil {
		return nil, fmt.Errorf("failed to unban user: %w", err)
	}

	response := model.UnbanUserPayload{
		UserID: req.UserID,
	}

	return &response, nil
}
