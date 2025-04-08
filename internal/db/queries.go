package db

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

type Note struct {
	Path    string
	Content string
}

var ErrNotePathHashUnresolvedCollision = errors.New("note path hash unresolved collision")
var ErrNoteVersionAlreadyExists = errors.New("note version already exists")

func (q *Queries) InsertNote(ctx context.Context, arg Note) error {
	sha := sha256.New()

	sha.Write([]byte(arg.Path))
	pathHash := base64.URLEncoding.EncodeToString(sha.Sum(nil))

	sha.Reset()
	sha.Write([]byte(arg.Content))
	contentHash := base64.URLEncoding.EncodeToString(sha.Sum(nil))

	var notePath *InsertNotePathRow

	for i := 6; i < len(pathHash); i++ {
		notePathParams := InsertNotePathParams{
			Value:     arg.Path,
			ValueHash: pathHash[:i],

			LatestContentHash: contentHash,
		}

		insertedRow, insertErr := q.InsertNotePath(ctx, notePathParams)
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

	increaseParams := IncrementNoteVersionCountParams{
		ID: notePath.ID,

		LatestContentHash: contentHash,
	}

	version, err := q.IncrementNoteVersionCount(ctx, increaseParams)
	if err != nil {
		return fmt.Errorf("failed to IncrementNoteVersionCount: %w", err)
	}

	noteVersion := InsertNoteVersionParams{
		PathID:  notePath.ID,
		Version: version,
		Content: arg.Content,
	}

	err = q.InsertNoteVersion(ctx, noteVersion)
	if err != nil {
		return fmt.Errorf("failed to InsertNoteVersion: %w", err)
	}

	return nil
}
