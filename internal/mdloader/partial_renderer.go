package mdloader

import (
	"bytes"
	"trip2g/internal/model"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
)

type PartialRenderer struct {
	md       goldmark.Markdown
	ast      ast.Node
	content  []byte
	resolver *myLinkResolver
	page     *model.NoteView
}

func (pr *PartialRenderer) SetContent(astNode ast.Node, content []byte) {
	pr.ast = astNode
	pr.content = content
}

// SetPage sets the page reference for correct asset resolution during rendering.
func (pr *PartialRenderer) SetPage(page *model.NoteView) {
	pr.page = page
}

// withCurrentPage temporarily sets resolver.currentPage to this page for rendering.
func (pr *PartialRenderer) withCurrentPage(fn func()) {
	if pr.resolver == nil || pr.page == nil {
		fn()
		return
	}

	prev := pr.resolver.currentPage
	pr.resolver.currentPage = pr.page
	fn()
	pr.resolver.currentPage = prev
}

func (pr *PartialRenderer) Sections(level int) []model.NoteViewSection {
	return pr.sectionsFromNodes(pr.collectTopLevelNodes(), level)
}

func (pr *PartialRenderer) collectTopLevelNodes() []ast.Node {
	if pr.ast == nil {
		return nil
	}

	var nodes []ast.Node
	for child := pr.ast.FirstChild(); child != nil; child = child.NextSibling() {
		nodes = append(nodes, child)
	}
	return nodes
}

func (pr *PartialRenderer) sectionsFromNodes(allNodes []ast.Node, level int) []model.NoteViewSection {
	if len(allNodes) == 0 || pr.content == nil {
		return nil
	}

	var blocks []model.NoteViewSection

	type sectionRange struct {
		title        string
		titleHTML    string
		contentStart int
		contentEnd   int
	}

	var ranges []sectionRange
	var currentRange *sectionRange

	for i, node := range allNodes {
		heading, ok := node.(*ast.Heading)
		if !ok {
			continue
		}

		// If we find a heading of the target level.
		if heading.Level == level {
			// Save previous range if exists.
			if currentRange != nil {
				currentRange.contentEnd = i
				ranges = append(ranges, *currentRange)
			}

			// Start new range.
			currentRange = &sectionRange{
				title:        extractHeadingText(pr.content, heading),
				titleHTML:    pr.renderHeading(heading),
				contentStart: i + 1,
			}
			continue
		}

		// If we find a heading of same or higher level, finish current range.
		if currentRange != nil && heading.Level <= level {
			currentRange.contentEnd = i
			ranges = append(ranges, *currentRange)
			currentRange = nil
		}
	}

	// Add the last range if exists.
	if currentRange != nil {
		currentRange.contentEnd = len(allNodes)
		ranges = append(ranges, *currentRange)
	}

	// Build sections from ranges.
	for _, r := range ranges {
		contentNodes := allNodes[r.contentStart:r.contentEnd]

		section := model.NoteViewSection{
			Title:       r.title,
			TitleHTML:   r.titleHTML,
			ContentHTML: pr.renderNodeRange(allNodes, r.contentStart, r.contentEnd),
		}

		// Capture nodes for nested Sections()/Section() calls.
		section.SectionsFunc = pr.makeSectionsFunc(contentNodes)
		section.SectionFunc = pr.makeSectionFunc(contentNodes)

		blocks = append(blocks, section)
	}

	return blocks
}

func (pr *PartialRenderer) makeSectionsFunc(nodes []ast.Node) func(int) []model.NoteViewSection {
	return func(level int) []model.NoteViewSection {
		return pr.sectionsFromNodes(nodes, level)
	}
}

func (pr *PartialRenderer) makeSectionFunc(nodes []ast.Node) func(string) *model.NoteViewSection {
	return func(title string) *model.NoteViewSection {
		return pr.sectionFromNodes(nodes, title)
	}
}

func (pr *PartialRenderer) sectionFromNodes(allNodes []ast.Node, title string) *model.NoteViewSection {
	if len(allNodes) == 0 || pr.content == nil {
		return nil
	}

	// Find the heading with matching title.
	for i, node := range allNodes {
		heading, ok := node.(*ast.Heading)
		if !ok {
			continue
		}

		headingText := extractHeadingText(pr.content, heading)
		if headingText != title {
			continue
		}

		// Found the heading, now collect content until next heading of same or higher level.
		contentEnd := len(allNodes)
		for j := i + 1; j < len(allNodes); j++ {
			nextHeading, isHeading := allNodes[j].(*ast.Heading)
			if isHeading && nextHeading.Level <= heading.Level {
				contentEnd = j
				break
			}
		}

		contentNodes := allNodes[i+1 : contentEnd]

		section := &model.NoteViewSection{
			Title:       headingText,
			TitleHTML:   pr.renderHeading(heading),
			ContentHTML: pr.renderNodeRange(allNodes, i+1, contentEnd),
		}

		section.SectionsFunc = pr.makeSectionsFunc(contentNodes)
		section.SectionFunc = pr.makeSectionFunc(contentNodes)

		return section
	}

	return nil
}

// HeadingBlocks is deprecated, use Sections instead.
func (pr *PartialRenderer) HeadingBlocks(level int) []model.NoteViewSection {
	return pr.Sections(level)
}

// Section finds a section by its heading title.
// Returns nil if no heading with the given title is found.
// The title is matched against the plain text content of the heading.
func (pr *PartialRenderer) Section(title string) *model.NoteViewSection {
	return pr.sectionFromNodes(pr.collectTopLevelNodes(), title)
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

	pr.withCurrentPage(func() {
		// Render only the children of the heading (the content inside).
		for child := heading.FirstChild(); child != nil; child = child.NextSibling() {
			err := pr.md.Renderer().Render(&buf, pr.content, child)
			if err != nil {
				continue // Skip nodes that can't be rendered.
			}
		}
	})

	return buf.String()
}

func (pr *PartialRenderer) renderNodeRange(allNodes []ast.Node, start, end int) string {
	if start < 0 || start >= len(allNodes) || end <= start {
		return ""
	}

	var buf bytes.Buffer

	pr.withCurrentPage(func() {
		for i := start; i < end && i < len(allNodes); i++ {
			node := allNodes[i]
			err := pr.md.Renderer().Render(&buf, pr.content, node)
			if err != nil {
				continue // Skip nodes that can't be rendered.
			}
		}
	})

	return buf.String()
}
