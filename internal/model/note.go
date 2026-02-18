package model

import (
	"fmt"
	"html/template"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"trip2g/internal/russkayalatinica"
	"unicode"

	"github.com/yuin/goldmark/ast"
)

// TOC display constants.
const (
	TOCDisplayAuto = iota
	TOCDisplayShow
	TOCDisplayHide
)

type NoteViewHeading struct {
	Text  string
	Level int
	ID    string
}

type NoteViewHeadings []NoteViewHeading

type NoteWarningLevel int

const (
	NoteWarningInfo NoteWarningLevel = iota
	NoteWarningWarning
	NoteWarningCritical
)

type NoteWarning struct {
	Level   NoteWarningLevel
	Message string
}

type NoteAssetReplace struct {
	ID   int64
	URL  string
	Hash string

	AbsolutePath string

	ExpiresAt time.Time
}

type NoteViewSection struct {
	Title       string
	TitleHTML   string
	ContentHTML string

	// SectionsFunc is set by PartialRenderer to enable nested section extraction.
	SectionsFunc func(level int) []NoteViewSection `json:"-"`
	// SectionFunc is set by PartialRenderer to enable finding a section by title.
	SectionFunc func(title string) *NoteViewSection `json:"-"`
}

// Sections returns subsections at the specified heading level.
// This allows nested iteration over sections in templates.
func (s *NoteViewSection) Sections(level int) []NoteViewSection {
	if s.SectionsFunc == nil {
		return nil
	}
	return s.SectionsFunc(level)
}

// Section finds a subsection by its heading title.
// Returns nil if no heading with the given title is found.
func (s *NoteViewSection) Section(title string) *NoteViewSection {
	if s.SectionFunc == nil {
		return nil
	}
	return s.SectionFunc(title)
}

// NoteViewHeadingBlock is an alias for NoteViewSection (deprecated, use NoteViewSection).
type NoteViewHeadingBlock = NoteViewSection

type NoteViewPartialRenderer interface {
	Sections(level int) []NoteViewSection
	Section(title string) *NoteViewSection
	Introduce() NoteViewSection

	// HeadingBlocks is deprecated, use Sections instead.
	HeadingBlocks(level int) []NoteViewSection
}

type SearchResult struct {
	HighlightedTitle   *string
	HighlightedContent []string
	URL                string
	Score              float64 // Combined score for ranking (higher is better)

	NoteView *NoteView
}

type AppliedFrontmatterPatch struct {
	PatchID     int
	Description string
}

type NoteView struct {
	Path  string
	Title string

	PathID    int64
	VersionID int64
	CreatedAt time.Time

	Content []byte
	HTML    template.HTML
	ast     ast.Node // hide from JSON

	PartialRenderer NoteViewPartialRenderer

	// If the field `free_paragraphs` is present, then thin field will
	// includes the first rendered paragraphs equal to this number.
	FreeHTML template.HTML

	Permalink string
	IsIndex   bool

	PermalinkOriginal string

	Free     bool // without the paywall
	Redirect *string

	Description *string // meta description for SEO

	InLinks         map[string]struct{} // permlinks of notes linking to this note
	RawMeta         map[string]interface{}
	OriginalRawMeta map[string]interface{} // RawMeta before patches, used to re-apply patches correctly on cached reloads

	ResolvedLinks map[string]string // local link to absolute link mapping

	SubgraphNames []string
	Subgraphs     map[string]*NoteSubgraph `json:"-"`

	Assets map[string]struct{}

	AssetReplaces map[string]*NoteAssetReplace

	ReadingTime       int // in minutes
	ReadingComplexity int // 0 - easy, 1 - medium, 2 - hard

	Headings NoteViewHeadings // extracted from AST

	HeadingCount map[string]int // for id generation

	TOCDisplay int // TOCDisplayAuto, TOCDisplayShow, TOCDisplayHide - from meta

	EmbededClass string

	Warnings                  []NoteWarning
	AppliedFrontmatterPatches []AppliedFrontmatterPatch

	FirstImage *string

	Layout string

	Slug string // custom URL override from YAML metadata

	// MCP (Model Context Protocol) fields
	MCPMethod      string // method name for MCP tools/list
	MCPDescription string // description for MCP tools/list

	// RSS feed fields from frontmatter.
	RSSTitle       string
	RSSDescription string

	// Routes holds the parsed routes for this note.
	// Populated from frontmatter "route" (string) or "routes" ([]string).
	// Does NOT affect Permalink or nv.Map. Only populates RouteMap.
	Routes []ParsedRoute

	// Vector embedding for semantic search (loaded separately)
	Embedding []float32 `json:"-"`
}

