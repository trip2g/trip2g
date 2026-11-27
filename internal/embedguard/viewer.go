package embedguard

import (
	"context"
	"html"
	"sync"

	"trip2g/internal/model"
)

// Reader answers whether the current viewer may read a note.
type Reader interface {
	CanReadNote(ctx context.Context, note *model.NoteView) (bool, error)
}

// Viewer strips from note HTML the embedded notes its holder cannot read.
type Viewer struct {
	ctx    context.Context //nolint:containedctx // bound to one request for its whole life
	reader Reader
	notes  *model.NoteViews

	openPartials bool

	mu      sync.Mutex
	byID    map[int64]*model.NoteView
	verdict map[int64]verdict
}

type verdict struct {
	keep        bool
	replacement string
}

// NewViewer binds the request's viewer, as ctx identifies it, to the notes the
// embedded blocks are looked up in.
func NewViewer(ctx context.Context, reader Reader, notes *model.NoteViews) *Viewer {
	return &Viewer{
		ctx:     ctx,
		reader:  reader,
		notes:   notes,
		verdict: make(map[int64]verdict),
	}
}

// WithOpenPartials lets the viewer keep ungated partials and site chrome whole,
// whoever reads them, as a rendered page does: they are public by
// construction. API consumers leave it off and answer to CanReadNote alone.
func (v *Viewer) WithOpenPartials() *Viewer {
	v.openPartials = true

	return v
}

// Strip returns htmlText without the markers and without every embedded block
// the viewer cannot read. A nil Viewer refuses every block.
func (v *Viewer) Strip(htmlText string) string {
	if !HasMarkers(htmlText) {
		return htmlText
	}

	if v == nil {
		return Strip(htmlText, refuse)
	}

	return Strip(htmlText, v.decide)
}

func refuse(int64) (bool, string) {
	return false, ""
}

func (v *Viewer) decide(id int64) (bool, string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	cached, ok := v.verdict[id]
	if ok {
		return cached.keep, cached.replacement
	}

	result := v.judge(id)
	v.verdict[id] = result

	return result.keep, result.replacement
}

func (v *Viewer) judge(id int64) verdict {
	note := v.lookup(id)
	if note == nil {
		return verdict{}
	}

	if v.CanRead(note) {
		return verdict{keep: true}
	}

	return verdict{replacement: link(note)}
}

// CanRead reports whether the viewer may read note, by the same rule Strip
// applies to an embedded block. A nil Viewer is an anonymous one.
func (v *Viewer) CanRead(note *model.NoteView) bool {
	if note == nil {
		return false
	}

	if v == nil {
		return note.IsAnonymouslyReadable() || note.IsUngatedPartial() || note.IsSiteChrome()
	}

	if v.openPartials && (note.IsUngatedPartial() || note.IsSiteChrome()) {
		return true
	}

	canRead, err := v.reader.CanReadNote(v.ctx, note)

	return err == nil && canRead
}

func (v *Viewer) lookup(id int64) *model.NoteView {
	if v.notes == nil {
		return nil
	}

	if v.byID == nil {
		v.byID = make(map[int64]*model.NoteView, len(v.notes.PathMap))
		for _, note := range v.notes.PathMap {
			v.byID[note.PathID] = note
		}
	}

	return v.byID[id]
}

func link(note *model.NoteView) string {
	return `<a href="` + html.EscapeString(note.PermalinkEncoded()) + `">` + html.EscapeString(note.Title) + `</a>`
}
