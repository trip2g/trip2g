package main

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/templateviews"
)

// NoteVersionEditor loads who pushed the given note version (recorded at commit
// time) as a templateviews.NoteEditor. Returns nil when the version is unknown.
//
// SECURITY: admin/editor-only data. The caller is responsible for gating
// display to admins/editors — never expose it on public pages. Typical use is
// to attach it as a lazy resolver via Note.SetLastEditedByResolver on
// admin-only render paths.
func (a *app) NoteVersionEditor(ctx context.Context, versionID int64) (*templateviews.NoteEditor, error) {
	row, err := a.Queries.NoteVersionEditor(ctx, versionID)
	if err != nil {
		if db.IsNoFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load note version editor %d: %w", versionID, err)
	}

	editor := &templateviews.NoteEditor{CreatedAt: row.CreatedAt}
	if row.UserEmail != nil {
		editor.UserEmail = *row.UserEmail
	}
	if row.ApiKeyDescription != nil {
		editor.APIKeyDescription = *row.ApiKeyDescription
	}

	return editor, nil
}
