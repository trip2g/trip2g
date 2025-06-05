package updateredirect

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	UpdateRedirect(ctx context.Context, params db.UpdateRedirectParams) (db.Redirect, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

func Resolve(ctx context.Context, env Env, input model.UpdateRedirectInput) (model.UpdateRedirectOrErrorPayload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	params := db.UpdateRedirectParams{
		ID:         input.ID,
		Pattern:    input.Pattern,
		IgnoreCase: input.IgnoreCase,
		IsRegex:    input.IsRegex,
		Target:     input.Target,
	}

	redirect, err := env.UpdateRedirect(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update redirect: %w", err)
	}

	response := model.UpdateRedirectPayload{
		Redirect: &redirect,
	}

	return &response, nil
}