type NoteSubgraph struct {
	Name    string
	Home    *NoteView
	Sidebar *NoteView
}

type NoteViews struct {
	// Warning: this map may contain the same note under different URLs!
	// (For example: I spent an hour debugging a link resolution issue, and it turned out they were resolved twice.)
	Map map[string]*NoteView // TODO: rename to PermalinkMap

	PathMap map[string]*NoteView

	List []*NoteView

	Sitemap []byte `json:"-"`

	Subgraphs map[string]*NoteSubgraph `json:"-"`

	// RouteMap indexes notes by host -> path -> *NoteView.
	// host="" for main domain alias routes.
	// host="foo.com" for custom domain routes.
	// Only notes with explicit "route"/"routes" frontmatter are indexed here.
	RouteMap map[string]map[string]*NoteView

	// DomainSitemaps stores pre-generated sitemaps for each custom domain.
	// Key = normalized domain (e.g., "foo.com").
	DomainSitemaps map[string][]byte `json:"-"`

	Version string
}

func (n *NoteView) HTMLString() string {
	return string(n.HTML)
}

func (n *NoteView) ID() string {
	return n.Permalink
}

func (n *NoteView) Ast() ast.Node {
	return n.ast
}

func (n *NoteView) AddWarning(level NoteWarningLevel, message string, args ...any) {
	n.Warnings = append(n.Warnings, NoteWarning{
		Level:   level,
		Message: fmt.Sprintf(message, args...),
	})
}

func normalizeURLPart(s string) string {
	if len(s) == 0 {
		return ""
	}

	res := strings.ToLower(s)

	var b strings.Builder
	for _, r := range res {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}

	res = b.String()
	res = multiUnderscoreRE.ReplaceAllString(res, "_")
	res = strings.Trim(res, "_")

	if s[0] == '_' {
		res = "_" + res
	}

	return res
}

// escapeAndJoinPath escapes each segment of a path and joins them.
func escapeAndJoinPath(path string) string {
	parts := strings.Split(path, "/")
	escapedParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		escapedParts = append(escapedParts, url.PathEscape(part))
	}
	return "/" + strings.Join(escapedParts, "/")
}

// preparePermalinkFromSlug handles custom slug URL override.
func (n *NoteView) preparePermalinkFromSlug() {
	if strings.HasPrefix(n.Slug, "/") {
		// Absolute slug - use as full path
		n.IsIndex = strings.HasSuffix(n.Slug, "/index") || strings.HasSuffix(n.Slug, "/_index")
		n.Permalink = escapeAndJoinPath(n.Slug)
		n.PermalinkOriginal = n.Slug
		return
	}

	// Relative slug - replace only the filename, keep directory
	dir := filepath.Dir(n.Path)
	if dir == "." {
		dir = ""
	}

	var fullPath string
	if dir == "" {
		fullPath = "/" + n.Slug
	} else {
		fullPath = "/" + dir + "/" + n.Slug
	}

	n.IsIndex = strings.HasSuffix(n.Slug, "/index") || strings.HasSuffix(n.Slug, "/_index")
	n.Permalink = escapeAndJoinPath(fullPath)
	n.PermalinkOriginal = fullPath
}

