package updatenotes

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

type Env interface {
	LatestNoteViews() *appmodel.NoteViews
	InsertNote(ctx context.Context, note appmodel.RawNote) (int64, error)
	HideNotePath(ctx context.Context, params db.HideNotePathParams) error
	PrepareLatestNotes(ctx context.Context, partial bool) (*appmodel.NoteViews, error)
	HandleLatestNotesAfterSave(ctx context.Context, pathIDs []int64) error
}

func hashContent(content []byte) string {
	h := sha256.New()
	h.Write(content)
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

//nolint:gocognit // per-change branches share state with the outer loop; extraction would require threading paths/pathIDs and a multi-return payload sentinel through helpers.
func Resolve(ctx context.Context, env Env, input model.UpdateNotesInput) (model.UpdateNotesOrErrorPayload, error) {
	nvs := env.LatestNoteViews()
	var paths []string
	var pathIDs []int64
	hid := false

	// nvs.PathMap is keyed by note.Path (filesystem path, e.g. "todo.md").
	// NoteViews.GetByPath uses the Permalink map (URL path), so PathMap is the correct lookup here.
	//
	// Note: if the same path appears more than once in changes, the second
	// operation reads stale content — callers must not patch the same path twice per call.
	for _, change := range input.Changes {
		switch {
		case change.Upsert != nil:
			upsert := change.Upsert
			// ExpectedHash gates the upsert with optimistic concurrency.
			// actualHash defaults to "" for an absent note (the nv != nil block is skipped),
			// so expectedHash == "" is the create-only sentinel: it asserts "expect this note
			// to be absent" — absent → actualHash "" matches → create; an existing note always
			// hashes non-empty → HashMismatch (never overwritten). A non-empty expectedHash is
			// the usual optimistic update (matches only the exact current content).
			if upsert.ExpectedHash != nil {
				nv := nvs.PathMap[upsert.Path]
				var actualHash string
				if nv != nil {
					actualHash = hashContent(nv.Content)
				}
				if actualHash != *upsert.ExpectedHash {
					return model.UpdateNotesHashMismatchPayload{
						Path:       upsert.Path,
						ActualHash: actualHash,
					}, nil
				}
			}
			pathID, err := env.InsertNote(ctx, appmodel.RawNote{Path: upsert.Path, Content: upsert.Content})
			if err != nil {
				return nil, fmt.Errorf("updatenotes: insert upsert %s: %w", upsert.Path, err)
			}
			pathIDs = append(pathIDs, pathID)
			paths = append(paths, upsert.Path)
		case change.Patch != nil:
			patch := change.Patch
			nv := nvs.PathMap[patch.Path]
			if nv == nil {
				return &model.ErrorPayload{Message: fmt.Sprintf("note not found: %s", patch.Path)}, nil
			}
			if patch.ExpectedHash != nil {
				actualHash := hashContent(nv.Content)
				if actualHash != *patch.ExpectedHash {
					return model.UpdateNotesHashMismatchPayload{
						Path:       patch.Path,
						ActualHash: actualHash,
					}, nil
				}
			}
			content := string(nv.Content)
			idx := strings.Index(content, patch.Find)
			if idx == -1 {
				return model.UpdateNotesPatchNotFoundPayload{Path: patch.Path, Find: patch.Find}, nil
			}
			// Check for multiple occurrences
			if strings.Contains(content[idx+len(patch.Find):], patch.Find) {
				return model.UpdateNotesPatchNotFoundPayload{Path: patch.Path, Find: patch.Find}, nil
			}
			newContent := content[:idx] + patch.Replace + content[idx+len(patch.Find):]
			pathID, err := env.InsertNote(ctx, appmodel.RawNote{Path: patch.Path, Content: newContent})
			if err != nil {
				return nil, fmt.Errorf("updatenotes: insert patch %s: %w", patch.Path, err)
			}
			pathIDs = append(pathIDs, pathID)
			paths = append(paths, patch.Path)
		case change.Hide != nil:
			// Hide is a metadata operation. Unlike the standalone hideNotes mutation,
			// this does not trigger webhooks — extend Env if webhook support is needed.
			hide := change.Hide
			err := env.HideNotePath(ctx, db.HideNotePathParams{
				HiddenBy: &input.ApiKey.CreatedBy,
				Value:    hide.Path,
			})
			if err != nil {
				return nil, fmt.Errorf("updatenotes: hide %s: %w", hide.Path, err)
			}
			paths = append(paths, hide.Path)
			hid = true
		default:
			continue
		}
	}

	// Content changes reload NoteViews and run after-save handling.
	// Hide-only batches still reload NoteViews so the hidden paths stop
	// resolving on the public site (rendernotepage reads the in-memory cache),
	// but skip HandleLatestNotesAfterSave since no content was saved.
	if len(pathIDs) > 0 {
		if _, err := env.PrepareLatestNotes(ctx, false); err != nil {
			return nil, fmt.Errorf("updatenotes: prepare latest notes: %w", err)
		}
		if err := env.HandleLatestNotesAfterSave(ctx, pathIDs); err != nil {
			return nil, fmt.Errorf("updatenotes: handle latest notes after save: %w", err)
		}
	} else if hid {
		if _, err := env.PrepareLatestNotes(ctx, false); err != nil {
			return nil, fmt.Errorf("updatenotes: prepare latest notes after hide: %w", err)
		}
	}

	return model.UpdateNotesSuccessPayload{Paths: paths}, nil
}
