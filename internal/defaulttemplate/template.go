package defaulttemplate

import (
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/model"
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

	Title     string
	JSURLs    []string
	CSSURLs   []string
	InlineCSS string
	DevMode   string

	MetaDescription *string
	MetaRobots      string

	OGTags map[string]string

	HTMLInjections map[string][]db.HtmlInjection

	HrefLangs []HrefLang
	HTMLLang  string // for <html lang="xx">, set from note.Lang
	UILang    string // user's preferred interface language, set from trip2g_lang cookie

	OnboardingMode bool
	NotFoundMode   bool
	PaywallError   *PaywallError
	UserToken      *usertoken.Data
	Lang           string

	TelegramLinks []model.TelegramPostLink
}

// AllTelegramLinks returns TelegramLinks from DB/frontmatter (Source 1+2)
// merged with links resolved from frontmatter alternatives (Source 3).
func (ctx *Ctx) AllTelegramLinks() []model.TelegramPostLink {
	result := append([]model.TelegramPostLink{}, ctx.TelegramLinks...)

	if ctx.Note == nil || ctx.Notes == nil {
		return result
	}

	raw := ctx.Note.M().Get("alternatives")
	if raw == nil {
		return result
	}

	altList, ok := raw.([]interface{})
	if !ok {
		return result
	}

	for _, alt := range altList {
		altStr, ok := alt.(string)
		if !ok {
			continue
		}
		// Strip [[ and ]] from wikilink syntax.
		target := strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(altStr), "]]"), "[[")
		if target == "" {
			continue
		}

		resolved := ctx.Notes.ByWikilink(target)
		if resolved == nil {
			continue
		}

		link, ok := resolved.Unwrap().ExtractTelegramPublishMessageLink()
		if !ok {
			continue
		}

		result = append(result, model.TelegramPostLink{
			ChatTitle: model.ExtractChannelFromTelegramLink(link),
			URL:       link,
		})
	}

	return result
}

// noteExists returns true if a note with the given wikilink name exists in ctx.Notes.
func (ctx *Ctx) noteExists(name string) bool {
	if ctx.Notes == nil {
		return false
	}
	permalink := "/" + strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	return ctx.Notes.ByPermalink(permalink) != nil
}

// SidebarWidgets returns widgets for the given position ("left" or "right").
// If no frontmatter key is set, falls back to _left_sidebar.md / _right_sidebar.md if they exist.
// Returns nil if explicitly set to false or no default file found.
func (ctx *Ctx) SidebarWidgets(position string) []WidgetRef {
	key := position + "_sidebar"
	defaultName := "_" + key

	if ctx.Note == nil {
		// No current note: use default file if it exists.
		if ctx.noteExists(defaultName) {
			return []WidgetRef{{Kind: WidgetContent, Value: defaultName}}
		}
		return nil
	}

	m := ctx.Note.M()
	raw := m.Get(key)

	if raw == nil {
		// No frontmatter key: use default file if it exists.
		if ctx.noteExists(defaultName) {
			return []WidgetRef{{Kind: WidgetContent, Value: defaultName}}
		}
		return nil
	}

	// Check for bool false (sidebar explicitly disabled).
	if b, ok := raw.(bool); ok && !b {
		return nil
	}

	// Parse as slice.
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
// If not set, falls back to _header.md if it exists in the vault.
// Returns ContentRefNone if explicitly set to false/none or no default file found.
func (ctx *Ctx) HeaderRef() ContentRef {
	if ctx.Note != nil {
		raw := ctx.Note.M().Get("header")
		if raw != nil {
			return parseContentRef(raw)
		}
	}
	if ctx.noteExists("_header") {
		return ContentRef{Kind: ContentRefWikiLink, Value: "_header"}
	}
	return ContentRef{Kind: ContentRefNone}
}

// FooterRef returns the footer reference from frontmatter.
// If not set, falls back to _footer.md if it exists in the vault.
// Returns ContentRefNone if explicitly set to false/none or no default file found.
func (ctx *Ctx) FooterRef() ContentRef {
	if ctx.Note != nil {
		raw := ctx.Note.M().Get("footer")
		if raw != nil {
			return parseContentRef(raw)
		}
	}
	if ctx.noteExists("_footer") {
		return ContentRef{Kind: ContentRefWikiLink, Value: "_footer"}
	}
	return ContentRef{Kind: ContentRefNone}
}

// MagazineSortProperty returns the frontmatter key used for magazine sorting.
// Notes with this property are listed first (sorted by its value desc),
// followed by the rest sorted by created_at desc.
// Returns "" when not set — all notes are sorted by created_at desc.
func (ctx *Ctx) MagazineSortProperty() string {
	if ctx.Note == nil {
		return ""
	}
	return ctx.Note.M().GetString("magazine_sort_property", "")
}

// MagazineIncludeProperty returns the frontmatter key used to filter magazine notes.
// When set, only notes that have this property are included.
// Returns "" when not set — all notes matched by the glob are included.
func (ctx *Ctx) MagazineIncludeProperty() string {
	if ctx.Note == nil {
		return ""
	}
	return ctx.Note.M().GetString("magazine_include_property", "")
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
