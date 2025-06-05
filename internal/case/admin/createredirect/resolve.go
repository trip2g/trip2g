package createredirect

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	InsertRedirect(ctx context.Context, params db.InsertRedirectParams) (db.Redirect, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

func Resolve(ctx context.Context, env Env, input model.CreateRedirectInput) (model.CreateRedirectOrErrorPayload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	params := db.InsertRedirectParams{
		CreatedBy:  int64(token.ID),
		Pattern:    input.Pattern,
		IgnoreCase: input.IgnoreCase,
		IsRegex:    input.IsRegex,
		Target:     input.Target,
	}

	redirect, err := env.InsertRedirect(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create redirect: %w", err)
	}

	response := model.CreateRedirectPayload{
		Redirect: &redirect,
	}

	return &response, nil
}