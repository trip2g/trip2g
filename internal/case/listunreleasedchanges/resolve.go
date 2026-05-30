package listunreleasedchanges

import (
	"context"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/webhookutil"
)

type Env interface {
	AllLiveNotes(ctx context.Context) ([]db.AllLiveNotesRow, error)
	LatestNoteViews() *appmodel.NoteViews
}

// Resolve returns notes diverging between the live release and latest edited versions.
func Resolve(ctx context.Context, env Env, filter model.NoteChangesFilter) ([]*model.UnreleasedChange, error) {
	liveRows, err := env.AllLiveNotes(ctx)
	if err != nil {
		return nil, err
	}

	liveByPathID := make(map[int64]db.AllLiveNotesRow, len(liveRows))
	for _, row := range liveRows {
		if matchesFilter(row.Path, filter) {
			liveByPathID[row.PathID] = row
		}
	}

	latestViews := env.LatestNoteViews()
	seen := make(map[int64]bool)
	var changes []*model.UnreleasedChange

	for _, nv := range latestViews.List {
		if !matchesFilter(nv.Path, filter) {
			continue
		}
		newContent := string(nv.Content)
		lv, inLive := liveByPathID[nv.PathID]
		seen[nv.PathID] = true

		if !inLive {
			vid := nv.VersionID
			changes = append(changes, &model.UnreleasedChange{
				Path:            nv.Path,
				PathID:          nv.PathID,
				Title:           nv.Title,
				ChangeType:      model.NoteChangeTypeAdded,
				LatestVersionID: &vid,
				NewContent:      &newContent,
			})
		} else if lv.VersionID != nv.VersionID {
			lvid := lv.VersionID
			nvid := nv.VersionID
			oldContent := lv.Content
			changes = append(changes, &model.UnreleasedChange{
				Path:            nv.Path,
				PathID:          nv.PathID,
				Title:           nv.Title,
				ChangeType:      model.NoteChangeTypeModified,
				LiveVersionID:   &lvid,
				LatestVersionID: &nvid,
				OldContent:      &oldContent,
				NewContent:      &newContent,
			})
		}
	}

	for pathID, lv := range liveByPathID {
		if seen[pathID] {
			continue
		}
		lvid := lv.VersionID
		oldContent := lv.Content
		changes = append(changes, &model.UnreleasedChange{
			Path:          lv.Path,
			PathID:        pathID,
			Title:         "",
			ChangeType:    model.NoteChangeTypeRemoved,
			LiveVersionID: &lvid,
			OldContent:    &oldContent,
		})
	}

	return changes, nil
}

func matchesFilter(path string, filter model.NoteChangesFilter) bool {
	if !webhookutil.MatchesAny(path, filter.IncludePatterns) {
		return false
	}
	if len(filter.ExcludePatterns) > 0 && webhookutil.MatchesAny(path, filter.ExcludePatterns) {
		return false
	}
	return true
}
