package db

import (
	"context"
	"crypto/sha1"
	"fmt"
	"strings"
)

type Note struct {
	Path    string
	Content string
}

var ErrNotePathHashUnresolvedCollision = fmt.Errorf("note path hash unresolved collision")

func (q *Queries) InsertNote(ctx context.Context, arg Note) error {
	sha := sha1.New()

	sha.Write([]byte(arg.Path))
	pathHash := fmt.Sprintf("%x", sha.Sum(nil))

	sha.Reset()
	sha.Write([]byte(arg.Content))
	contentHash := fmt.Sprintf("%x", sha.Sum(nil))

	for tryCount := 6; ; tryCount++ {
		notePath := InsertNotePathParams{
			Path:     arg.Path,
			PathHash: pathHash[:tryCount],
		}

		insertErr := q.InsertNotePath(ctx, notePath)
		if insertErr != nil {
			if strings.Contains(insertErr.Error(), "note_paths.path_hash") {
				if tryCount >= len(pathHash)-1 {
					return ErrNotePathHashUnresolvedCollision
				}

				continue
			}

			return fmt.Errorf("failed to InsertNotePath: %w", insertErr)
		}

		break
	}

	pathRow, err := q.IncrementNoteVersionCount(ctx, arg.Path)
	if err != nil {
		return fmt.Errorf("failed to GetNotePathID: %w", err)
	}

	noteVersion := InsertNoteVersionParams{
		PathID:  pathRow.ID,
		Version: pathRow.VersionCount,
		Content: arg.Content,

		ContentHash: contentHash,
	}

	err = q.InsertNoteVersion(ctx, noteVersion)
	if err != nil {
		return fmt.Errorf("failed to InsertNoteVersion: %w", err)
	}

	return nil
}
