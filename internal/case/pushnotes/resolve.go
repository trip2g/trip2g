package pushnotes

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"

	appmodel "trip2g/internal/model"
)

type Env interface {
	Logger() logger.Logger
	InsertNote(ctx context.Context, update appmodel.RawNote) error
	InsertSubgraph(ctx context.Context, name string) error
	PrepareLatestNotes(ctx context.Context) (*appmodel.NoteViews, error)
	UnhideNotePath(ctx context.Context, value string) error
}

func Resolve(ctx context.Context, env Env, input model.PushNotesInput) (model.PushNotesOrErrorPayload, error) {
	// with empty updates, we should return assets anyway
	// if len(input.Updates) == 0 {
	// 	return &model.ErrorPayload{Message: "No updates provided"}, nil
	// }

	for _, update := range input.Updates {
		if strings.ToLower(filepath.Ext(update.Path)) != ".md" {
			return &model.ErrorPayload{Message: "Only .md files are supported"}, nil
		}

		// Once I accidentally pushed an image as content and the system accepted it.
		// This is just a small safeguard check.
		contentType := http.DetectContentType([]byte(update.Content))
		if contentType != "text/plain; charset=utf-8" {
			return &model.ErrorPayload{Message: "Only markdown content is supported"}, nil
		}

		note := appmodel.RawNote{
			Path:    update.Path,
			Content: update.Content,
		}

		insertErr := env.InsertNote(ctx, note)
		if insertErr != nil {
			return nil, fmt.Errorf("failed to insert note: %w", insertErr)
		}

		// Reset hidden_by and hidden_at when note is pushed
		unhideErr := env.UnhideNotePath(ctx, update.Path)
		if unhideErr != nil {
			return nil, fmt.Errorf("failed to unhide note path: %w", unhideErr)
		}
	}

	nvs, err := env.PrepareLatestNotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare notes: %w", err)
	}

	// env.Logger().Info("insert subgraphs", "subgraphs", nvs.Subgraphs)

	for _, subgraph := range nvs.Subgraphs {
		insertErr := env.InsertSubgraph(ctx, subgraph.Name)
		if insertErr != nil {
			return nil, fmt.Errorf("failed to insert subgraph: %w", insertErr)
		}
	}

	response := model.PushNotesPayload{
		Notes: nvs.List,
	}

	return &response, nil
}