func (n *NoteView) PreparePermalink() {
	// If slug is set in metadata, use it instead of normalizing path
	if n.Slug != "" {
		n.preparePermalinkFromSlug()
		return
	}

	// Default behavior: normalize path with transliteration
	link := n.Path

	// Remove .md extension if present
	if len(link) > 3 && link[len(link)-3:] == ".md" {
		link = link[:len(link)-3]
	}

	// Split path into parts
	parts := strings.Split(link, "/")
	newParts := make([]string, 0, len(parts))

	for idx, part := range parts {
		if part == "" {
			continue
		}

		// Normalize each part of the path
		np := normalizeURLPart(part)
		if np == "" {
			continue
		}

		isLast := idx == len(parts)-1
		if isLast && (np == "index" || np == "_index") {
			n.IsIndex = true
			break
		}

		newParts = append(newParts, np)
	}

	n.PermalinkOriginal = "/" + strings.Join(newParts, "/")
	n.Permalink = russkayalatinica.Translit(n.PermalinkOriginal)
}

func (n *NoteView) IsHomePage() bool {
	for _, subgraph := range n.SubgraphNames {
		if n.Permalink == "/"+subgraph {
			return true
		}
	}

	return false
}

func (n *NoteView) SetAst(node ast.Node) {
	n.ast = node
}

// TOC returns the table of contents headings based on the TOCDisplay setting.
func (n *NoteView) TOC() NoteViewHeadings {
	switch n.TOCDisplay {
	case TOCDisplayHide:
		return NoteViewHeadings{}
	case TOCDisplayShow:
		return n.Headings
	case TOCDisplayAuto:
		// Auto mode: show TOC if:
		// - There are 5 or more headings, OR
		// - Reading time is 2 minutes or more
		if len(n.Headings) >= 5 || n.ReadingTime >= 2 {
			return n.Headings
		}
		return NoteViewHeadings{}
	default:
		return NoteViewHeadings{}
	}
}

func (n *NoteView) ExtractSubgraphs() error {
	subgraphs := make(map[string]struct{})

	err := extractSubgraphs(subgraphs, n.RawMeta["subgraph"])
	if err != nil {
		return fmt.Errorf("error extracting subgraph: %w", err)
	}

	err = extractSubgraphs(subgraphs, n.RawMeta["subgraphs"])
	if err != nil {
		return fmt.Errorf("error extracting subgraphs: %w", err)
	}

	res := make([]string, 0, len(subgraphs))

	for k := range subgraphs {
		res = append(res, k)
	}

	n.SubgraphNames = res

	return nil
}

func (n *NoteView) ExtractMetaData() error {
	var err error

	n.Redirect, err = n.extractString("redirect")
	if err != nil {
		// TODO: add this error to warnings
		return err
	}

	n.Description, err = n.extractString("description")
	if err != nil {
		return err
	}

	n.extractReadingTime()

	err = n.extractReadingComplexity()
	if err != nil {
		return err
	}

	n.extractHeadingsAndGenerateIDs()

	n.extractTOCDisplay()

	n.extractEmbededClass()

	n.extractLayout()

	n.extractMCPFields()

	n.extractRSSFields()

	n.Routes = n.ExtractRoutes()

	return nil
}

func (n *NoteView) extractMCPFields() {
	if method, ok := n.RawMeta["mcp_method"].(string); ok {
		n.MCPMethod = method
	}
	if desc, ok := n.RawMeta["mcp_description"].(string); ok {
		n.MCPDescription = desc
	}
}

func (n *NoteView) extractRSSFields() {
	if title, ok := n.RawMeta["rss_title"].(string); ok {
		n.RSSTitle = title
	}
	if desc, ok := n.RawMeta["rss_description"].(string); ok {
		n.RSSDescription = desc
	}
}

func (n *NoteView) extractLayout() {
	layout, ok := n.RawMeta["layout"]
	if ok {
		layoutStr, isString := layout.(string)
		if isString {
			n.Layout = layoutStr
		} else {
			n.Warnings = append(n.Warnings, NoteWarning{
				Level:   NoteWarningWarning,
				Message: fmt.Sprintf("invalid layout type: %T, must be string", layout),
			})
		}
	}
}

var classRE = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
var multiUnderscoreRE = regexp.MustCompile(`_+`)

func (n *NoteView) extractEmbededClass() {
	valueI, ok := n.RawMeta["embed_class"]
	if ok {
		value, isString := valueI.(string)
		if isString {
			n.EmbededClass = classRE.ReplaceAllString(value, "_")
		}
	}
}

