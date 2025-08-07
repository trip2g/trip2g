package mdloader

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/yuin/goldmark/ast"

	"trip2g/internal/model"
)

// generateFreeHTML generates free content HTML based on metadata configuration.
func (ldr *loader) generateFreeHTML(p *model.NoteView) error {
	// Check metadata for free_cut or free_paragraphs
	var freeCut int
	var freeParagraphs int

	// Check for free_cut in metadata
	if cutValue, ok := p.RawMeta["free_cut"]; ok {
		switch v := cutValue.(type) {
		case bool:
			if v {
				freeCut = 1 // free_cut: true means cut after first paragraph
			}
		case float64:
			freeCut = int(v)
		case int:
			freeCut = v
		}
	}

	// Check for free_paragraphs in metadata
	if paragraphsValue, ok := p.RawMeta["free_paragraphs"]; ok {
		switch v := paragraphsValue.(type) {
		case float64:
			freeParagraphs = int(v)
		case int:
			freeParagraphs = v
		}
	}

	// If neither is set in metadata, use config default
	if freeCut == 0 && freeParagraphs == 0 {
		freeParagraphs = ldr.config.FreeParagraphs
	}

	// If still nothing is set, no free HTML needed
	if freeCut == 0 && freeParagraphs == 0 {
		return nil
	}

	// Determine how many paragraphs to include
	paragraphsToInclude := freeParagraphs
	if freeCut > 0 {
		paragraphsToInclude = freeCut
	}

	// Extract the first N paragraphs from the AST
	freeAst, err := ldr.extractFirstNParagraphs(p.Ast(), paragraphsToInclude)
	if err != nil {
		return fmt.Errorf("failed to extract free paragraphs: %w", err)
	}

	// Render the free content
	var buf bytes.Buffer
	err = ldr.md.Renderer().Render(&buf, p.Content, freeAst)
	if err != nil {
		return fmt.Errorf("failed to render free HTML: %w", err)
	}

	p.FreeHTML = template.HTML(buf.String()) //nolint:gosec // it's safe from admins

	return nil
}

// extractFirstNParagraphs creates a new AST containing only the first N paragraphs.
func (ldr *loader) extractFirstNParagraphs(root ast.Node, n int) (ast.Node, error) {
	if n <= 0 {
		return nil, fmt.Errorf("invalid number of paragraphs: %d", n)
	}

	// Create a new document node
	doc := ast.NewDocument()
	doc.SetBlankPreviousLines(false)

	paragraphCount := 0
	var currentContainer ast.Node = doc

	// Walk through the original AST and copy nodes
	err := ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		// Handle different node types
		switch node.Kind() {
		case ast.KindDocument:
			// Skip the document node itself
			return ast.WalkContinue, nil

		case ast.KindParagraph:
			if paragraphCount >= n {
				return ast.WalkStop, nil
			}

			// Clone the paragraph and its content
			para := ast.NewParagraph()
			currentContainer.AppendChild(currentContainer, para)

			// Copy all children of the paragraph
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				cloned := ldr.cloneNode(child)
				if cloned != nil {
					para.AppendChild(para, cloned)
				}
			}

			paragraphCount++
			return ast.WalkSkipChildren, nil

		case ast.KindHeading, ast.KindList, ast.KindBlockquote, ast.KindCodeBlock, ast.KindThematicBreak:
			// Include these block elements if we haven't reached the limit
			if paragraphCount >= n {
				return ast.WalkStop, nil
			}

			cloned := ldr.cloneNode(node)
			if cloned != nil {
				currentContainer.AppendChild(currentContainer, cloned)

				// For container nodes, we need to handle their children
				if node.ChildCount() > 0 {
					ldr.cloneChildren(node, cloned)
				}
			}

			// Count block elements as paragraphs
			paragraphCount++
			return ast.WalkSkipChildren, nil

		default:
			// Skip other nodes at the top level
			return ast.WalkContinue, nil
		}
	})

	if err != nil {
		return nil, err
	}

	return doc, nil
}

// cloneNode creates a deep copy of an AST node.
func (ldr *loader) cloneNode(node ast.Node) ast.Node {
	switch n := node.(type) {
	case *ast.Text:
		cloned := ast.NewTextSegment(n.Segment)
		return cloned
	case *ast.String:
		cloned := ast.NewString(n.Value)
		return cloned
	case *ast.CodeSpan:
		return ast.NewCodeSpan()
	case *ast.Emphasis:
		em := ast.NewEmphasis(n.Level)
		return em
	case *ast.Link:
		link := ast.NewLink()
		link.Destination = n.Destination
		link.Title = n.Title
		return link
	case *ast.Image:
		img := ast.NewImage(ast.NewLink())
		img.Destination = n.Destination
		img.Title = n.Title
		return img
	case *ast.Heading:
		return ast.NewHeading(n.Level)
	case *ast.Paragraph:
		return ast.NewParagraph()
	case *ast.List:
		list := ast.NewList(n.Marker)
		list.IsTight = n.IsTight
		list.Start = n.Start
		return list
	case *ast.ListItem:
		return ast.NewListItem(n.Offset)
	case *ast.Blockquote:
		return ast.NewBlockquote()
	case *ast.CodeBlock:
		return ast.NewCodeBlock()
	case *ast.FencedCodeBlock:
		fcb := ast.NewFencedCodeBlock(n.Info)
		fcb.SetLines(n.Lines())
		return fcb
	case *ast.ThematicBreak:
		return ast.NewThematicBreak()
	default:
		// For unknown types, return nil
		return nil
	}
}

// cloneChildren recursively clones all children of a node.
func (ldr *loader) cloneChildren(source, dest ast.Node) {
	for child := source.FirstChild(); child != nil; child = child.NextSibling() {
		cloned := ldr.cloneNode(child)
		if cloned != nil {
			dest.AppendChild(dest, cloned)
			if child.ChildCount() > 0 {
				ldr.cloneChildren(child, cloned)
			}
		}
	}
}
