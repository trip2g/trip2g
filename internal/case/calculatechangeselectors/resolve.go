package calculatechangeselectors

import (
	"context"
	"html/template"

	"trip2g/internal/htmldiff"
	appmodel "trip2g/internal/model"
)

type Env interface {
	LatestNoteViews() *appmodel.NoteViews
	PreviousLatestNoteHTML(pathID int64) (template.HTML, bool)
}

// Resolve returns CSS selectors for changed top-level blocks between the previous
// and current rendered HTML of a note. Returns nil if no previous HTML is available
// (create event, patchesChanged reload cycle, or race with a subsequent save).
func Resolve(_ context.Context, env Env, pathID int64) ([]string, error) {
	oldHTML, ok := env.PreviousLatestNoteHTML(pathID)
	if !ok {
		return nil, nil
	}

	nv := env.LatestNoteViews().GetByPathID(pathID)
	if nv == nil {
		return nil, nil
	}

	selector := htmldiff.FirstChangedBlock(string(oldHTML), string(nv.HTML))
	if selector == "" {
		return nil, nil
	}

	return []string{selector}, nil
}
