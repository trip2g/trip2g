package resetnotfoundpath

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	NotFoundPathByID(ctx context.Context, id int64) (db.NotFoundPath, error)
	ResetNotFoundPathTotalHits(ctx context.Context, id int64) (db.NotFoundPath, error)
}

func Resolve(ctx context.Context, env Env, input model.ResetNotFoundPathInput) (model.ResetNotFoundPathOrErrorPayload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin user token: %w", err)
	}

	// Check if the path exists
	_, err = env.NotFoundPathByID(ctx, input.ID)
	if err != nil {
		if db.IsNoFound(err) {
			return &model.ErrorPayload{Message: "not found path not found"}, nil
		}
		return nil, fmt.Errorf("failed to get not found path %d: %w", input.ID, err)
	}

	// Reset the total hits to 1
	resetPath, err := env.ResetNotFoundPathTotalHits(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to reset not found path total hits: %w", err)
	}

	return &model.ResetNotFoundPathPayload{
		NotFoundPath: &resetPath,
	}, nil
}
