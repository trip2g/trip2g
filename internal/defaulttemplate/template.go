package defaulttemplate

import (
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"
	"trip2g/internal/usertoken"

	"github.com/bmatcuk/doublestar/v4"
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

// SigninWallError holds data needed to render the sign-in wall page.
// Defined here (not imported from rendernotepage) to avoid import cycles.
type SigninWallError struct {
	Note *templateviews.Note
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

	OnboardingMode  bool
	NotFoundMode    bool
	PaywallError    *PaywallError
	SigninWallError *SigninWallError
	UserToken       *usertoken.Data
	Lang            string

	TelegramLinks []model.TelegramPostLink
	SimilarNotes  []*model.NoteView

	// LayoutSections holds vault-based layout section files for glob-based
	// header/footer/sidebar resolution. Populated from NoteViews.LayoutSections.
	LayoutSections []model.LayoutSectionEntry
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
		altStr, isStr := alt.(string)
		if !isStr {
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

		link, hasLink := resolved.Unwrap().ExtractTelegramPublishMessageLink()
		if !hasLink {
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
// Priority: 1) per-note key, 2) glob-matched layout section, 3) _left_sidebar/_right_sidebar fallback.
// Returns nil if explicitly set to false or no match found.
func (ctx *Ctx) SidebarWidgets(position string) []WidgetRef {
	key := position + "_sidebar"
	defaultName := "_" + key

	if ctx.Note == nil {
		if ctx.noteExists(defaultName) {
			return []WidgetRef{{Kind: WidgetContent, Value: defaultName}}
		}
		return nil
	}

	m := ctx.Note.M()
	raw := m.Get(key)

	if raw == nil {
		// Check glob-matched layout section before fallback.
		if match := ctx.resolveLayoutSection(key); match != nil {
			// If the layout section note declares content: [widget, ...], use those as widgets.
			if sectionNote := ctx.Notes.ByPath(match.NotePath); sectionNote != nil {
				if widgets := parseWidgetList(sectionNote.M().Get("content")); len(widgets) > 0 {
					return widgets
				}
			}
			return []WidgetRef{{Kind: WidgetContent, Value: match.NotePath}}
		}
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
// Priority: 1) per-note header: key, 2) glob-matched layout section, 3) _header fallback.
func (ctx *Ctx) HeaderRef() ContentRef {
	if ctx.Note != nil {
		raw := ctx.Note.M().Get("header")
		if raw != nil {
			return parseContentRef(raw)
		}
	}
	if match := ctx.resolveLayoutSection("header"); match != nil {
		return ContentRef{Kind: ContentRefFile, Value: match.NotePath}
	}
	if ctx.noteExists("_header") {
		return ContentRef{Kind: ContentRefWikiLink, Value: "_header"}
	}
	return ContentRef{Kind: ContentRefNone}
}

// FooterRef returns the footer reference from frontmatter.
// Priority: 1) per-note footer: key, 2) glob-matched layout section, 3) _footer fallback.
func (ctx *Ctx) FooterRef() ContentRef {
	if ctx.Note != nil {
		raw := ctx.Note.M().Get("footer")
		if raw != nil {
			return parseContentRef(raw)
		}
	}
	if match := ctx.resolveLayoutSection("footer"); match != nil {
		return ContentRef{Kind: ContentRefFile, Value: match.NotePath}
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

// MagazineExcludeFiles returns the glob pattern for magazine note exclusion.
// Notes matching this pattern are removed from the magazine.
// Returns "" when not set — no notes are excluded.
func (ctx *Ctx) MagazineExcludeFiles() string {
	if ctx.Note == nil {
		return ""
	}
	return ctx.Note.M().GetString("magazine_exclude_files", "")
}

// MagazineExcludeProperty returns the frontmatter key used to exclude magazine notes.
// When set, notes that have this property are excluded.
// Returns "" when not set — no notes are excluded by property.
func (ctx *Ctx) MagazineExcludeProperty() string {
	if ctx.Note == nil {
		return ""
	}
	return ctx.Note.M().GetString("magazine_exclude_property", "")
}

// resolveLayoutSection finds the best matching layout section file for the given section type.
// Returns nil if no layout section matches the current note.
func (ctx *Ctx) resolveLayoutSection(section string) *model.LayoutSectionEntry {
	if ctx.Note == nil || len(ctx.LayoutSections) == 0 {
		return nil
	}

	notePath := ctx.Note.Path()
	noteMeta := ctx.Note.M()

	var best *model.LayoutSectionEntry
	for i := range ctx.LayoutSections {
		entry := &ctx.LayoutSections[i]
		if entry.Section != section {
			continue
		}

		// Check include patterns.
		if !matchAnyGlob(entry.Includes, notePath) {
			continue
		}

		// Check exclude patterns.
		if matchAnyGlob(entry.Excludes, notePath) {
			continue
		}

		// Check include property.
		if entry.IncludeProperty != "" && noteMeta.Get(entry.IncludeProperty) == nil {
			continue
		}

		// Check exclude property.
		if entry.ExcludeProperty != "" && noteMeta.Get(entry.ExcludeProperty) != nil {
			continue
		}

		// Higher priority wins.
		if best == nil || entry.Priority > best.Priority {
			best = entry
		}
	}

	return best
}

// matchAnyGlob returns true if any pattern matches the path.
func matchAnyGlob(patterns []string, path string) bool {
	for _, pattern := range patterns {
		if matched, _ := doublestar.Match(pattern, path); matched {
			return true
		}
	}
	return false
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
