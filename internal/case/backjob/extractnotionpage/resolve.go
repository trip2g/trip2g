package extractnotionpage

import (
	"context"
	"trip2g/internal/logger"
)

type Params struct {
	PageID string
}

type Env interface {
	Logger() logger.Logger
}

func Resolve(ctx context.Context, env Env, task Params) error {
	env.Logger().Info("Extracting notion page", "pageID", task.PageID)

	// TODO: Implement notion page extraction logic

	return nil
}