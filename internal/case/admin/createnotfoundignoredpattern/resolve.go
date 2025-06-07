package createnotfoundignoredpattern

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
	InsertNotFoundIgnoredPattern(ctx context.Context, arg db.InsertNotFoundIgnoredPatternParams) (db.NotFoundIgnoredPattern, error)
	RefreshNotFoundTracker(ctx context.Context) error
}

func Resolve(ctx context.Context, env Env, input model.CreateNotFoundIgnoredPatternInput) (model.CreateNotFoundIgnoredPatternOrErrorPayload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin user token: %w", err)
	}

	// Validate the regex pattern
	_, err = regexp.Compile(input.Pattern)
	if err != nil {
		return &model.ErrorPayload{
			Message: fmt.Sprintf("invalid regex pattern: %s", err.Error()),
		}, nil
	}

	params := db.InsertNotFoundIgnoredPatternParams{
		Pattern:   input.Pattern,
		CreatedBy: int64(token.ID),
	}

	// Create the ignored pattern
	pattern, err := env.InsertNotFoundIgnoredPattern(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to insert not found ignored pattern: %w", err)
	}

	err = env.RefreshNotFoundTracker(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh not found tracker: %w", err)
	}

	payload := model.CreateNotFoundIgnoredPatternPayload{
		NotFoundIgnoredPattern: &pattern,
	}

	return &payload, nil
}
