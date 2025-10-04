package extractallnotionpages

import (
	"context"
	"fmt"
	"trip2g/internal/logger"
	"trip2g/internal/notiontypes"
)

type Params struct{}

type Result struct {
	PageCount int
}

type Env interface {
	Logger() logger.Logger
	NotionClient() notiontypes.Client
	QueueExtractNotionPage(ctx context.Context, pageID string) error
}

func Resolve(ctx context.Context, env Env, task Params) (*Result, error) {
	logger := env.Logger()
	client := env.NotionClient()

	logger.Info("Extracting all notion pages")

	pages, err := client.AllPages()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all notion pages: %w", err)
	}

	logger.Info("Enqueued all notion pages", "count", len(pages))

	for _, page := range pages {
		err := env.QueueExtractNotionPage(ctx, page.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to enqueue notion page %s: %w", page.ID, err)
		}
	}

	return &Result{PageCount: len(pages)}, nil
}
