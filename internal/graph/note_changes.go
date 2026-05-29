package graph

import (
	"trip2g/internal/graph/model"
	"trip2g/internal/notebus"
)

func (r *subscriptionResolver) buildNoteChangeItems(batch notebus.Batch) []model.NoteChangeItem {
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
