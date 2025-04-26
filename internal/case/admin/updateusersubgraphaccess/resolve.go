package updateusersubgraphaccess

import (
	"context"
	"time"
	"trip2g/internal/graph/model"
)

type Env interface{}

type Request struct {
	ID         int64
	SubgraphID int64
	ExpiresAt  time.Time
}

func (req Request) Resolve(ctx context.Context, env Env) (model.UpdateUserSubgraphAccessOrErrorPayload, error) {
	response := model.UpdateUserSubgraphAccessPayload{}

	return &response, nil
}
