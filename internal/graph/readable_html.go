package graph

import (
	"context"

	"trip2g/internal/embedguard"
	appmodel "trip2g/internal/model"
)

// readableHTML is the note's HTML without the embedded notes the requester
// cannot read.
func (r *Resolver) readableHTML(ctx context.Context, note *appmodel.NoteView) string {
	env := r.env(ctx)

	return embedguard.NewViewer(ctx, env, env.LatestNoteViews()).Strip(string(note.HTML))
}
