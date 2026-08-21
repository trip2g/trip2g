package pushnotes

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"trip2g/internal/appreq"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"

	appmodel "trip2g/internal/model"
)

type Env interface {
	Logger() logger.Logger
	InsertNote(ctx context.Context, update appmodel.RawNote) (appmodel.NoteSaveResult, error)
	InsertUncommittedPath(ctx context.Context, notePathID int64) error
	PrepareLatestNotes(ctx context.Context, partial bool) (*appmodel.NoteViews, error)
	HandleLatestNotesAfterSave(ctx context.Context, changedPathIDs []int64) error
	Layouts() *appmodel.Layouts
	LatestNoteViews() *appmodel.NoteViews
	CheckStorageLimits(ctx context.Context, additionalAssetBytes int64) (string, error)
	PublicURL() string
}

var allowedExtensins = map[string]struct{}{ //nolint:gochecknoglobals // it's a constant
	".md":         {},
	".html":       {},
	".canvas":     {},
	".base":       {},
	".excalidraw": {},
}

var allowedContentTypes = map[string]struct{}{ //nolint:gochecknoglobals // it's a constant
	"text/plain; charset=utf-8": {},
	"text/html; charset=utf-8":  {},
}

func Resolve(ctx context.Context, env Env, input model.PushNotesInput) (model.PushNotesOrErrorPayload, error) {
	// pushNotes is the full-vault sync lane with no per-path scoping — scoped
	// tokens are denied outright.
	if err := appreq.RequireUnscoped(ctx); err != nil {
		return &model.ErrorPayload{Message: err.Error()}, nil //nolint:nilerr // authorization denial returned as payload, not as error.
	}

	log := logger.WithPrefix(env.Logger(), "pushNotes:")

	// Check storage limits before writing.
	limitMsg, checkErr := env.CheckStorageLimits(ctx, 0)
	if checkErr != nil {
		return nil, checkErr
	}

	if limitMsg != "" {
		return &model.ErrorPayload{Message: limitMsg}, nil
	}

	// If no updates, just return current state without rebuilding
	if len(input.Updates) == 0 {
		nvs := env.LatestNoteViews()
		pushedNotes := buildPushedNotes(nvs, env.Layouts(), env.PublicURL())
		return &model.PushNotesPayload{Notes: pushedNotes, Updated: []model.PushedNote{}}, nil
	}

	skipCommit := input.SkipCommit != nil && *input.SkipCommit
	pathIDs := []int64{}
	// changedPathIDs is pathIDs minus the writes that changed nothing anyone can
	// see. A resync re-pushing identical content to a visible path is not a
	// change; a restore is, even though it stores no new version — the note
	// reappears on the site.
	changedPathIDs := []int64{}

	// Validate the whole batch before writing anything: callers (e.g. gitapi's
	// applyDiff) roll back only their own state on failure, so a partially
	// inserted prefix would diverge the DB from Git.
	for _, update := range input.Updates {
		if errPayload := validateUpdate(log, update); errPayload != nil {
			return errPayload, nil
		}
	}

	for _, update := range input.Updates {
		note := appmodel.RawNote{
			Path:    update.Path,
			Content: update.Content,
		}

		log.Info("insert note", "path", update.Path)

		saved, err := env.InsertNote(ctx, note)
		if err != nil {
			return nil, fmt.Errorf("failed to insert note: %w", err)
		}

		// Every pushed path is reported back; only the ones that actually changed
		// carry side effects. A vault resync re-pushes thousands of identical
		// notes, and each of those used to fire webhooks, embeddings and SSE.
		pathIDs = append(pathIDs, saved.PathID)
		if saved.AffectsSnapshot() {
			changedPathIDs = append(changedPathIDs, saved.PathID)
		}
	}

	// Nothing reached the served snapshot: report against the views already in
	// memory instead of paying for a reload.
	nvs := env.LatestNoteViews()
	if len(changedPathIDs) > 0 {
		var prepErr error
		nvs, prepErr = env.PrepareLatestNotes(ctx, input.Partial)
		if prepErr != nil {
			return nil, fmt.Errorf("failed to prepare notes: %w", prepErr)
		}
	}

	// If skipCommit, save path IDs to uncommitted table and skip HandleLatestNotesAfterSave.
	// The uncommitted table is what commitNotes later reports and acts on, so a
	// path missing here is invisible to the client and never gets its side
	// effects. An unchanged push has nothing to defer.
	if skipCommit {
		for _, pathID := range changedPathIDs {
			insertErr := env.InsertUncommittedPath(ctx, pathID)
			if insertErr != nil {
				return nil, fmt.Errorf("failed to insert uncommitted path: %w", insertErr)
			}
		}
	} else if len(changedPathIDs) > 0 {
		err := env.HandleLatestNotesAfterSave(ctx, changedPathIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to handle latest notes after save: %w", err)
		}
	}

	pushedNotes := buildPushedNotes(nvs, env.Layouts(), env.PublicURL())
	updatedNotes := buildUpdatedNotes(nvs, pathIDs, env.PublicURL())

	return &model.PushNotesPayload{Notes: pushedNotes, Updated: updatedNotes}, nil
}

