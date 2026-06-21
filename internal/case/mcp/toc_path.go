package mcp

import (
	"bytes"
	"regexp"
	"strings"
	"unicode"

	"trip2g/internal/model"

	"golang.org/x/net/html"
	"golang.org/x/text/unicode/norm"
)

// TOCItem represents one heading in a document's table of contents.
// Path is the full breadcrumb from root to this heading — use it as toc_path
// in note_html to open exactly this section, even when titles repeat.
type TOCItem struct {
	Title string   `json:"title"`
	Level int      `json:"level"`
	Path  []string `json:"path"`
}

// buildNoteTOC builds a flat TOC list with full hierarchical paths from NoteViewHeadings.
// Handles repeated heading names by including the full ancestor chain in Path.
func buildNoteTOC(headings model.NoteViewHeadings) []TOCItem {
	if len(headings) == 0 {
		return nil
	}

	result := make([]TOCItem, 0, len(headings))
	titleStack := make([]string, 0, 6)
	levelStack := make([]int, 0, 6)

	for _, h := range headings {
		// Pop ancestors of equal or deeper level.
		for len(levelStack) > 0 && levelStack[len(levelStack)-1] >= h.Level {
			titleStack = titleStack[:len(titleStack)-1]
			levelStack = levelStack[:len(levelStack)-1]
		}
		path := append(append([]string(nil), titleStack...), h.Text)
		result = append(result, TOCItem{
			Title: h.Text,
			Level: h.Level,
			Path:  path,
		})
		titleStack = append(titleStack, h.Text)
		levelStack = append(levelStack, h.Level)
	}

	return result
}

// tocPathForSnippet returns a best-effort heading breadcrumb (outermost → innermost)
// for the data-header section that contains the given snippet. Snippet may contain
// <mark> tags; they are stripped before matching.
//
// To avoid false negatives when the snippet spans multiple sections, only the
// context immediately around the first <mark> block is used as the search target.
//
// chunkContent is the optional chunk body (format: "{breadcrumb}\n\n{body}") used
// as a fallback when fuzzy HTML matching fails. Pass "" to skip the chunk fallback.
//
// The function returns nil only when the note has no data-header sections at all
// (or the snippet is empty). In that case the caller should read the whole note.
// Otherwise it falls back first to the chunk breadcrumb, then to the first section.
func tocPathForSnippet(noteHTML, snippet, chunkContent string) []string {
	if noteHTML == "" || snippet == "" {
		return nil
	}

	raw := normalizeForMatch(markedContext(snippet))
	if len([]rune(raw)) < 4 {
		// snippet too short for reliable matching — try chunk fallback first
		if path := tocPathFromChunkContent(noteHTML, chunkContent); path != nil {
			return path
		}
		return firstSectionPath(noteHTML)
	}

	doc, err := html.Parse(strings.NewReader(noteHTML))
	if err != nil {
		return firstSectionPath(noteHTML)
	}

	path, _ := findDeepestSection(doc, raw, nil)
	if path != nil {
		return path
	}

	// Fuzzy match failed — try chunk breadcrumb fallback.
	if p := tocPathFromChunkContent(noteHTML, chunkContent); p != nil {
		return p
	}

	// Final fallback: first section or root.
	return firstSectionPath(noteHTML)
}

// findDeepestSection walks the HTML tree and returns the path to the innermost
// data-header div whose text content contains target.
func findDeepestSection(n *html.Node, target string, currentPath []string) ([]string, bool) {
	if n.Type == html.ElementNode && n.Data == "div" {
		if header := htmlNodeAttr(n, "data-header"); header != "" {
			return matchHeaderDiv(n, header, target, currentPath)
		}
	}
	return walkHTMLChildren(n, target, currentPath)
}

func matchHeaderDiv(n *html.Node, header, target string, currentPath []string) ([]string, bool) {
	sectionText := normalizeForMatch(htmlNodeText(n))
	if !strings.Contains(sectionText, target) {
		return nil, false
	}
	newPath := append(append([]string(nil), currentPath...), header)
	// Prefer the deepest (most specific) match from children.
	if path, ok := walkHTMLChildren(n, target, newPath); ok {
		return path, true
	}
	return newPath, true
}

