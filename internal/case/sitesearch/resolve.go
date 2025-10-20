package sitesearch

import (
	"context"

	_ "github.com/blevesearch/bleve/v2/analysis/lang/ru"

	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
)

type Env interface {
	SearchLatestNotes(query string) (*model.SearchConnection, error)
	SearchLiveNotes(query string) (*model.SearchConnection, error)
	Logger() logger.Logger
}

type noteContent struct {
	Title string
	Body  string
}

func Resolve(ctx context.Context, env Env, input model.SearchInput) (*model.SearchConnection, error) {
	conn, err := env.SearchLatestNotes(input.Query)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
