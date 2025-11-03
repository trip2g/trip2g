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
	HandleLatestNotesAfterSave(ctx context.Context, changedPathIDs []int64) error
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
	log := logger.WithPrefix(env.Logger(), "pushNotes:")
	changedPaths := map[string]struct{}{}

	for _, update := range input.Updates {
		if errPayload := validateUpdate(log, update); errPayload != nil {
			return errPayload, nil
		}

		note := appmodel.RawNote{
			Path:    update.Path,
			Content: update.Content,
		}

		log.Info("insert note", "path", update.Path)

		err := env.InsertNote(ctx, note)
		if err != nil {
			return nil, fmt.Errorf("failed to insert note: %w", err)
		}

		changedPaths[update.Path] = struct{}{}
	}

	nvs, err := env.PrepareLatestNotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare notes: %w", err)
	}

	pathIDs := []int64{}

	for _, note := range nvs.List {
		_, changed := changedPaths[note.Path]
		if changed {
			pathIDs = append(pathIDs, note.PathID)
		}
	}

	// TODO: mv to HandleLatestNotesAfterSave
	for _, subgraph := range nvs.Subgraphs {
		insertErr := env.InsertSubgraph(ctx, subgraph.Name)
		if insertErr != nil {
			return nil, fmt.Errorf("failed to insert subgraph: %w", insertErr)
		}
	}

	err = env.HandleLatestNotesAfterSave(ctx, pathIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to handle latest notes after save: %w", err)
	}

	pushedNotes := buildPushedNotes(nvs, env.Layouts())

	return &model.PushNotesPayload{Notes: pushedNotes}, nil
}

func validateUpdate(log logger.Logger, update model.PushNoteInput) *model.ErrorPayload {
	_, allowed := allowedExtensins[strings.ToLower(filepath.Ext(update.Path))]
	if !allowed {
		log.Info("unsupported file extension", "path", update.Path)
		return &model.ErrorPayload{Message: "Only .md and .html files are supported"}
	}

	contentType := http.DetectContentType([]byte(update.Content))
	_, allowed = allowedContentTypes[contentType]
	if !allowed {
		msg := fmt.Sprintf("Unsupported content type: %s", contentType)
		log.Info("unsupported content type", "path", update.Path, "contentType", contentType)
		return &model.ErrorPayload{Message: msg}
	}

	return nil
}

func buildNoteAssets(note *appmodel.NoteView) []model.PushedNoteAsset {
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

	return assets
}

func buildLayoutAssets(layout appmodel.Layout) []model.PushedNoteAsset {
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

	return assets
}

func buildPushedNotes(nvs *appmodel.NoteViews, layouts *appmodel.Layouts) []model.PushedNote {
	pushedNotes := []model.PushedNote{}

	for _, note := range nvs.List {
		assets := buildNoteAssets(note)

		pushedNotes = append(pushedNotes, model.PushedNote{
			ID:     note.VersionID,
			Path:   note.Path,
			Assets: assets,
		})
	}

	for _, layout := range layouts.Map {
		assets := buildLayoutAssets(layout)

		pushedNotes = append(pushedNotes, model.PushedNote{
			ID:     layout.VersionID,
			Path:   layout.Path,
			Assets: assets,
		})
	}

	return pushedNotes
}
