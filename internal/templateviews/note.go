package templateviews

import (
	"bytes"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
	"trip2g/internal/formspec"
	"trip2g/internal/model"

	"golang.org/x/net/html"
)

// Note wraps model.NoteView for template usage.
// Provides a stable API that decouples templates from internal model changes.
type Note struct {
	nv           *model.NoteView
	domainHost   string
	lastEditedBy func() *NoteEditor
}

// NoteEditor identifies who pushed a note's current version: the signed-in user
// (web/site edits) and/or the API key (obsidian-sync pushes), plus when.
// Either identity may be empty when unknown.
//
// SECURITY: admin/editor-only data. Callers MUST gate display to admins/editors
// and never render it on public pages.
type NoteEditor struct {
	UserEmail         string    // email of the user who pushed (empty if none)
	APIKeyDescription string    // description of the API key that pushed (empty if none)
	Client            string    // value of X-trip2g-client header at push time (empty if absent)
	CreatedAt         time.Time // when the version was committed
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

// HasH1 returns true if the note content starts with an H1 heading,
// which is used as the title. When true, the template should skip
// rendering a separate <h1> title element.
func (n *Note) HasH1() bool {
	return n.nv.HasH1
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

func (n *Note) VersionID() string {
	return strconv.FormatInt(n.nv.VersionID, 10)
}

// Path returns the note path ID for data attributes.
func (n *Note) Path() string {
	return n.nv.Path
}

// IsSystem returns true if the note is a system note (any path component starts with "_").
func (n *Note) IsSystem() bool {
	return n.nv.IsSystem()
}

// Permalink returns the note URL path.
func (n *Note) Permalink() string {
	return n.nv.Permalink
}

// PermalinkEncoded returns the percent-encoded permalink for use in HTML href attributes.
func (n *Note) PermalinkEncoded() string {
	return n.nv.PermalinkEncoded()
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

// Author returns the frontmatter author (empty if unset).
func (n *Note) Author() string {
	return n.nv.Author
}

// UpdatedAt returns the frontmatter updated/modified time (zero if unset).
func (n *Note) UpdatedAt() time.Time {
	return n.nv.UpdatedAt
}

// Tags returns the frontmatter tags (nil if unset).
func (n *Note) Tags() []string {
	return n.nv.Tags
}

// OGImageURL returns the resolved social-preview image URL (empty if unset).
func (n *Note) OGImageURL() string {
	return n.nv.OGImageURL()
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

// SetLastEditedByResolver attaches a lazy resolver that loads who pushed the
// note's current version (backed by the NoteVersionEditor read query). Used by
// admin/editor render paths; left unset on public renders.
func (n *Note) SetLastEditedByResolver(resolve func() *NoteEditor) {
	n.lastEditedBy = resolve
}

// LastEditedBy returns who pushed the note's current version, or nil when
// unknown or no resolver is attached. The resolver runs on first access.
//
// SECURITY: admin/editor-only data. Callers MUST gate display to
// admins/editors and never render it on public pages.
func (n *Note) LastEditedBy() *NoteEditor {
	if n.lastEditedBy == nil {
		return nil
	}
	return n.lastEditedBy()
}

// LastEditedByLabel returns a short human label for who pushed the note's
// current version: the editor's email, or failing that the API key description,
// with " · via <client>" appended when the push carried an X-trip2g-client
// header. Returns "" when there is no editor.
//
// SECURITY: admin/editor-only data. Callers MUST gate rendering on
// currentUser.IsAdmin() and never expose it on public pages.
func (n *Note) LastEditedByLabel() string {
	editor := n.LastEditedBy()
	if editor == nil {
		return ""
	}

	label := editor.UserEmail
	if label == "" {
		label = editor.APIKeyDescription
	}
	if editor.Client != "" {
		label += " · via " + editor.Client
	}
	return label
}

// HasCharts reports whether the note contains any datachart blocks. Drives
// conditional loading of the chart widget script.
func (n *Note) HasCharts() bool {
	return len(n.nv.Charts) > 0
}

// HasCodeLanguage reports whether the note contains a fenced code block with the
// given language (case-insensitive). Drives conditional loading of per-language
// widget scripts (e.g. mermaid).
func (n *Note) HasCodeLanguage(lang string) bool {
	return n.nv.HasCodeLanguage(lang)
}

// HasAnyCodeBlock reports whether the note contains at least one fenced code
// block (any language). Used to conditionally load the codeblock widget (copy
// button + syntax highlighting).
func (n *Note) HasAnyCodeBlock() bool {
	return len(n.nv.CodeLanguages) > 0
}

var langAliases = map[string]string{ //nolint:gochecknoglobals // package-level lookup table
	"english":    "en",
	"russian":    "ru",
	"deutsch":    "de",
	"german":     "de",
	"french":     "fr",
	"español":    "es",
	"spanish":    "es",
	"chinese":    "zh",
	"japanese":   "ja",
	"portuguese": "pt",
	"arabic":     "ar",
	"korean":     "ko",
	"us":         "en",
}

var langNames = map[string]string{ //nolint:gochecknoglobals // package-level lookup table
	"en": "English",
	"ru": "Русский",
	"de": "Deutsch",
	"fr": "Français",
	"es": "Español",
	"zh": "中文",
	"ja": "日本語",
	"pt": "Português",
	"ar": "العربية",
	"ko": "한국어",
}

func normalizeLang(lang string) string {
	lower := strings.ToLower(strings.TrimSpace(lang))
	if code, ok := langAliases[lower]; ok {
		return code
	}
	return lower
}

// Lang returns the normalized language code (e.g., "en", "ru").
// Empty string if not set.
func (n *Note) Lang() string {
	return normalizeLang(n.nv.Lang)
}

// LangName returns the native display name for the note's language.
// Falls back to the normalized code if unknown.
func (n *Note) LangName() string {
	code := normalizeLang(n.nv.Lang)
	if name, ok := langNames[code]; ok {
		return name
	}
	return code
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

// FirstImageURL returns the URL of the first image asset.
// Returns "" if no first image or no asset replace found.
func (n *Note) FirstImageURL() string {
	if n.nv.FirstImage == nil {
		return ""
	}
	ar, ok := n.nv.AssetReplaces[*n.nv.FirstImage]
	if !ok || ar == nil {
		return ""
	}
	return ar.URL
}

// FirstListHTML returns the outer HTML of the first <ul> element in rendered HTML.
// Returns "" if no <ul> found.
func (n *Note) FirstListHTML() string {
	htmlStr := n.HTMLString()
	if htmlStr == "" {
		return ""
	}

	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return ""
	}

	ul := findFirstElement(doc, "ul")
	if ul == nil {
		return ""
	}

	var buf bytes.Buffer
	if renderErr := html.Render(&buf, ul); renderErr != nil {
		return ""
	}
	return buf.String()
}

// findFirstElement finds the first element with the given tag name in the HTML tree.
func findFirstElement(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findFirstElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}

// FormSpecJSON returns JSON for the <script id="form-spec"> tag a custom layout
// embeds in the page. The payload mirrors the default template's form-spec:
//
//	{ "note_version_id": 123, "forms": { "": {...}, "named": {...} } }
//
// Returns "" if the note has neither `form:` nor `forms:` in frontmatter.
// `form_ref:` is not resolved here — it requires a NVS context; layouts that
// need it should keep the form spec inline on the rendered note.
func (n *Note) FormSpecJSON() string {
	if n.nv == nil {
		return ""
	}
	rawMeta := n.nv.RawMeta

	forms := make(map[string]*formspec.FormSpec)

	if formsRaw, ok := rawMeta["forms"]; ok {
		formsMap, isMap := formsRaw.(map[string]interface{})
		if !isMap {
			return ""
		}
		for key, fRaw := range formsMap {
			spec, err := formspec.ParseFromRawMeta(map[string]interface{}{"form": fRaw})
			if err != nil || spec == nil {
				continue
			}
			forms[key] = spec
		}
	} else {
		spec, err := formspec.ParseFromRawMeta(rawMeta)
		if err != nil || spec == nil {
			return ""
		}
		forms[""] = spec
	}

	if len(forms) == 0 {
		return ""
	}

	payload := struct {
		NoteVersionID int64                         `json:"note_version_id"`
		Forms         map[string]*formspec.FormSpec `json:"forms"`
	}{
		NoteVersionID: n.nv.VersionID,
		Forms:         forms,
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(payload); err != nil {
		return ""
	}
	return strings.TrimRight(buf.String(), "\n")
}

// SubgraphNamesJSON returns JSON of note's SubgraphNames for paywall widget.
func (n *Note) SubgraphNamesJSON() string {
	raw, err := json.Marshal(n.nv.SubgraphNames)
	if err != nil {
		return "null"
	}
	return string(raw)
}