func walkHTMLChildren(n *html.Node, target string, currentPath []string) ([]string, bool) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if path, ok := findDeepestSection(c, target, currentPath); ok {
			return path, true
		}
	}
	return nil, false
}

// sectionHTMLByTocPath returns the innerHTML of the data-header section identified
// by path (array of heading titles from outermost to innermost).
// Returns "" if not found.
func sectionHTMLByTocPath(noteHTML string, path []string) string {
	if noteHTML == "" || len(path) == 0 {
		return ""
	}

	doc, err := html.Parse(strings.NewReader(noteHTML))
	if err != nil {
		return ""
	}

	node := navigateSectionPath(doc, path)
	if node == nil {
		return ""
	}

	// Return innerHTML: children of the section div (includes <hN> and content).
	var buf bytes.Buffer
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		_ = html.Render(&buf, c)
	}
	return buf.String()
}

// navigateSectionPath follows the path array step by step, at each step finding
// the first data-header div with the matching title inside the current subtree.
func navigateSectionPath(root *html.Node, path []string) *html.Node {
	current := root
	for _, title := range path {
		current = findFirstSectionWithTitle(current, title)
		if current == nil {
			return nil
		}
	}
	return current
}

func findFirstSectionWithTitle(root *html.Node, title string) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" {
			if htmlNodeAttr(n, "data-header") == title {
				found = n
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return found
}

// htmlNodeText returns the plain text content of an HTML node.
func htmlNodeText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

// htmlNodeAttr returns the value of an attribute on an HTML element node.
func htmlNodeAttr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

// markedContext returns the text immediately surrounding the first <mark> block
// in the snippet (up to 120 chars before and 120 chars after the marked region).
// This avoids false negatives when a snippet spans multiple sections.
// Returns the original snippet when no <mark> tags are found.
func markedContext(snippet string) string {
	start := strings.Index(snippet, "<mark>")
	end := strings.LastIndex(snippet, "</mark>")
	if start == -1 || end == -1 {
		return snippet
	}
	end += len("</mark>")

	const window = 120
	lo := start - window
	if lo < 0 {
		lo = 0
	}
	hi := end + window
	if hi > len(snippet) {
		hi = len(snippet)
	}
	return snippet[lo:hi]
}

// htmlPlainText strips all HTML tags from s and returns plain text.
func htmlPlainText(s string) string {
	if !strings.ContainsRune(s, '<') {
		return s
	}
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return s
	}
	return htmlNodeText(doc)
}

// isSpaceVariant reports whether r is a Unicode space variant that should be
// mapped to a regular ASCII space.
func isSpaceVariant(r rune) bool {
	// U+00A0 NO-BREAK SPACE, U+2002-U+200A (various typographic spaces),
	// U+202F NARROW NO-BREAK SPACE, U+205F MEDIUM MATHEMATICAL SPACE,
	// U+3000 IDEOGRAPHIC SPACE.
	return r == 0x00A0 ||
		(r >= 0x2002 && r <= 0x200A) ||
		r == 0x202F ||
		r == 0x205F ||
		r == 0x3000
}

// isZeroWidth reports whether r is a zero-width character that should be dropped.
func isZeroWidth(r rune) bool {
	// U+200B ZERO WIDTH SPACE, U+200C ZERO WIDTH NON-JOINER,
	// U+200D ZERO WIDTH JOINER, U+FEFF BOM / ZERO WIDTH NO-BREAK SPACE,
	// U+00AD SOFT HYPHEN.
	return r == 0x200B || r == 0x200C || r == 0x200D || r == 0xFEFF || r == 0x00AD
}