func validateUpdate(log logger.Logger, update model.PushNoteInput) *model.ErrorPayload {
	// Check for .html.json first (JSON layout format) since filepath.Ext returns .json
	allowed := strings.HasSuffix(strings.ToLower(update.Path), ".html.json")
	if !allowed {
		_, allowed = allowedExtensins[strings.ToLower(filepath.Ext(update.Path))]
	}
	if !allowed {
		log.Info("unsupported file extension", "path", update.Path)
		return &model.ErrorPayload{Message: "Only .md, .html, .html.json, .canvas, .base, and .excalidraw files are supported"}
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

func buildNoteAssets(note *appmodel.NoteView, publicURL string) []model.PushedNoteAsset {
	assets := []model.PushedNoteAsset{}

	for relativePath := range note.Assets {
		var hash *string
		var assetID int64
		var absolutePath string
		var url string

		replace, ok := note.AssetReplaces[relativePath]
		if ok && replace != nil {
			hash = &replace.Hash
			assetID = replace.ID
			absolutePath = replace.AbsolutePath
			url = appmodel.AbsoluteURL(publicURL, replace.URL)
		}

		assets = append(assets, model.PushedNoteAsset{
			ID:           relativePath,
			AssetID:      assetID,
			Path:         relativePath,
			Sha256Hash:   hash,
			AbsolutePath: absolutePath,
			URL:          url,
		})
	}

	return assets
}

func buildLayoutAssets(layout appmodel.Layout, publicURL string) []model.PushedNoteAsset {
	assets := []model.PushedNoteAsset{}

	for _, asset := range layout.Assets {
		var hash *string
		var assetID int64
		var absolutePath string
		var url string

		replace, ok := layout.AssetReplaces[asset.Path]
		if ok && replace != nil {
			hash = &replace.Hash
			assetID = replace.ID
			absolutePath = replace.AbsolutePath
			url = appmodel.AbsoluteURL(publicURL, replace.URL)
		}

		assets = append(assets, model.PushedNoteAsset{
			ID:           asset.Path,
			AssetID:      assetID,
			Path:         asset.Path,
			Sha256Hash:   hash,
			AbsolutePath: absolutePath,
			URL:          url,
		})
	}

	return assets
}

func buildPushedNotes(nvs *appmodel.NoteViews, layouts *appmodel.Layouts, publicURL string) []model.PushedNote {
	warnings := nvs.Warnings()
	pushedNotes := []model.PushedNote{}

	for _, note := range nvs.List {
		assets := buildNoteAssets(note, publicURL)
		urlVal := nvs.ResolveFullURL(note, publicURL)
		noteWarnings := warnings[note.Permalink]
		if noteWarnings == nil {
			noteWarnings = []appmodel.NoteWarning{}
		}
		pushedNotes = append(pushedNotes, model.PushedNote{
			ID:       note.VersionID,
			Path:     note.Path,
			Assets:   assets,
			URL:      &urlVal,
			Warnings: noteWarnings,
		})
	}

	for _, layout := range layouts.Map {
		assets := buildLayoutAssets(layout, publicURL)
		layoutWarnings := layout.Warnings
		if layoutWarnings == nil {
			layoutWarnings = []appmodel.NoteWarning{}
		}
		pushedNotes = append(pushedNotes, model.PushedNote{
			ID:       layout.VersionID,
			Path:     layout.Path,
			Assets:   assets,
			URL:      nil,
			Warnings: layoutWarnings,
		})
	}

	return pushedNotes
}

// buildUpdatedNotes returns PushedNote entries for only the notes with the given pathIDs.
// Notes not found in nvs (e.g. deleted) are silently skipped. Layouts are always skipped
// here since they never register into nvs (see noteloader.Load) - their warnings only
// surface through the full notes list built by buildPushedNotes.
func buildUpdatedNotes(nvs *appmodel.NoteViews, pathIDs []int64, publicURL string) []model.PushedNote {
	result := make([]model.PushedNote, 0, len(pathIDs))
	for _, id := range pathIDs {
		note := nvs.GetByPathID(id)
		if note == nil {
			continue
		}
		urlVal := nvs.ResolveFullURL(note, publicURL)
		noteWarnings := note.Warnings
		if noteWarnings == nil {
			noteWarnings = []appmodel.NoteWarning{}
		}
		result = append(result, model.PushedNote{
			ID:       note.VersionID,
			Path:     note.Path,
			Assets:   buildNoteAssets(note, publicURL),
			URL:      &urlVal,
			Warnings: noteWarnings,
		})
	}
	return result
}
