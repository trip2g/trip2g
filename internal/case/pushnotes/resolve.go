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
}

var allowedExtensins = map[string]string{
	".md":   "text/plain; charset=utf-8",
	".html": "text/html; charset=utf-8",
}

func Resolve(ctx context.Context, env Env, input model.PushNotesInput) (model.PushNotesOrErrorPayload, error) {
	// with empty updates, we should return assets anyway
	// if len(input.Updates) == 0 {
	// 	return &model.ErrorPayload{Message: "No updates provided"}, nil
	// }

	for _, update := range input.Updates {
		expectedContentType, allowed := allowedExtensins[strings.ToLower(filepath.Ext(update.Path))]
		if !allowed {
			return &model.ErrorPayload{Message: "Only .md and .html files are supported"}, nil
		}

		// Once I accidentally pushed an image as content and the system accepted it.
		// This is just a small safeguard check.
		contentType := http.DetectContentType([]byte(update.Content))
		if contentType != expectedContentType {
			msg := fmt.Sprintf("%s: Expected content type: %s, actual: %s", update.Path, expectedContentType, contentType)
			return &model.ErrorPayload{Message: msg}, nil
		}

		note := appmodel.RawNote{
			Path:    update.Path,
			Content: update.Content,
		}

		insertErr := env.InsertNote(ctx, note)
		if insertErr != nil {
			return nil, fmt.Errorf("failed to insert note: %w", insertErr)
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