// normalizeForMatch normalizes a string for fuzzy section matching.
// It is applied to BOTH the snippet target and the section's text so the
// comparison is free of encoding/whitespace/unicode artefacts.
//
// Pipeline (in order):
//  1. Parse as HTML and concat text nodes — same extractor for both sides.
//  2. html.UnescapeString for any remaining entity references.
//  3. Replace non-breaking / zero-width space characters with a regular space.
//  4. NFC normalize via golang.org/x/text/unicode/norm.
//  5. Fold typographic quotes and dashes to their ASCII equivalents.
//  6. strings.ToLower.
//  7. Collapse whitespace (strings.Fields + Join).
func normalizeForMatch(s string) string {
	// Step 1: extract plain text via HTML parser so entity/tag handling is identical
	// on both the snippet side (which may contain <mark> tags) and the goldmark HTML
	// side (which may contain <strong>, <em>, etc.).
	s = htmlPlainText(s)

	// Step 2: unescape any remaining HTML entities (e.g. &amp; that survived as text).
	s = html.UnescapeString(s)

	// Step 3: replace Unicode space variants and drop zero-width characters.
	// Uses helper functions to avoid embedding problematic Unicode literals directly.
	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		switch {
		case isZeroWidth(r):
			// drop
		case isSpaceVariant(r):
			sb.WriteByte(' ')
		default:
			sb.WriteRune(r)
		}
	}
	s = sb.String()

	// Step 4: NFC normalize.
	s = norm.NFC.String(s)

	// Step 5: fold typographic punctuation conservatively.
	s = foldTypographic(s)

	// Step 6: lowercase.
	s = strings.ToLower(s)

	// Step 7: collapse whitespace.
	s = strings.Join(strings.Fields(s), " ")

	return s
}

// foldTypographic maps common typographic variants to their ASCII equivalents.
// We only fold when the visual meaning is the same (quotes, dashes) to avoid
// false positives for languages where these are semantically distinct.
func foldTypographic(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		switch r {
		// Smart double quotes → "
		case '“', '”', '«', '»':
			sb.WriteByte('"')
		// Smart single quotes / apostrophes → '
		case '‘', '’', '‚', '‛':
			sb.WriteByte('\'')
		// Dashes → -
		case '–', '—', '−':
			sb.WriteByte('-')
		// Ellipsis → ...
		case '…':
			sb.WriteString("...")
		default:
			if unicode.IsPrint(r) || unicode.IsSpace(r) {
				sb.WriteRune(r)
			}
		}
	}
	return sb.String()
}

// mdInlineStripRe strips the most common Markdown inline syntax from a heading
// text extracted by parseHeading (which preserves raw markdown).
// This makes breadcrumb segments comparable to data-header values (which use
// goldmark's plain-text extraction).
var mdInlineStripRe = regexp.MustCompile(
	// Images: ![alt](url) or ![alt][ref]
	`!\[([^\]]*)\](?:\([^)]*\)|\[[^\]]*\])` +
		// Links: [text](url) or [text][ref]
		`|\[([^\]]*)\](?:\([^)]*\)|\[[^\]]*\])` +
		// Bold+italic: ***text*** or ___text___
		`|\*{3}([^*]+)\*{3}` +
		`|_{3}([^_]+)_{3}` +
		// Bold: **text** or __text__
		`|\*{2}([^*]+)\*{2}` +
		`|_{2}([^_]+)_{2}` +
		// Italic: *text* or _text_
		`|\*([^*]+)\*` +
		`|_([^_]+)_` +
		// Inline code double-backtick: ``text``
		"|``([^`]+)``" +
		// Inline code single-backtick: `text`
		"|`([^`]+)`",
)

// stripMarkdownInline removes inline markdown syntax from a heading text that
// was extracted by mdchunk.parseHeading (raw markdown). The result should match
// the data-header value produced by goldmark's headingPlainText for the same
// heading.
func stripMarkdownInline(s string) string {
	return mdInlineStripRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := mdInlineStripRe.FindStringSubmatch(m)
		// Return the first non-empty capture group (the inner text).
		for _, g := range sub[1:] {
			if g != "" {
				return g
			}
		}
		return ""
	})
}

