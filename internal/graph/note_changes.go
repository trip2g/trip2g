package graph

import (
	"context"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/notebus"
)

func (r *subscriptionResolver) buildNoteChangeItems(ctx context.Context, batch notebus.Batch) []model.NoteChangeItem {
	nvs := r.DefaultEnv.LatestNoteViews()
	var items []model.NoteChangeItem
	for _, change := range batch.Changes {
		switch change.Event {
		case "create", "update":
			ev := model.NoteUpsertEvent{
				Path:      change.Path,
				PathID:    change.PathID,
				EventType: model.NoteUpsertEventType(change.Event),
			}
			if nv := nvs.GetByPathID(change.PathID); nv != nil {
				ev.VersionID = nv.VersionID
				ev.Title = nv.Title
				ev.NoteView = nv
			} else if change.Path != "" {
				// The note isn't in LatestNoteViews (e.g. not currently cached). Emit the real
				// latest version id from the DB rather than leaving 0: a 0 here is <= any client's
				// self-echo baseline and would be wrongly suppressed, dropping a genuine change.
				ev.VersionID = r.latestVersionIDByPath(ctx, change.Path)
			}
			items = append(items, ev)
		case "remove":
			ev := model.NoteHideEvent{Path: change.Path}
			if change.PathID != 0 {
				id := change.PathID
				ev.PathID = &id
			}
			items = append(items, ev)
		}
	}
	return items
}

// latestVersionIDByPath returns the latest stored version id for a note path, or 0 if
// it cannot be resolved. The id is note_versions.id — the same space as NoteView.VersionID
// and the admin noteVersionHistory, so a client can compare it against its baseline.
func (r *subscriptionResolver) latestVersionIDByPath(ctx context.Context, path string) int64 {
	rows, err := r.DefaultEnv.NoteVersionHistoryByPath(ctx, db.NoteVersionHistoryByPathParams{
		Value: path,
		Limit: 1,
	})
	if err != nil || len(rows) == 0 {
		return 0
	}
	return rows[0].ID
}
