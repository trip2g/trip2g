package banuser

import (
	"context"
	"fmt"
	"trip2g/internal/graph/model"
)

type Env interface {
	BanUser(ctx context.Context, userID int64, bannedBy int64, reason string) error
}

func Resolve(ctx context.Context, env Env, req model.BanUserInput) (model.BanUserOrErrorPayload, error) {
	err := env.BanUser(ctx, int64(req.UserID), int64(req.BannedBy), req.Reason)
	if err != nil {
		return nil, fmt.Errorf("failed to ban user: %w", err)
	}

	response := model.BanUserPayload{
		UserID: req.UserID,
	}

	return &response, nil
}