// tocPathFromChunkContent derives a toc_path from a chunk's breadcrumb line.
// chunkContent has the format "{title} > {h1} > {h2}...\n\n{body}".
// The first segment is the note title (not a section); subsequent segments are
// heading texts as produced by parseHeading (raw markdown inline).
//
// The derived path segments are stripped of markdown inline syntax so they
// match the data-header attribute values (goldmark plain-text extraction).
// If noteHTML is provided, each segment is verified against the rendered HTML;
// if any segment does not match a data-header, the function returns nil.
func tocPathFromChunkContent(noteHTML, chunkContent string) []string {
	if chunkContent == "" {
		return nil
	}
	// Split on first \n\n to isolate the breadcrumb line.
	breadcrumb := chunkContent
	if idx := strings.Index(chunkContent, "\n\n"); idx >= 0 {
		breadcrumb = chunkContent[:idx]
	}

	parts := strings.Split(breadcrumb, " > ")
	if len(parts) < 2 {
		// Only title, no headings — content is before the first heading (intro).
		return nil
	}

	// Drop the title (first part); remaining are heading breadcrumb segments.
	headingParts := parts[1:]

	// Strip markdown inline syntax from each segment.
	path := make([]string, 0, len(headingParts))
	for _, p := range headingParts {
		clean := strings.TrimSpace(stripMarkdownInline(p))
		if clean == "" {
			return nil // malformed breadcrumb
		}
		path = append(path, clean)
	}

	// Verify path round-trips through the noteHTML if provided.
	if noteHTML != "" {
		if !pathExistsInHTML(noteHTML, path) {
			return nil
		}
	}

	return path
}

// pathExistsInHTML verifies that the given toc_path resolves to a section in
// the noteHTML by attempting to navigate to it.
func pathExistsInHTML(noteHTML string, path []string) bool {
	doc, err := html.Parse(strings.NewReader(noteHTML))
	if err != nil {
		return false
	}
	return navigateSectionPath(doc, path) != nil
}

// firstSectionPath returns the path to the first data-header section in the
// note HTML. Used as a final fallback when all other methods fail.
// Returns nil only when the note has no sections at all.
func firstSectionPath(noteHTML string) []string {
	if noteHTML == "" {
		return nil
	}
	doc, err := html.Parse(strings.NewReader(noteHTML))
	if err != nil {
		return nil
	}
	var firstHeader string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if firstHeader != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" {
			if h := htmlNodeAttr(n, "data-header"); h != "" {
				firstHeader = h
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if firstHeader == "" {
		return nil
	}
	return []string{firstHeader}
}

// tocChildren returns the direct children of the TOC node addressed by parentPath.
// An empty parentPath returns the top-level sections. Each child reports whether it
// has children of its own, so an agent can walk the tree level by level (expand)
// without loading the note's content or its full flat TOC.
func tocChildren(headings model.NoteViewHeadings, parentPath []string) []TOCNode {
	all := buildNoteTOC(headings)
	out := make([]TOCNode, 0)
	for _, item := range all {
		if len(item.Path) != len(parentPath)+1 || !tocPathHasPrefix(item.Path, parentPath) {
			continue
		}
		out = append(out, TOCNode{
			Title:       item.Title,
			Level:       item.Level,
			Path:        item.Path,
			HasChildren: tocHasDirectChild(all, item.Path),
		})
	}
	return out
}

// tocPathHasPrefix reports whether path starts with prefix.
func tocPathHasPrefix(path, prefix []string) bool {
	if len(path) < len(prefix) {
		return false
	}
	for i := range prefix {
		if path[i] != prefix[i] {
			return false
		}
	}
	return true
}

// tocHasDirectChild reports whether any node in all is a direct child of path.
func tocHasDirectChild(all []TOCItem, path []string) bool {
	for _, item := range all {
		if len(item.Path) == len(path)+1 && tocPathHasPrefix(item.Path, path) {
			return true
		}
	}
	return false
}