func (n *NoteView) extractReadingTime() {
	if len(n.Content) == 0 {
		n.ReadingTime = 0
		return
	}

	// Check if reading time is set in metadata
	if readingTimeI, ok := n.RawMeta["reading_time"]; ok {
		switch rt := readingTimeI.(type) {
		case int:
			n.ReadingTime = rt
			return
		case float64:
			n.ReadingTime = int(rt)
			return
		case string:
			// Try to parse string as number
			if rt != "" {
				var parsedTime int
				if _, err := fmt.Sscanf(rt, "%d", &parsedTime); err == nil {
					n.ReadingTime = parsedTime
					return
				}
			}
		}
	}

	// Short-circuit for short content: ~1000 chars per minute
	estimatedTime := len(n.Content) / 1000
	if estimatedTime <= 4 {
		if estimatedTime < 1 {
			estimatedTime = 1
		}
		n.ReadingTime = estimatedTime
		return
	}

	// For longer content, count words (single-pass, no allocations)
	wordCount := countWordsForReadingTime(string(n.Content))

	// Average reading speed: 200 words per minute
	readingTime := (wordCount + 199) / 200
	if readingTime < 1 {
		readingTime = 1
	}

	n.ReadingTime = readingTime
}

func (n *NoteView) extractReadingComplexity() error {
	// Default complexity is 0 (easy)
	n.ReadingComplexity = 0

	// Check if reading complexity is set in metadata
	complexityI, ok := n.RawMeta["complexity"]
	if !ok {
		// Also check for "reading_complexity" key
		complexityI, ok = n.RawMeta["reading_complexity"]
		if !ok {
			return nil
		}
	}

	switch complexity := complexityI.(type) {
	case int:
		if complexity >= 0 && complexity <= 2 {
			n.ReadingComplexity = complexity
		} else {
			return fmt.Errorf("invalid reading complexity: %d, must be 0 (easy), 1 (medium), or 2 (hard)", complexity)
		}
	case float64:
		complexityInt := int(complexity)
		if complexityInt >= 0 && complexityInt <= 2 {
			n.ReadingComplexity = complexityInt
		} else {
			return fmt.Errorf("invalid reading complexity: %f, must be 0 (easy), 1 (medium), or 2 (hard)", complexity)
		}
	case string:
		switch strings.ToLower(complexity) {
		case "easy", "0":
			n.ReadingComplexity = 0
		case "medium", "1":
			n.ReadingComplexity = 1
		case "hard", "2":
			n.ReadingComplexity = 2
		default:
			return fmt.Errorf("invalid reading complexity: %s, must be 'easy', 'medium', 'hard', or 0-2", complexity)
		}
	default:
		return fmt.Errorf("invalid reading complexity type: %T, must be int, float64, or string", complexity)
	}

	return nil
}

func (n *NoteView) extractTOCDisplay() {
	// Default to auto
	n.TOCDisplay = TOCDisplayAuto

	// Check if toc is set in metadata
	tocI, ok := n.RawMeta["toc"]
	if !ok {
		return
	}

	switch toc := tocI.(type) {
	case string:
		switch strings.ToLower(toc) {
		case "auto":
			n.TOCDisplay = TOCDisplayAuto
		case "show", "true", "yes":
			n.TOCDisplay = TOCDisplayShow
		case "hide", "false", "no":
			n.TOCDisplay = TOCDisplayHide
		}
	case bool:
		if toc {
			n.TOCDisplay = TOCDisplayShow
		} else {
			n.TOCDisplay = TOCDisplayHide
		}
	}
}

var onlyCharsRE = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func (n *NoteView) generateHeadingID(headingText string) string {
	id := russkayalatinica.Translit(headingText)
	id = onlyCharsRE.ReplaceAllString(id, "_")
	id = strings.ToLower(id)
	id = fmt.Sprintf("%s_%d", id, n.PathID)

	if n.HeadingCount == nil {
		n.HeadingCount = make(map[string]int)
	}

	n.HeadingCount[id]++

	if n.HeadingCount[id] > 1 {
		id = fmt.Sprintf("%s-%d", id, n.HeadingCount[id])
	}

	return id
}

