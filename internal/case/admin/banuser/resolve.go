package banuser

import (
	"context"
	"fmt"
	"strings"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

type Env interface {
	BanUser(ctx context.Context, params db.BanUserParams) error
	ResetBanCache(ctx context.Context) error
}

type Input = model.BanUserInput
type Payload = model.BanUserOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	params := db.BanUserParams{
		UserID: input.UserID,
		Reason: input.Reason,
	}

	err := env.BanUser(ctx, params)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: user_bans.user_id") {
			return &model.ErrorPayload{Message: "User already banned"}, nil
		}

		return nil, fmt.Errorf("failed to ban user: %w", err)
	}

	err = env.ResetBanCache(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to reset ban cache: %w", err)
	}

	response := model.BanUserPayload{
		UserID: input.UserID,
	}

	return &response, nil
}
