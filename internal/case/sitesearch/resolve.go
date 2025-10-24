package sitesearch

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/usertoken"

	appmodel "trip2g/internal/model"
)

type Env interface {
	SearchLatestNotes(query string) ([]appmodel.SearchResult, error)
	SearchLiveNotes(query string) ([]appmodel.SearchResult, error)
	CurrentUserToken(ctx context.Context) (*usertoken.Data, error)
	CanReadNote(ctx context.Context, note *appmodel.NoteView) (bool, error)
	LatestConfig() db.ConfigVersion
	Logger() logger.Logger
}

func Resolve(ctx context.Context, env Env, input model.SearchInput) (*model.SearchConnection, error) {
	userToken, err := env.CurrentUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	config := env.LatestConfig()

	var results []appmodel.SearchResult

	// choose the right source
	if config.ShowDraftVersions || userToken.IsAdmin() {
		results, err = env.SearchLatestNotes(input.Query)
		if err != nil {
			return nil, fmt.Errorf("failed to SearchLatestNotes: %w", err)
		}
	} else {
		results, err = env.SearchLiveNotes(input.Query)
		if err != nil {
			return nil, fmt.Errorf("failed to SearchLiveNotes: %w", err)
		}
	}

	// filter results based on permissions
	conn := model.SearchConnection{}
	hiddenResults := []appmodel.SearchResult{}

	for _, res := range results {
		if res.NoteView != nil {
			canRead, readErr := env.CanReadNote(ctx, res.NoteView)
			if readErr != nil {
				return nil, fmt.Errorf("failed to check CanReadNote: %w", readErr)
			}

			if canRead {
				conn.Nodes = append(conn.Nodes, res)
				continue
			}

			croppedResult := appmodel.SearchResult{
				HighlightedTitle:   res.HighlightedTitle,
				URL:                res.URL,
				HighlightedContent: []string{"Закрытый материал."},
			}

			hiddenResults = append(hiddenResults, croppedResult)
		}
	}

	// push hidden results to the end of the list
	conn.Nodes = append(conn.Nodes, hiddenResults...)

	return &conn, nil
}
