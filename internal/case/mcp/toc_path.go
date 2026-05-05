package mcp

import (
	"bytes"
	"strings"

	"trip2g/internal/model"

	"golang.org/x/net/html"
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

// tocPathForSnippet returns the heading breadcrumb (outermost → innermost) of the
// data-header section that contains the given snippet. Snippet may contain <mark>
// tags; they are stripped before matching. Returns nil if not found.
//
// To avoid false negatives when the snippet spans multiple sections, only the
// context immediately around the first <mark> block is used as the search target.
func tocPathForSnippet(noteHTML, snippet string) []string {
	if noteHTML == "" || snippet == "" {
		return nil
	}

	raw := strings.Join(strings.Fields(strings.ToLower(htmlPlainText(markedContext(snippet)))), " ")
	if len([]rune(raw)) < 4 {
		return nil
	}
	target := raw

	doc, err := html.Parse(strings.NewReader(noteHTML))
	if err != nil {
		return nil
	}

	path, _ := findDeepestSection(doc, target, nil)
	return path
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
	sectionText := strings.Join(strings.Fields(strings.ToLower(htmlNodeText(n))), " ")
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
