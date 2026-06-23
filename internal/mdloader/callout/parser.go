package callout

import (
	"strings"
	"unicode"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// transformer walks the parsed document and converts Obsidian callout
// blockquotes (those whose first line is `[!type] ...`) into Callout nodes.
//
// The body keeps the original blockquote children verbatim — the marker line is
// stripped in place from the first paragraph. This is important: every body
// node's text segments still reference the original source buffer, so the
// normal render pass (with its wikilink/image resolvers) renders them
// correctly.
type transformer struct{}

// calloutInfo is the parsed result of a `[!type]...` marker line.
type calloutInfo struct {
	Type     string
	Title    string
	Foldable bool
	Expanded bool
}

// blockquoteReplacement defers a blockquote→callout swap until after the walk,
// because mutating the AST mid-walk breaks sibling traversal.
type blockquoteReplacement struct {
	bq   *ast.Blockquote
	node *Node
}

// Transform implements parser.ASTTransformer.
func (t *transformer) Transform(node *ast.Document, reader text.Reader, _ parser.Context) {
	source := reader.Source()

	var replacements []blockquoteReplacement

	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Kind() != ast.KindBlockquote {
			return ast.WalkContinue, nil
		}

		bq, ok := n.(*ast.Blockquote)
		if !ok {
			return ast.WalkContinue, nil
		}

		cn := buildCallout(bq, source)
		if cn == nil {
			return ast.WalkContinue, nil
		}

		replacements = append(replacements, blockquoteReplacement{bq: bq, node: cn})
		// Skip descent: the blockquote is being replaced and its body has
		// already been moved into the callout node.
		return ast.WalkSkipChildren, nil
	})

	for _, r := range replacements {
		parent := r.bq.Parent()
		if parent != nil {
			parent.ReplaceChild(parent, r.bq, r.node)
		}
	}
}

// buildCallout inspects a blockquote and, if its first paragraph starts with a
// `[!type]` marker, returns a populated Callout node holding the body content.
// Returns nil for plain blockquotes.
func buildCallout(bq *ast.Blockquote, source []byte) *Node {
	first := bq.FirstChild()
	if first == nil || first.Kind() != ast.KindParagraph {
		return nil
	}

	para, ok := first.(*ast.Paragraph)
	if !ok {
		return nil
	}

	firstLine := paragraphFirstLine(para, source)
	info, found := parseMarker(firstLine)
	if !found {
		return nil
	}

	cn := &Node{
		CalloutType: info.Type,
		Title:       info.Title,
		Foldable:    info.Foldable,
		Expanded:    info.Expanded,
	}

	// Strip the marker line from the first paragraph in place, keeping any
	// body that shared the line (e.g. `> [!note]\n> body`).
	stripMarkerLine(para)

	// Move all blockquote children into the callout body, dropping the marker
	// paragraph if stripping left it empty.
	for child := bq.FirstChild(); child != nil; {
		next := child.NextSibling()
		bq.RemoveChild(bq, child)
		if child == first && !child.HasChildren() {
			child = next
			continue
		}
		cn.AppendChild(cn, child)
		child = next
	}

	return cn
}

// stripMarkerLine removes the marker line (`[!type]...`) from the first
// paragraph of a callout blockquote, preserving any body that followed on
// subsequent lines of the same paragraph.
//
// The marker occupies the paragraph's first source line and its leading inline
// children up to (and including) the node carrying the soft line break. Those
// inline children are removed, and the first line segment is dropped, leaving
// the body content intact with valid source positions.
func stripMarkerLine(para *ast.Paragraph) {
	// Remove the inline children belonging to the marker line: everything up
	// to and including the first child whose SoftLineBreak is set. If no child
	// has a soft break, the whole paragraph is just the marker line — remove
	// all inline children.
	for child := para.FirstChild(); child != nil; {
		next := child.NextSibling()
		para.RemoveChild(para, child)
		if tn, ok := child.(*ast.Text); ok && tn.SoftLineBreak() {
			break
		}
		child = next
	}

	// Drop the marker source line from the paragraph's line segments.
	lines := para.Lines()
	if lines.Len() <= 1 {
		lines.Clear()
	} else {
		lines.SetSliced(1, lines.Len())
	}
}

// paragraphFirstLine returns the text of the paragraph's first source line.
func paragraphFirstLine(para *ast.Paragraph, source []byte) string {
	lines := para.Lines()
	if lines.Len() == 0 {
		return ""
	}
	seg := lines.At(0)
	return string(seg.Value(source))
}

// parseMarker parses a callout marker from the first line of a blockquote.
// Recognised forms:
//
//	[!type]
//	[!type]-            (foldable, collapsed)
//	[!type]+            (foldable, expanded)
//	[!type] Title
//	[!type]- Title
//	[!type]+ Title
//
// It returns the parsed info and whether a marker was found.
func parseMarker(line string) (calloutInfo, bool) {
	s := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(s, "[!") {
		return calloutInfo{}, false
	}

	end := strings.IndexByte(s, ']')
	if end < 0 {
		return calloutInfo{}, false
	}

	typ := s[2:end]
	if typ == "" || !isValidType(typ) {
		return calloutInfo{}, false
	}

	info := calloutInfo{Type: strings.ToLower(typ)}

	rest := s[end+1:]
	if len(rest) > 0 {
		switch rest[0] {
		case '-':
			info.Foldable = true
			info.Expanded = false
			rest = rest[1:]
		case '+':
			info.Foldable = true
			info.Expanded = true
			rest = rest[1:]
		}
	}

	info.Title = strings.TrimSpace(rest)

	return info, true
}

// isValidType reports whether the callout type token contains only characters
// valid in an Obsidian callout identifier (letters, digits, '-', '_').
func isValidType(typ string) bool {
	for _, r := range typ {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}