func (n *NoteView) extractHeadingsAndGenerateIDs() {
	n.Headings = nil

	if n.ast == nil {
		return
	}

	_ = ast.Walk(n.ast, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if heading, ok := node.(*ast.Heading); ok {
			headingText := extractHeadingText(n.Content, heading)
			if headingText != "" {
				var id string

				rawID, withID := node.AttributeString("id")
				if withID {
					id = string(rawID.([]byte)) //nolint:errcheck // type assertion is safe here
				} else {
					id = n.generateHeadingID(headingText)
					node.SetAttributeString("id", []byte(id))
				}

				n.Headings = append(n.Headings, NoteViewHeading{
					Text:  headingText,
					Level: heading.Level,
					ID:    id,
				})
			}
		}

		return ast.WalkContinue, nil
	})

	n.Headings.Normalize()
}

func (nv NoteViewHeadings) Normalize() {
	if len(nv) == 0 {
		return
	}

	// Collect all unique levels that exist
	existingLevels := make(map[int]bool)
	for _, heading := range nv {
		existingLevels[heading.Level] = true
	}

	// Create sorted slice of existing levels
	var levels []int
	for level := range existingLevels {
		levels = append(levels, level)
	}

	// Sort levels in ascending order
	for i := 0; i < len(levels); i++ {
		for j := i + 1; j < len(levels); j++ {
			if levels[j] < levels[i] {
				levels[i], levels[j] = levels[j], levels[i]
			}
		}
	}

	// Create mapping from old level to new level
	levelMapping := make(map[int]int)
	for i, oldLevel := range levels {
		levelMapping[oldLevel] = i + 1 // Start from 1
	}

	// Apply the mapping to all headings
	for i := range nv {
		nv[i].Level = levelMapping[nv[i].Level]
	}
}

func extractHeadingText(source []byte, heading *ast.Heading) string {
	var text strings.Builder

	for child := heading.FirstChild(); child != nil; child = child.NextSibling() {
		extractTextFromNode(source, child, &text)
	}

	return strings.TrimSpace(text.String())
}

func extractTextFromNode(source []byte, node ast.Node, text *strings.Builder) {
	switch n := node.(type) {
	case *ast.Text:
		text.Write(n.Segment.Value(source))
	case *ast.String:
		text.Write(n.Value)
	default:
		// For other node types (like links, emphasis, etc.), extract text from children
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			extractTextFromNode(source, child, text)
		}
	}
}

func (n *NoteView) extractString(key string) (*string, error) {
	description, ok := n.RawMeta[key]
	if !ok {
		return nil, nil
	}

	str, ok := description.(string)
	if !ok {
		return nil, fmt.Errorf("invalid %s type: %T", key, description)
	}

	return &str, nil
}

func (n *NoteView) ExtractTitle() string {
	title, ok := n.RawMeta["title"]
	if ok {
		str, sOk := title.(string)
		if sOk {
			return str
		}
	}

	// nodeCount := 0
	// docTitle := ""
	//
	// find the first heading in .Ast
	// Need to remove the heading node before rendering
	// ast.Walk(p.Ast, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
	// 	nodeCount++
	//
	// 	if nodeCount > 5 {
	// 		return ast.WalkStop, nil
	// 	}
	//
	// 	if n.Kind() == ast.KindHeading {
	// 		heading := n.(*ast.Heading)
	//
	// 		if heading.Level != 1 {
	// 			return ast.WalkContinue, nil
	// 		}
	//
	// 		docTitle = string(heading.Text(p.Content))
	// 		return ast.WalkStop, nil
	// 	}
	//
	// 	return ast.WalkContinue, nil
	// })
	//
	// if docTitle != "" {
	// 	return docTitle
	// }

	return filepath.Base(n.Path[:len(n.Path)-len(".md")])
}

func NewNoteViews() *NoteViews {
	return &NoteViews{
		Map:            make(map[string]*NoteView),
		PathMap:        make(map[string]*NoteView),
		Subgraphs:      make(map[string]*NoteSubgraph),
		RouteMap:       make(map[string]map[string]*NoteView),
		DomainSitemaps: make(map[string][]byte),
	}
}

