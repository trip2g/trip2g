package db

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
)

type Note struct {
	Path    string
	Content string
}

var ErrNotePathHashUnresolvedCollision = fmt.Errorf("note path hash unresolved collision")
var ErrNoteVersionAlreadyExists = fmt.Errorf("note version already exists")

func (q *Queries) InsertNote(ctx context.Context, arg Note) error {
	sha := sha1.New()

	sha.Write([]byte(arg.Path))
	pathHash := base64.URLEncoding.EncodeToString(sha.Sum(nil))

	sha.Reset()
	sha.Write([]byte(arg.Content))
	contentHash := base64.URLEncoding.EncodeToString(sha.Sum(nil))

	var notePath *InsertNotePathRow

	for i := 6; i < len(pathHash); i++ {
		notePathParams := InsertNotePathParams{
			Path:     arg.Path,
			PathHash: pathHash[:i],
		}

		insertedRow, insertErr := q.InsertNotePath(ctx, notePathParams)
		if insertErr != nil {
			// check if the error is a unique constraint violation
			if strings.Contains(insertErr.Error(), "note_paths.path_hash") {
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

	if notePath.LatestContentHash.Valid && notePath.LatestContentHash.String == contentHash {
		return ErrNoteVersionAlreadyExists
	}

	increaseParams := IncrementNoteVersionCountParams{
		ID: notePath.ID,
		LatestContentHash: sql.NullString{
			String: contentHash,
			Valid:  true,
		},
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
