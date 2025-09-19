package sitesearch

import (
	"context"
	"trip2g/internal/graph/model"
)

type Env interface {
}

func Resolve(ctx context.Context, env Env, input model.SearchInput) (*model.SearchConnection, error) {
	return &model.SearchConnection{}, nil
}
