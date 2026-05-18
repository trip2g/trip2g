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

func Resolve(ctx context.Context, env Env, input model.UpdateNotesInput) (model.UpdateNotesOrErrorPayload, error) {
	nvs := env.LatestNoteViews()
	var paths []string
	var pathIDs []int64

	for _, change := range input.Changes {
		if change.Upsert == nil && change.Patch == nil && change.Hide == nil {
			continue
		}

		if change.Upsert != nil {
			upsert := change.Upsert
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
				return nil, err
			}
			pathIDs = append(pathIDs, pathID)
			paths = append(paths, upsert.Path)

		} else if change.Patch != nil {
			patch := change.Patch
			nv := nvs.PathMap[patch.Path]
			if nv == nil {
				return model.ErrorPayload{Message: fmt.Sprintf("note not found: %s", patch.Path)}, nil
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
			if strings.Index(content[idx+len(patch.Find):], patch.Find) != -1 {
				return model.UpdateNotesPatchNotFoundPayload{Path: patch.Path, Find: patch.Find}, nil
			}
			newContent := content[:idx] + patch.Replace + content[idx+len(patch.Find):]
			pathID, err := env.InsertNote(ctx, appmodel.RawNote{Path: patch.Path, Content: newContent})
			if err != nil {
				return nil, err
			}
			pathIDs = append(pathIDs, pathID)
			paths = append(paths, patch.Path)

		} else if change.Hide != nil {
			hide := change.Hide
			err := env.HideNotePath(ctx, db.HideNotePathParams{
				HiddenBy: &input.ApiKey.CreatedBy,
				Value:    hide.Path,
			})
			if err != nil {
				return nil, err
			}
			paths = append(paths, hide.Path)
		}
	}

	if len(pathIDs) > 0 {
		if _, err := env.PrepareLatestNotes(ctx, false); err != nil {
			return nil, err
		}
		if err := env.HandleLatestNotesAfterSave(ctx, pathIDs); err != nil {
			return nil, err
		}
	}

	return model.UpdateNotesSuccessPayload{Paths: paths}, nil
}
