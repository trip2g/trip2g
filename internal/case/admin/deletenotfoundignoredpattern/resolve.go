package deletenotfoundignoredpattern

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	NotFoundIgnoredPatternByID(ctx context.Context, id int64) (db.NotFoundIgnoredPattern, error)
	DeleteNotFoundIgnoredPattern(ctx context.Context, id int64) error
	RefreshNotFoundTracker(ctx context.Context) error
}

func Resolve(ctx context.Context, env Env, input model.DeleteNotFoundIgnoredPatternInput) (model.DeleteNotFoundIgnoredPatternOrErrorPayload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin user token: %w", err)
	}

	// Check if the pattern exists
	_, err = env.NotFoundIgnoredPatternByID(ctx, input.ID)
	if err != nil {
		if db.IsNoFound(err) {
			return &model.ErrorPayload{Message: "pattern not found"}, nil
		}
		return nil, fmt.Errorf("failed to get pattern %d: %w", input.ID, err)
	}

	// Delete the pattern
	err = env.DeleteNotFoundIgnoredPattern(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete not found ignored pattern: %w", err)
	}

	err = env.RefreshNotFoundTracker(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh not found tracker: %w", err)
	}

	return &model.DeleteNotFoundIgnoredPatternPayload{
		DeletedID: input.ID,
	}, nil
}