func (nv *NoteViews) Copy() *NoteViews {
	res := *nv
	return &res
}

func (nv *NoteViews) ResolveURL(note *NoteView) string {
	return note.Permalink
}

func (nv *NoteViews) ExtractNoteList() {
	nv.List = make([]*NoteView, 0, len(nv.Map))

	keys := make([]string, 0, len(nv.Map))

	// the Map can contains different paths for same note,
	// so we need to extract unique notes by ids.
	ids := make(map[int64]struct{})

	for k := range nv.Map {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		_, exists := ids[nv.Map[k].PathID]
		if exists {
			continue
		}

		ids[nv.Map[k].PathID] = struct{}{}

		nv.List = append(nv.List, nv.Map[k])
	}
}

func (nv *NoteViews) ExtractSubgraphs() {
	for _, page := range nv.Map {
		for _, ps := range page.SubgraphNames {
			_, ok := nv.Subgraphs[ps]
			if !ok {
				nv.Subgraphs[ps] = &NoteSubgraph{
					Name: ps,
				}
			}

			page.Subgraphs[ps] = nv.Subgraphs[ps]
		}
	}

	for name, subgraph := range nv.Subgraphs {
		sidebarPath := fmt.Sprintf("/_sidebar_%s", name)
		sidebar, ok := nv.Map[sidebarPath]
		if ok {
			subgraph.Sidebar = sidebar
		}

		homePathVariants := []string{
			fmt.Sprintf("/_index_%s", name),
			fmt.Sprintf("/_home_%s", name),
			fmt.Sprintf("/%s", name),
		}

		for _, homePath := range homePathVariants {
			home, homeOk := nv.Map[homePath]
			if homeOk {
				subgraph.Home = home
				break
			}
		}
	}
}

func (nvs *NoteViews) HomePages(note *NoteView) []*NoteView {
	var res []*NoteView

	for _, ps := range note.SubgraphNames {
		subgraph, ok := nvs.Subgraphs[ps]
		if !ok {
			continue
		}

		if subgraph.Home != nil {
			res = append(res, subgraph.Home)
		}
	}

	return res
}

// Sidebars returns the sidebar for a given path.
// Looking path: note meta "sidebar" path to md, "_sidebar_<subgraph>.md", "_sidebar.md".
func (nvs *NoteViews) Sidebars(note *NoteView) []*NoteView {
	sidebarI, sidebarOk := note.RawMeta["sidebar"]
	if sidebarOk {
		switch s := sidebarI.(type) {
		case string:
			noteSidebar, ok := nvs.Map[s]
			if ok {
				return []*NoteView{noteSidebar}
			}
		case bool:
			if !s {
				return nil
			}
		}
	}

	var res []*NoteView

	for _, ps := range note.SubgraphNames {
		subgraph, ok := nvs.Subgraphs[ps]
		if !ok {
			continue
		}

		if subgraph.Sidebar != nil {
			res = append(res, subgraph.Sidebar)
		}
	}

	// without subgraph sidebars then try to use the default sidebar
	if len(res) == 0 {
		sidebar, ok := nvs.Map["/_sidebar"]
		if ok {
			return append(res, sidebar)
		}
	}

	return res
}

// func (nv NoteViews) Subgraphs() ([]string, error) {
// 	subgraphs := make(map[string]struct{})
//
// 	for _, page := range nv.Map {
// 		for _, ps := range page.Subgraphs {
// 			subgraphs[ps] = struct{}{}
// 		}
// 	}
//
// 	res := make([]string, 0, len(subgraphs))
//
// 	for k := range subgraphs {
// 		res = append(res, k)
// 	}
//
// 	return res, nil
// }

func extractSubgraphs(target map[string]struct{}, val interface{}) error {
	switch val := val.(type) {
	case string:
		target[val] = struct{}{}
	case []interface{}:
		for _, v := range val {
			if vStr, ok := v.(string); ok {
				target[vStr] = struct{}{}
			} else {
				return fmt.Errorf("invalid subgraph type: %T", v)
			}
		}
	case nil:
		return nil
	default:
		return fmt.Errorf("invalid subgraph type: %T", val)
	}

	return nil
}

