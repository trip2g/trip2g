package templateviews

import (
	"time"
	"trip2g/internal/model"
)

// Note wraps model.NoteView for template usage.
// Provides a stable API that decouples templates from internal model changes.
type Note struct {
	nv *model.NoteView
}

// NewNote creates a new template Note wrapper.
func NewNote(nv *model.NoteView) *Note {
	if nv == nil {
		return nil
	}
	return &Note{nv: nv}
}

// Title returns the note title.
func (n *Note) Title() string {
	return n.nv.Title
}

// HTMLString returns the rendered HTML content.
func (n *Note) HTMLString() string {
	return string(n.nv.HTML)
}

// ContentString returns the raw markdown content.
func (n *Note) ContentString() string {
	return string(n.nv.Content)
}

// PathID returns the note path ID for data attributes.
func (n *Note) PathID() int64 {
	return n.nv.PathID
}

// Permalink returns the note URL path.
func (n *Note) Permalink() string {
	return n.nv.Permalink
}

// CreatedAt returns the note creation time.
func (n *Note) CreatedAt() time.Time {
	return n.nv.CreatedAt
}

// ReadingTime returns estimated reading time in minutes.
func (n *Note) ReadingTime() int {
	return n.nv.ReadingTime
}

// ReadingComplexity returns reading complexity (0-2).
func (n *Note) ReadingComplexity() int {
	return n.nv.ReadingComplexity
}

// IsHomePage returns true if this note is a subgraph home page.
func (n *Note) IsHomePage() bool {
	return n.nv.IsHomePage()
}

// Description returns the SEO meta description.
func (n *Note) Description() string {
	if n.nv.Description == nil {
		return ""
	}
	return *n.nv.Description
}

// PartialRenderer returns the partial renderer for content splitting.
func (n *Note) PartialRenderer() model.NoteViewPartialRenderer {
	return n.nv.PartialRenderer
}

// TOC returns the table of contents headings.
func (n *Note) TOC() model.NoteViewHeadings {
	return n.nv.TOC()
}

// M returns the meta accessor for frontmatter values.
func (n *Note) M() *Meta {
	return &Meta{raw: n.nv.RawMeta}
}

// Unwrap returns the underlying NoteView (for internal use).
func (n *Note) Unwrap() *model.NoteView {
	return n.nv
}
