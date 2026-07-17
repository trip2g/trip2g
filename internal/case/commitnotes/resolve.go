package commitnotes

import (
	"context"
	"fmt"
	"trip2g/internal/appreq"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

type Env interface {
	PrepareLatestNotes(ctx context.Context, partial bool) (*appmodel.NoteViews, error)
	HandleLatestNotesAfterSave(ctx context.Context, changedPathIDs []int64) error
	ListUncommittedPaths(ctx context.Context) ([]int64, error)
	ClearUncommittedPaths(ctx context.Context) error
	PublicURL() string
}

type Payload = model.CommitNotesOrErrorPayload

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
			Assets:   []model.PushedNoteAsset{},
			URL:      &urlVal,
			Warnings: noteWarnings,
		})
	}
	return result
}

func Resolve(ctx context.Context, env Env) (Payload, error) {
	// commitNotes has no path input, so per-path scoping is undefined — scoped
	// tokens are denied outright.
	if err := appreq.RequireUnscoped(ctx); err != nil {
		return &model.ErrorPayload{Message: err.Error()}, nil //nolint:nilerr // authorization denial returned as payload, not as error.
	}

	pathIDs, err := env.ListUncommittedPaths(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list uncommitted paths: %w", err)
	}

	nvs, err := env.PrepareLatestNotes(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare notes: %w", err)
	}

	err = env.HandleLatestNotesAfterSave(ctx, pathIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to handle latest notes after save: %w", err)
	}

	err = env.ClearUncommittedPaths(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to clear uncommitted paths: %w", err)
	}

	updated := buildUpdatedNotes(nvs, pathIDs, env.PublicURL())
	return &model.CommitNotesPayload{Success: true, Updated: updated}, nil
}
