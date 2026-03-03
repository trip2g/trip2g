package templateviews

import (
	"sort"
	"time"
	"trip2g/internal/model"
)

// Note wraps model.NoteView for template usage.
// Provides a stable API that decouples templates from internal model changes.
type Note struct {
	nv         *model.NoteView
	domainHost string
}

// NewNote creates a new template Note wrapper.
func NewNote(nv *model.NoteView) *Note {
	if nv == nil {
		return nil
	}
	return &Note{nv: nv}
}

// NewNoteWithDomain creates a Note wrapper with domain context for
// domain-aware HTML rendering in custom layouts.
func NewNoteWithDomain(nv *model.NoteView, domainHost string) *Note {
	if nv == nil {
		return nil
	}
	return &Note{nv: nv, domainHost: domainHost}
}

// Title returns the note title.
func (n *Note) Title() string {
	return n.nv.Title
}

// HTMLString returns the rendered HTML content.
// When domain context is set, returns domain-specific HTML if available.
// domainHost is "" for main domain — DomainHTML[""] holds main-domain re-rendered
// HTML where links to custom-domain-only notes use full URLs (https://foo.com/path).
func (n *Note) HTMLString() string {
	if n.nv.DomainHTML != nil {
		if domainHTML, ok := n.nv.DomainHTML[n.domainHost]; ok {
			return string(domainHTML)
		}
	}
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

// Path returns the note path ID for data attributes.
func (n *Note) Path() string {
	return n.nv.Path
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

// Lang returns the note's language code (e.g., "en", "ru").
// Empty string if not set.
func (n *Note) Lang() string {
	return n.nv.Lang
}

// HasLangAlternatives returns true if this note has language alternative versions.
func (n *Note) HasLangAlternatives() bool {
	return len(n.nv.LangAlternatives) > 0
}

// LangAlternative returns the Note wrapper for a specific language code.
// Returns nil if not found.
func (n *Note) LangAlternative(lang string) *Note {
	if n.nv.LangAlternatives == nil {
		return nil
	}
	alt, ok := n.nv.LangAlternatives[lang]
	if !ok {
		return nil
	}
	return NewNote(alt)
}

// LangAlternativesList returns all language alternatives as a sorted slice
// for template iteration (e.g., to render a language switcher).
func (n *Note) LangAlternativesList() []*Note {
	if len(n.nv.LangAlternatives) == 0 {
		return nil
	}
	langs := make([]string, 0, len(n.nv.LangAlternatives))
	for lang := range n.nv.LangAlternatives {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	result := make([]*Note, 0, len(langs))
	for _, lang := range langs {
		result = append(result, NewNote(n.nv.LangAlternatives[lang]))
	}
	return result
}
