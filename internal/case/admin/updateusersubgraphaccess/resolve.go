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
	ID         int64
	ExpiresAt  *time.Time
	SubgraphID int64
}

type Input = Request
type Payload = model.UpdateUserSubgraphAccessOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	return input.Resolve(ctx, env)
}

func (input *Request) Resolve(ctx context.Context, env Env) (Payload, error) {
	params := db.UpdateUserSubgraphAccessParams{
		ID: input.ID,

		ExpiresAt:  db.ToNullableTime(input.ExpiresAt),
		SubgraphID: input.SubgraphID,
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
