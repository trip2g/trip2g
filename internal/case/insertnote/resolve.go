package insertnote

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg insertnote_test . Env

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"trip2g/internal/db"
	"trip2g/internal/model"
)

var ErrNotePathHashUnresolvedCollision = errors.New("note path hash unresolved collision")
var ErrNoteVersionAlreadyExists = errors.New("note version already exists")

type Env interface {
	InsertNotePath(ctx context.Context, arg db.InsertNotePathParams) (db.InsertNotePathRow, error)
	IncrementNoteVersionCount(ctx context.Context, arg db.IncrementNoteVersionCountParams) (int64, error)
	InsertNoteVersion(ctx context.Context, arg db.InsertNoteVersionParams) (int64, error)
	InsertNoteVersionDeliveryAttribution(ctx context.Context, arg db.InsertNoteVersionDeliveryAttributionParams) error
	UnhideNotePath(ctx context.Context, value string) error
	// NoteVersionActor returns who is pushing this version: the acting user id,
	// the authenticating API key id, and the client identifier from the
	// X-trip2g-client request header. Any field may be nil when unknown.
	NoteVersionActor(ctx context.Context) model.NoteActor
}

// Resolve writes the note and reports what the write actually did: which path it
// hit, the version it inserted (0 when the content was already stored), and
// whether it brought a hidden path back. Callers use that to tell a real change
// from a write that changed nothing.
func Resolve(ctx context.Context, env Env, arg model.RawNote) (model.NoteSaveResult, error) {
	sha := sha256.New()

	sha.Write([]byte(arg.Path))
	pathHash := base64.URLEncoding.EncodeToString(sha.Sum(nil))

	sha.Reset()
	sha.Write([]byte(arg.Content))
	contentHash := base64.URLEncoding.EncodeToString(sha.Sum(nil))

	var notePath *db.InsertNotePathRow

	for i := 6; i < len(pathHash); i++ {
		notePathParams := db.InsertNotePathParams{
			Value:     arg.Path,
			ValueHash: pathHash[:i],

			LatestContentHash: contentHash,
		}

		insertedRow, insertErr := env.InsertNotePath(ctx, notePathParams)
		if insertErr != nil {
			// check if the error is a unique constraint violation
			if strings.Contains(insertErr.Error(), "note_paths.value_hash") {
				continue
			}

			return model.NoteSaveResult{}, fmt.Errorf("failed to InsertNotePath: %w", insertErr)
		}

		notePath = &insertedRow

		break
	}

	if notePath == nil {
		return model.NoteSaveResult{}, ErrNotePathHashUnresolvedCollision
	}

	// A push always brings a hidden note back, content change or not: writing is
	// the only way to unhide, so a restored file with unchanged bytes must still
	// reappear. Reported to the caller, since the served snapshot has to be
	// reloaded for it even when no version follows.
	result := model.NoteSaveResult{PathID: notePath.ID, Unhidden: notePath.HiddenBy != nil}
	if result.Unhidden {
		if err := env.UnhideNotePath(ctx, arg.Path); err != nil {
			return model.NoteSaveResult{}, fmt.Errorf("failed to unhide note path: %w", err)
		}
	}

	if notePath.VersionCount > 0 && notePath.LatestContentHash == contentHash {
		// Content hasn't changed, no new version created.
		return result, nil
	}

	increaseParams := db.IncrementNoteVersionCountParams{
		ID: notePath.ID,

		LatestContentHash: contentHash,
	}

	version, err := env.IncrementNoteVersionCount(ctx, increaseParams)
	if err != nil {
		return model.NoteSaveResult{}, fmt.Errorf("failed to IncrementNoteVersionCount: %w", err)
	}

	actor := env.NoteVersionActor(ctx)

	noteVersion := db.InsertNoteVersionParams{
		PathID:            notePath.ID,
		Version:           version,
		Content:           arg.Content,
		CreatedByUserID:   actor.UserID,
		CreatedByApiKeyID: actor.APIKeyID,
		CreatedByClient:   actor.Client,
	}

	versionID, err := env.InsertNoteVersion(ctx, noteVersion)
	if err != nil {
		return model.NoteSaveResult{}, fmt.Errorf("failed to InsertNoteVersion: %w", err)
	}

	if actor.DeliveryKind != nil && actor.DeliveryID != nil {
		err = env.InsertNoteVersionDeliveryAttribution(ctx, db.InsertNoteVersionDeliveryAttributionParams{
			NoteVersionID: versionID,
			DeliveryKind:  *actor.DeliveryKind,
			DeliveryID:    *actor.DeliveryID,
		})
		if err != nil {
			return model.NoteSaveResult{}, fmt.Errorf("failed to InsertNoteVersionDeliveryAttribution: %w", err)
		}
	}

	result.VersionID = versionID

	return result, nil
}
