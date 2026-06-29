package graph

import (
	"context"

	"trip2g/internal/db"
	"trip2g/internal/model"
)

// resolveNotePathContent returns the content to sync back for a note path.
//
// Priority:
//  1. a registered layout (Layouts().Map) → its OriginalContent
//  2. a cached note view (LatestNoteViews().PathMap) → its content
//  3. the latest stored note_versions row (fetchLatest)
//
// Step 3 is the data-loss guard. A path is LISTED by notePaths with a real,
// non-empty latest_content_hash but can be ABSENT from both caches when, for
// example, it is a layout-shaped note under a subfolder ("demo/_layouts/…",
// which noteloader only registers under the top-level "_layouts/"), or during a
// transient note_paths.version_count vs note_versions.version divergence.
// Layouts are also hidden from LatestNoteViews. Returning "" in that case makes
// the sync plugin overwrite the local file with empty content, wiping it.
func resolveNotePathContent(
	obj *db.NotePath,
	layouts *model.Layouts,
	nvs *model.NoteViews,
	fetchLatest func() (string, bool, error),
) (string, error) {
	for _, layout := range layouts.Map {
		if layout.Path == obj.Value {
			// Return original content for sync (JSON for .html.json, HTML for .html)
			return layout.OriginalContent, nil
		}
	}

	if nv := nvs.PathMap[obj.Value]; nv != nil {
		return string(nv.Content), nil
	}

	content, found, err := fetchLatest()
	if err != nil {
		return "", err
	}
	if found {
		return content, nil
	}

	// No stored version exists for this path (genuinely absent). Its advertised
	// latest_content_hash would be empty too, so the client won't pull it.
	return "", nil
}

// latestNoteVersionContent fetches the raw content of the latest stored version
// for a path directly from note_versions, bypassing the Layouts/NoteViews
// caches. It is robust to version_count divergence because it selects the row
// with the highest version number (NoteVersionHistoryByPath orders by version
// desc) rather than the one matching note_paths.version_count. Returns
// (content, found, error); found is false only when no version row exists.
func latestNoteVersionContent(ctx context.Context, env Env, path string) (string, bool, error) {
	rows, err := env.NoteVersionHistoryByPath(ctx, db.NoteVersionHistoryByPathParams{
		Value: path,
		Limit: 1,
	})
	if err != nil {
		return "", false, err
	}
	if len(rows) == 0 {
		return "", false, nil
	}

	nv, err := env.NoteVersionByID(ctx, rows[0].ID)
	if err != nil {
		return "", false, err
	}
	return nv.Content, true, nil
}
