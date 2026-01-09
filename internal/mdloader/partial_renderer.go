package mdloader

import (
	"bytes"
	"trip2g/internal/model"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
)

type PartialRenderer struct {
	md      goldmark.Markdown
	ast     ast.Node
	content []byte
}

func (pr *PartialRenderer) SetContent(astNode ast.Node, content []byte) {
	pr.ast = astNode
	pr.content = content
}

func (pr *PartialRenderer) Sections(level int) []model.NoteViewSection {
	if pr.ast == nil || pr.content == nil {
		return nil
	}

	var blocks []model.NoteViewSection
	var allNodes []ast.Node

	// First pass: collect all top-level nodes
	for child := pr.ast.FirstChild(); child != nil; child = child.NextSibling() {
		allNodes = append(allNodes, child)
	}

	var currentBlock *model.NoteViewSection
	var contentStart = -1

	for i, node := range allNodes {
		if heading, ok := node.(*ast.Heading); ok {
			// If we find a heading of the target level
			if heading.Level == level {
				// Save previous block if exists
				if currentBlock != nil {
					currentBlock.ContentHTML = pr.renderNodeRange(allNodes, contentStart, i)
					blocks = append(blocks, *currentBlock)
				}

				// Start new block
				currentBlock = &model.NoteViewSection{
					TitleHTML: pr.renderHeading(heading),
				}
				contentStart = i + 1
				continue
			}

			// If we find a heading of same or higher level, finish current block
			if currentBlock != nil && heading.Level <= level {
				currentBlock.ContentHTML = pr.renderNodeRange(allNodes, contentStart, i)
				blocks = append(blocks, *currentBlock)
				currentBlock = nil
				contentStart = -1
			}
		}
	}

	// Add the last block if exists
	if currentBlock != nil {
		currentBlock.ContentHTML = pr.renderNodeRange(allNodes, contentStart, len(allNodes))
		blocks = append(blocks, *currentBlock)
	}

	return blocks
}

// HeadingBlocks is deprecated, use Sections instead.
func (pr *PartialRenderer) HeadingBlocks(level int) []model.NoteViewSection {
	return pr.Sections(level)
}

// Section finds a section by its heading title.
// Returns nil if no heading with the given title is found.
// The title is matched against the plain text content of the heading.
func (pr *PartialRenderer) Section(title string) *model.NoteViewSection {
	if pr.ast == nil || pr.content == nil {
		return nil
	}

	var allNodes []ast.Node

	// Collect all top-level nodes
	for child := pr.ast.FirstChild(); child != nil; child = child.NextSibling() {
		allNodes = append(allNodes, child)
	}

	// Find the heading with matching title
	for i, node := range allNodes {
		heading, ok := node.(*ast.Heading)
		if !ok {
			continue
		}

		// Get plain text title for comparison
		headingText := extractHeadingText(pr.content, heading)
		if headingText != title {
			continue
		}

		// Found the heading, now collect content until next heading of same or higher level
		contentEnd := len(allNodes)
		for j := i + 1; j < len(allNodes); j++ {
			nextHeading, isHeading := allNodes[j].(*ast.Heading)
			if isHeading && nextHeading.Level <= heading.Level {
				contentEnd = j
				break
			}
		}

		return &model.NoteViewSection{
			TitleHTML:   pr.renderHeading(heading),
			ContentHTML: pr.renderNodeRange(allNodes, i+1, contentEnd),
		}
	}

	return nil
}

// extractHeadingText extracts plain text from a heading node.
func extractHeadingText(content []byte, heading *ast.Heading) string {
	var text bytes.Buffer

	for child := heading.FirstChild(); child != nil; child = child.NextSibling() {
		extractTextFromNodeRecursive(content, child, &text)
	}

	return text.String()
}

// extractTextFromNodeRecursive extracts plain text from a node and its children.
func extractTextFromNodeRecursive(content []byte, node ast.Node, buf *bytes.Buffer) {
	switch n := node.(type) {
	case *ast.Text:
		buf.Write(n.Segment.Value(content))
	case *ast.String:
		buf.Write(n.Value)
	default:
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			extractTextFromNodeRecursive(content, child, buf)
		}
	}
}

func (pr *PartialRenderer) Introduce() model.NoteViewSection {
	if pr.ast == nil || pr.content == nil {
		return model.NoteViewSection{}
	}

	var allNodes []ast.Node

	// Collect all top-level nodes
	for child := pr.ast.FirstChild(); child != nil; child = child.NextSibling() {
		allNodes = append(allNodes, child)
	}

	// Find the first heading of any level
	var firstHeadingIndex = -1
	for i, node := range allNodes {
		if _, ok := node.(*ast.Heading); ok {
			firstHeadingIndex = i
			break
		}
	}

	// If no headings found, return all content
	if firstHeadingIndex == -1 {
		return model.NoteViewSection{
			TitleHTML:   "",
			ContentHTML: pr.renderNodeRange(allNodes, 0, len(allNodes)),
		}
	}

	// Return content before the first heading
	return model.NoteViewSection{
		TitleHTML:   "",
		ContentHTML: pr.renderNodeRange(allNodes, 0, firstHeadingIndex),
	}
}

func (pr *PartialRenderer) renderHeading(heading *ast.Heading) string {
	var buf bytes.Buffer

	// Render only the children of the heading (the content inside)
	for child := heading.FirstChild(); child != nil; child = child.NextSibling() {
		err := pr.md.Renderer().Render(&buf, pr.content, child)
		if err != nil {
			continue // Skip nodes that can't be rendered
		}
	}

	return buf.String()
}

func (pr *PartialRenderer) renderNodeRange(allNodes []ast.Node, start, end int) string {
	if start < 0 || start >= len(allNodes) || end <= start {
		return ""
	}

	var buf bytes.Buffer

	for i := start; i < end && i < len(allNodes); i++ {
		node := allNodes[i]
		err := pr.md.Renderer().Render(&buf, pr.content, node)
		if err != nil {
			continue // Skip nodes that can't be rendered
		}
	}

	return buf.String()
}
