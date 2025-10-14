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

func (pr *PartialRenderer) HeadingBlocks(level int) []model.NoteViewHeadingBlock {
	if pr.ast == nil || pr.content == nil {
		return nil
	}

	var blocks []model.NoteViewHeadingBlock
	var allNodes []ast.Node

	// First pass: collect all top-level nodes
	for child := pr.ast.FirstChild(); child != nil; child = child.NextSibling() {
		allNodes = append(allNodes, child)
	}

	var currentBlock *model.NoteViewHeadingBlock
	var contentStart int = -1

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
				currentBlock = &model.NoteViewHeadingBlock{
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