func (nv NoteViews) VisibleList() []*NoteView {
	views := []*NoteView{}

	for _, note := range nv.List {
		if strings.Contains(note.Permalink, "/_") {
			continue
		}

		views = append(views, note)
	}

	return views
}

func (nv NoteViews) IDMap() map[int64]*NoteView {
	idMap := make(map[int64]*NoteView, len(nv.Map))

	for _, page := range nv.Map {
		idMap[page.PathID] = page
	}

	return idMap
}

func (nv NoteViews) GetByPathID(id int64) *NoteView {
	for _, note := range nv.Map {
		if note.PathID == id {
			return note
		}
	}

	return nil
}

func (nv NoteViews) GetByVersionID(id int64) *NoteView {
	for _, note := range nv.Map {
		if note.VersionID == id {
			return note
		}
	}

	return nil
}

func (nv NoteViews) GetByPath(v string) *NoteView {
	note, ok := nv.Map[v]
	if !ok {
		return nil
	}

	return note
}

func (nv *NoteViews) Warnings() map[string][]NoteWarning {
	res := make(map[string][]NoteWarning)

	for _, note := range nv.List {
		if len(note.Warnings) == 0 {
			continue
		}

		res[note.Permalink] = append(res[note.Permalink], note.Warnings...)
	}

	return res
}

func (nv *NoteViews) RegisterNote(note *NoteView) {
	nv.Map[note.Permalink] = note
	nv.Map[note.PermalinkOriginal] = note
	nv.PathMap[note.Path] = note

	if note.IsIndex {
		if note.Permalink == "/" {
			nv.Map["/index"] = note
		} else {
			nv.Map[note.Permalink+"/index"] = note
		}
	}

	nv.RegisterNoteRoutes(note)
}

// GetByRoute looks up a note by host and path in RouteMap.
func (nv *NoteViews) GetByRoute(host, path string) *NoteView {
	routes, ok := nv.RouteMap[host]
	if !ok {
		return nil
	}
	return routes[path]
}

// RegisterNoteRoutes populates RouteMap for a note.
// Nil-safe: initializes RouteMap if needed.
func (nv *NoteViews) RegisterNoteRoutes(note *NoteView) {
	if len(note.Routes) == 0 {
		return
	}
	if nv.RouteMap == nil {
		nv.RouteMap = make(map[string]map[string]*NoteView)
	}
	for _, r := range note.Routes {
		if nv.RouteMap[r.Host] == nil {
			nv.RouteMap[r.Host] = make(map[string]*NoteView)
		}
		// Empty Path means "use note's own Permalink" (set when user writes "foo.com" without path).
		path := r.Path
		if path == "" {
			path = note.Permalink
		}
		nv.RouteMap[r.Host][path] = note
	}
}

// CustomDomains returns all non-empty hosts declared in RouteMap.
func (nv *NoteViews) CustomDomains() []string {
	var domains []string
	for host := range nv.RouteMap {
		if host != "" {
			domains = append(domains, host)
		}
	}
	return domains
}

// countWordsForReadingTime counts words in a single pass, skipping code blocks.
// This is optimized for reading time calculation - exact accuracy is not required.
// No allocations, no regex - just a simple state machine.
func countWordsForReadingTime(content string) int {
	wordCount := 0
	inWord := false
	inCodeBlock := false
	backtickCount := 0

	for i := range len(content) {
		c := content[i]

		// Track code block fences (```)
		if c == '`' {
			backtickCount++
			if backtickCount == 3 {
				inCodeBlock = !inCodeBlock
				backtickCount = 0
			}
			inWord = false
			continue
		}
		backtickCount = 0

		// Skip content inside code blocks
		if inCodeBlock {
			continue
		}

		// Check if character is a word character (letter or digit)
		isWordChar := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c >= 0x80 // 0x80+ for UTF-8 multibyte

		if isWordChar {
			if !inWord {
				wordCount++
				inWord = true
			}
		} else {
			inWord = false
		}
	}

	return wordCount
}
