package db

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"errors"
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
	pathHash := fmt.Sprintf("%x", sha.Sum(nil))

	sha.Reset()
	sha.Write([]byte(arg.Content))
	contentHash := fmt.Sprintf("%x", sha.Sum(nil))

	for i := 6; i < len(pathHash); i++ {
		notePathParams := InsertNotePathParams{
			Path:     arg.Path,
			PathHash: pathHash[:i],
		}

		insertErr := q.InsertNotePath(ctx, notePathParams)
		if insertErr != nil {
			// check if the error is a unique constraint violation
			if strings.Contains(insertErr.Error(), "note_paths.path_hash") {
				if i == len(pathHash)-1 {
					return ErrNotePathHashUnresolvedCollision
				}

				continue
			}

			return fmt.Errorf("failed to InsertNotePath: %w", insertErr)
		}

		break
	}

	latestContentHash := sql.NullString{
		String: contentHash,
		Valid:  false,
	}

	increaseParams := IncrementNoteVersionCountParams{
		Path:                arg.Path,
		LatestContentHash:   latestContentHash,
		LatestContentHash_2: latestContentHash,
	}

	pathRow, err := q.IncrementNoteVersionCount(ctx, increaseParams)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoteVersionAlreadyExists
		}

		return fmt.Errorf("failed to IncrementNoteVersionCount: %w", err)
	}

	noteVersion := InsertNoteVersionParams{
		PathID:  pathRow.ID,
		Version: pathRow.VersionCount,
		Content: arg.Content,
	}

	err = q.InsertNoteVersion(ctx, noteVersion)
	if err != nil {
		return fmt.Errorf("failed to InsertNoteVersion: %w", err)
	}

	return nil
}
