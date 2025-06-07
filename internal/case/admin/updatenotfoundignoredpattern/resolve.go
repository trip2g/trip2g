package updatenotfoundignoredpattern

import (
	"context"
	"fmt"
	"regexp"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	NotFoundIgnoredPatternByID(ctx context.Context, id int64) (db.NotFoundIgnoredPattern, error)
	UpdateNotFoundIgnoredPattern(ctx context.Context, arg db.UpdateNotFoundIgnoredPatternParams) (db.NotFoundIgnoredPattern, error)
}

func Resolve(ctx context.Context, env Env, input model.UpdateNotFoundIgnoredPatternInput) (model.UpdateNotFoundIgnoredPatternOrErrorPayload, error) {
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

	// Validate the new regex pattern
	_, err = regexp.Compile(input.Pattern)
	if err != nil {
		return &model.ErrorPayload{
			Message: fmt.Sprintf("invalid regex pattern: %s", err.Error()),
		}, nil
	}

	params := db.UpdateNotFoundIgnoredPatternParams{
		ID:      input.ID,
		Pattern: input.Pattern,
	}

	// Update the pattern
	updatedPattern, err := env.UpdateNotFoundIgnoredPattern(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update not found ignored pattern: %w", err)
	}

	return &model.UpdateNotFoundIgnoredPatternPayload{
		NotFoundIgnoredPattern: &updatedPattern,
	}, nil
}
