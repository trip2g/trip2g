package extractnotionpages

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/notiontypes"
)

type Params struct {
	IntegrationID *int64

	PageID *string
}

type Result struct {
	ExtractedCount int
}

type Env interface {
	Logger() logger.Logger
	NotionClientByIntegrationID(integrationID int64) notiontypes.Client
	AllNotionIntegrations(ctx context.Context) ([]db.NotionIntegration, error)
	InsertNote(ctx context.Context, update model.RawNote) (int64, error)
}

func Resolve(ctx context.Context, env Env, task Params) error {
	log := env.Logger()
	if task.PageID != nil && task.IntegrationID == nil {
		return fmt.Errorf("if PageID(%s) is specified, IntegrationID must also be specified", *task.PageID)
	}

	integrations, err := env.AllNotionIntegrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all notion integrations: %w", err)
	}

	for _, integration := range integrations {
		if task.IntegrationID != nil && *task.IntegrationID != integration.ID {
			continue
		}

		client := env.NotionClientByIntegrationID(integration.ID)
		pageIDS := []string{}

		if task.PageID != nil {
			pageIDS = append(pageIDS, *task.PageID)
		} else {
			pages, pErr := client.AllPages()
			if pErr != nil {
				return fmt.Errorf("failed to get all pages for integration %d: %w", integration.ID, pErr)
			}

			for _, page := range pages {
				pageIDS = append(pageIDS, page.ID)
			}
		}

		for _, pageID := range pageIDS {
			page, pErr := client.GetPage(pageID)
			if pErr != nil {
				return fmt.Errorf("failed to get page %s: %w", pageID, pErr)
			}

			pageContent, pErr := client.GetPageContent(pageID)
			if pErr != nil {
				return fmt.Errorf("failed to get page content for page %s: %w", pageID, pErr)
			}

			rawNote, pErr := notiontypes.ExtractRawNote(page, pageContent, integration.BasePath)
			if pErr != nil {
				return fmt.Errorf("failed to extract raw note for page %s: %w", pageID, pErr)
			}

			_, insertErr := env.InsertNote(ctx, *rawNote)
			if insertErr != nil {
				log.Warn("failed to insert note", "error", insertErr)
				// return fmt.Errorf("failed to insert note for page %s: %w", pageID, insertErr)
			}
		}
	}

	return nil
}
