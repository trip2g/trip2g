package mcp

import (
	"context"
	"html/template"

	"trip2g/internal/embedguard"
	"trip2g/internal/model"
)

type mcpNoteViewer interface {
	mcpNoteReader
	LatestNoteViews() *model.NoteViews
}

// mcpEmbedReader judges an embedded note the way the MCP surface judges any
// note, API keys and federation scopes included.
type mcpEmbedReader struct {
	env mcpNoteReader
}

func (r mcpEmbedReader) CanReadNote(ctx context.Context, note *model.NoteView) (bool, error) {
	return canReadMCPNote(ctx, r.env, note)
}

// readableNote is note with the embedded notes the caller cannot read taken
// out of its HTML. The shared note is never changed.
func readableNote(ctx context.Context, env mcpNoteViewer, note *model.NoteView) *model.NoteView {
	if !embedguard.HasMarkers(string(note.HTML)) {
		return note
	}

	viewer := embedguard.NewViewer(ctx, mcpEmbedReader{env: env}, env.LatestNoteViews())

	noteCopy := *note
	noteCopy.HTML = template.HTML(viewer.Strip(string(note.HTML))) //nolint:gosec // the note's own HTML, minus the blocks the caller cannot read

	return &noteCopy
}
