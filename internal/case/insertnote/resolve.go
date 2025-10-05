package insertnote

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
	InsertNoteVersion(ctx context.Context, arg db.InsertNoteVersionParams) error
	UnhideNotePath(ctx context.Context, value string) error
}

func Resolve(ctx context.Context, env Env, arg model.RawNote) error {
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

			return fmt.Errorf("failed to InsertNotePath: %w", insertErr)
		}

		notePath = &insertedRow

		break
	}

	if notePath == nil {
		return ErrNotePathHashUnresolvedCollision
	}

	if notePath.VersionCount > 0 && notePath.LatestContentHash == contentHash {
		return ErrNoteVersionAlreadyExists
	}

	increaseParams := db.IncrementNoteVersionCountParams{
		ID: notePath.ID,

		LatestContentHash: contentHash,
	}

	version, err := env.IncrementNoteVersionCount(ctx, increaseParams)
	if err != nil {
		return fmt.Errorf("failed to IncrementNoteVersionCount: %w", err)
	}

	noteVersion := db.InsertNoteVersionParams{
		PathID:  notePath.ID,
		Version: version,
		Content: arg.Content,
	}

	err = env.InsertNoteVersion(ctx, noteVersion)
	if err != nil {
		return fmt.Errorf("failed to InsertNoteVersion: %w", err)
	}

	// Reset hidden_by and hidden_at when note is pushed
	err = env.UnhideNotePath(ctx, arg.Path)
	if err != nil {
		return fmt.Errorf("failed to unhide note path: %w", err)
	}

	return nil
}
