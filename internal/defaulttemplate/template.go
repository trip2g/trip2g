package defaulttemplate

import (
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/templateviews"
	"trip2g/internal/usertoken"
)

//go:generate go tool github.com/valyala/quicktemplate/qtc -dir=. -ext=html

// HrefLang represents a single hreflang alternate link.
type HrefLang struct {
	Lang string // "en", "ru", "x-default"
	Href string // full URL including scheme+host
}

// PaywallError holds data needed to render the paywall page.
// Defined here (not imported from rendernotepage) to avoid import cycles.
type PaywallError struct {
	Note          *templateviews.Note
	SubgraphsJSON string // JSON of SubgraphNames for paywall widget
}

const (
	injectionPlaceholderHead    = "head"
	injectionPlaceholderBodyEnd = "body_end"
)

// Ctx holds all data needed to render a complete page.
type Ctx struct {
	Note  *templateviews.Note
	Notes *templateviews.NVS

	Title   string
	JSURLs  []string
	CSSURLs []string
	DevMode string

	MetaDescription *string
	MetaRobots      string

	OGTags map[string]string

	HTMLInjections map[string][]db.HtmlInjection

	HrefLangs []HrefLang
	HTMLLang  string // for <html lang="xx">

	OnboardingMode bool
	PaywallError   *PaywallError
	UserToken      *usertoken.Data
	Lang           string
	IsAdmin        bool
}

// SidebarWidgets returns widgets for the given position ("left" or "right").
// Returns nil if not configured or explicitly set to false.
func (ctx *Ctx) SidebarWidgets(position string) []WidgetRef {
	if ctx.Note == nil {
		return nil
	}

	m := ctx.Note.M()
	key := position + "_sidebar"
	raw := m.Get(key)
	if raw == nil {
		return nil
	}

	// Check for bool false (sidebar disabled)
	if b, ok := raw.(bool); ok && !b {
		return nil
	}

	// Parse as slice
	var items []interface{}
	switch v := raw.(type) {
	case []interface{}:
		items = v
	case []string:
		for _, s := range v {
			items = append(items, s)
		}
	case string:
		items = []interface{}{v}
	default:
		return nil
	}

	var widgets []WidgetRef
	for _, item := range items {
		if w, ok := parseWidgetRef(item); ok {
			widgets = append(widgets, w)
		}
	}

	if len(widgets) == 0 {
		return nil
	}
	return widgets
}

// ContentRefs returns the list of content blocks to render.
// If Note is nil: returns [{Magazine}] (root URL with no index note).
// If no "content" key: returns [{SelfContent}].
func (ctx *Ctx) ContentRefs() []ContentRef {
	if ctx.Note == nil {
		return []ContentRef{{Kind: ContentRefMagazine}}
	}

	m := ctx.Note.M()
	raw := m.Get("content")

	if raw == nil {
		return []ContentRef{{Kind: ContentRefSelfContent}}
	}

	// Parse as slice
	var items []interface{}
	switch v := raw.(type) {
	case []interface{}:
		items = v
	case []string:
		for _, s := range v {
			items = append(items, s)
		}
	case string:
		items = []interface{}{v}
	case bool:
		if !v {
			return nil
		}
		return []ContentRef{{Kind: ContentRefSelfContent}}
	default:
		return []ContentRef{{Kind: ContentRefSelfContent}}
	}

	var refs []ContentRef
	for _, item := range items {
		ref := parseContentRef(item)
		refs = append(refs, ref)
	}

	if len(refs) == 0 {
		return []ContentRef{{Kind: ContentRefSelfContent}}
	}
	return refs
}

// HeaderRef returns the header reference from frontmatter.
// Falls back to _navigation wikilink as site-wide default when not explicitly set.
// Returns ContentRefNone only if explicitly set to false/none.
func (ctx *Ctx) HeaderRef() ContentRef {
	if ctx.Note != nil {
		raw := ctx.Note.M().Get("header")
		if raw != nil {
			return parseContentRef(raw)
		}
	}
	return ContentRef{Kind: ContentRefWikiLink, Value: "_navigation"}
}

// FooterRef returns the footer reference from frontmatter.
// Falls back to _footer wikilink as site-wide default when not explicitly set.
// Returns ContentRefNone only if explicitly set to false/none.
func (ctx *Ctx) FooterRef() ContentRef {
	if ctx.Note != nil {
		raw := ctx.Note.M().Get("footer")
		if raw != nil {
			return parseContentRef(raw)
		}
	}
	return ContentRef{Kind: ContentRefWikiLink, Value: "_footer"}
}

// MagazineProperty returns the frontmatter key used for magazine sorting.
// Defaults to "magazine_priority".
func (ctx *Ctx) MagazineProperty() string {
	if ctx.Note == nil {
		return "magazine_priority"
	}
	return ctx.Note.M().GetString("magazine_property", "magazine_priority")
}

// MagazineIncludeFiles returns the glob pattern for magazine note inclusion.
// Defaults to "**/*.md".
func (ctx *Ctx) MagazineIncludeFiles() string {
	if ctx.Note == nil {
		return "**/*.md"
	}
	return ctx.Note.M().GetString("magazine_include_files", "**/*.md")
}

// resolveNoteRef resolves a ContentRef to a *templateviews.Note.
// Returns nil if the note cannot be found.
func (ctx *Ctx) resolveNoteRef(ref ContentRef) *templateviews.Note {
	if ctx.Notes == nil {
		return nil
	}
	switch ref.Kind {
	case ContentRefWikiLink:
		return ctx.Notes.ByPermalink("/" + strings.ToLower(strings.ReplaceAll(ref.Value, " ", "_")))
	case ContentRefFile:
		return ctx.Notes.ByPath(ref.Value)
	case ContentRefSelfContent, ContentRefMagazine, ContentRefNone:
		return nil
	default:
		return nil
	}
}
