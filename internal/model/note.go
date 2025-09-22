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

type NoteView struct {
	Path  string
	Title string

	PathID    int64
	VersionID int64
	CreatedAt time.Time

	Content []byte
	HTML    template.HTML
	ast     ast.Node // hide from JSON

	// If the field `free_paragraphs` is present, then thin field will
	// includes the first rendered paragraphs equal to this number.
	FreeHTML template.HTML

	Permalink string
	IsIndex   bool

	PermalinkOriginal string

	Free     bool // without the paywall
	Redirect *string

	Description *string // meta description for SEO

	InLinks map[string]struct{}
	RawMeta map[string]interface{}

	ResolvedLinks map[string]string // local link to absolute link mapping

	SubgraphNames []string
	Subgraphs     map[string]*NoteSubgraph `json:"-"`

	Assets map[string]struct{}

	AssetReplaces map[string]string

	ReadingTime       int // in minutes
	ReadingComplexity int // 0 - easy, 1 - medium, 2 - hard

	Headings NoteViewHeadings // extracted from AST

	HeadingCount map[string]int // for id generation

	TOCDisplay int // TOCDisplayAuto, TOCDisplayShow, TOCDisplayHide - from meta

	EmbededClass string

	Warnings []NoteWarning

	FirstImage *string
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

	Subgraphs map[string]*NoteSubgraph `json:"-"`

	Version string
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
	reMultiUnderscore := regexp.MustCompile(`_+`)
	res = reMultiUnderscore.ReplaceAllString(res, "_")
	res = strings.Trim(res, "_")

	if s[0] == '_' {
		res = "_" + res
	}

	return res
}

func (n *NoteView) PreparePermalink() {
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

	return nil
}

var classRE = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

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

	// Calculate reading time based on content
	content := string(n.Content)

	// Remove markdown syntax for more accurate word count
	content = removeMarkdownSyntax(content)

	// Count words
	wordCount := countWords(content)

	// Average reading speed: 200 words per minute
	// Round up to nearest minute
	readingTime := (wordCount + 199) / 200

	// Minimum reading time is 1 minute
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
		Map: make(map[string]*NoteView),

		PathMap: make(map[string]*NoteView),

		Subgraphs: make(map[string]*NoteSubgraph),
	}
}

func (nv *NoteViews) Copy() *NoteViews {
	res := *nv
	return &res
}

func (nv *NoteViews) ResolveURL(note *NoteView, defaultVersion string) string {
	if defaultVersion == "" {
		defaultVersion = "live"
	}

	// TODO: extract this logic from here and internal/mdloader/link_resolver.go
	if len(nv.Version) > 0 && nv.Version != defaultVersion {
		// parse url and add ?version= to the end
		u, err := url.Parse(note.Permalink)
		if err != nil {
			return ""
		}

		query := u.Query()
		query.Set("version", nv.Version)
		u.RawQuery = query.Encode()

		return u.String()
	}

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
}

// removeMarkdownSyntax removes common markdown syntax for more accurate word counting.
func removeMarkdownSyntax(content string) string {
	// Remove code blocks (```code```)
	codeBlockRegex := regexp.MustCompile("```[\\s\\S]*?```")
	content = codeBlockRegex.ReplaceAllString(content, " ")

	// Remove inline code (`code`)
	inlineCodeRegex := regexp.MustCompile("`[^`]*`")
	content = inlineCodeRegex.ReplaceAllString(content, " ")

	// Remove headers (# ## ###)
	headerRegex := regexp.MustCompile(`^#{1,6}\s+`)
	content = headerRegex.ReplaceAllString(content, "")

	// Remove links but keep the text [text](url) -> text
	linkRegex := regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	content = linkRegex.ReplaceAllString(content, "$1")

	// Remove wikilinks but keep the text [[link|text]] -> text or [[link]] -> link
	wikilinkRegex := regexp.MustCompile(`\[\[([^|\]]*?)(?:\|([^\]]*?))?\]\]`)
	content = wikilinkRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := wikilinkRegex.FindStringSubmatch(match)
		if len(parts) > 2 && parts[2] != "" {
			return parts[2] // Use display text if available
		}
		return parts[1] // Use link target
	})

	// Remove bold/italic markers (**text** *text*)
	boldItalicRegex := regexp.MustCompile(`\*+([^*]+)\*+`)
	content = boldItalicRegex.ReplaceAllString(content, "$1")

	// Remove strikethrough (~~text~~)
	strikethroughRegex := regexp.MustCompile(`~~([^~]+)~~`)
	content = strikethroughRegex.ReplaceAllString(content, "$1")

	// Remove HTML tags
	htmlRegex := regexp.MustCompile(`<[^>]*>`)
	content = htmlRegex.ReplaceAllString(content, " ")

	// Remove list markers (- * + 1.)
	listRegex := regexp.MustCompile(`^\s*[-*+]\s+|^\s*\d+\.\s+`)
	content = listRegex.ReplaceAllString(content, "")

	// Remove blockquotes (>)
	blockquoteRegex := regexp.MustCompile(`^\s*>\s*`)
	content = blockquoteRegex.ReplaceAllString(content, "")

	return content
}

// countWords counts the number of words in the given text.
func countWords(content string) int {
	if content == "" {
		return 0
	}

	// Split by whitespace and count non-empty words
	words := strings.Fields(content)
	wordCount := 0

	for _, word := range words {
		// Remove punctuation from start and end
		word = strings.Trim(word, ".,!?;:\"'()[]{}/-_=+")
		if word != "" && len(word) > 0 {
			wordCount++
		}
	}

	return wordCount
}
