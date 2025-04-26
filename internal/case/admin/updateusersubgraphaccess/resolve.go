package updateusersubgraphaccess

import (
	"context"
	"fmt"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

type Env interface {
	UpdateUserSubgraphAccess(ctx context.Context, arg db.UpdateUserSubgraphAccessParams) (db.UserSubgraphAccess, error)
}

type Request struct {
	ID        int64
	ExpiresAt time.Time
}

func (req *Request) Resolve(ctx context.Context, env Env) (model.UpdateUserSubgraphAccessOrErrorPayload, error) {
	params := db.UpdateUserSubgraphAccessParams{
		ID: req.ID,

		ExpiresAt: db.ToNullableTime(&req.ExpiresAt),
	}

	access, err := env.UpdateUserSubgraphAccess(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update user subgraph access: %w", err)
	}

	response := model.UpdateUserSubgraphAccessPayload{
		UserSubgraphAccess: &access,
	}

	return &response, nil
}
