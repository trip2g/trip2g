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
	HandleLatestNotesAfterSave(nvs *appmodel.NoteViews) error
	Layouts() *appmodel.Layouts
}

var allowedExtensins = map[string]struct{}{ //nolint:gochecknoglobals // it's a constant
	".md":   {},
	".html": {},
}

var allowedContentTypes = map[string]struct{}{ //nolint:gochecknoglobals // it's a constant
	"text/plain; charset=utf-8": {},
	"text/html; charset=utf-8":  {},
}

func Resolve(ctx context.Context, env Env, input model.PushNotesInput) (model.PushNotesOrErrorPayload, error) {
	// with empty updates, we should return assets anyway
	// if len(input.Updates) == 0 {
	// 	return &model.ErrorPayload{Message: "No updates provided"}, nil
	// }

	log := logger.WithPrefix(env.Logger(), "pushNotes:")

	for _, update := range input.Updates {
		_, allowed := allowedExtensins[strings.ToLower(filepath.Ext(update.Path))]
		if !allowed {
			log.Info("unsupported file extension", "path", update.Path)
			return &model.ErrorPayload{Message: "Only .md and .html files are supported"}, nil
		}

		// Once I accidentally pushed an image as content and the system accepted it.
		// This is just a small safeguard check.
		contentType := http.DetectContentType([]byte(update.Content))
		_, allowed = allowedContentTypes[contentType]

		if !allowed {
			msg := fmt.Sprintf("Unsupported content type: %s", contentType)
			log.Info("unsupported content type", "path", update.Path, "contentType", contentType)
			return &model.ErrorPayload{Message: msg}, nil
		}

		note := appmodel.RawNote{
			Path:    update.Path,
			Content: update.Content,
		}

		log.Info("insert note", "path", update.Path)

		insertErr := env.InsertNote(ctx, note)
		if insertErr != nil {
			return nil, fmt.Errorf("failed to insert note: %w", insertErr)
		}
	}

	nvs, err := env.PrepareLatestNotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare notes: %w", err)
	}

	// TODO: mv to HandleLatestNotesAfterSave
	for _, subgraph := range nvs.Subgraphs {
		insertErr := env.InsertSubgraph(ctx, subgraph.Name)
		if insertErr != nil {
			return nil, fmt.Errorf("failed to insert subgraph: %w", insertErr)
		}
	}

	err = env.HandleLatestNotesAfterSave(nvs)
	if err != nil {
		return nil, fmt.Errorf("failed to handle latest notes after save: %w", err)
	}

	pushedNotes := []model.PushedNote{}

	// prepare md notes
	for _, note := range nvs.List {
		assets := []model.PushedNoteAsset{}

		for relativePath := range note.Assets {
			var hash *string

			replace, ok := note.AssetReplaces[relativePath]
			if ok && replace != nil {
				hash = &replace.Hash
			}

			assets = append(assets, model.PushedNoteAsset{
				Path:       relativePath,
				Sha256Hash: hash,
			})
		}

		pushedNotes = append(pushedNotes, model.PushedNote{
			ID:     note.VersionID,
			Path:   note.Path,
			Assets: assets,
		})
	}

	// prepare layouts
	layouts := env.Layouts()

	for _, layout := range layouts.Map {
		assets := []model.PushedNoteAsset{}

		for _, asset := range layout.Assets {
			var hash *string

			replace, ok := layout.AssetReplaces[asset.Path]
			if ok {
				hash = &replace.Hash
			}

			assets = append(assets, model.PushedNoteAsset{
				Path:       asset.Path,
				Sha256Hash: hash,
			})
		}

		pushedNotes = append(pushedNotes, model.PushedNote{
			ID:     layout.VersionID,
			Path:   layout.Path,
			Assets: assets,
		})
	}

	response := model.PushNotesPayload{
		Notes: pushedNotes,
	}

	return &response, nil
}
