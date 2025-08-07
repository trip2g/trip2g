package mdloader

import (
	"bytes"
	"errors"
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
				freeCut = 1 // free_cut: true means cut at first --- or after first paragraph
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

	// Render free content by walking AST and rendering only the allowed nodes
	var buf bytes.Buffer
	err := ldr.renderFreeContent(&buf, p.Ast(), p.Content, freeCut, freeParagraphs)
	if err != nil {
		return fmt.Errorf("failed to render free HTML: %w", err)
	}

	p.FreeHTML = template.HTML(buf.String()) //nolint:gosec // it's safe from admins

	return nil
}

// renderFreeContent walks the AST and renders nodes until any limit is reached.
func (ldr *loader) renderFreeContent(buf *bytes.Buffer, root ast.Node, source []byte, cutLimit int, paragraphLimit int) error {
	if cutLimit <= 0 && paragraphLimit <= 0 {
		return errors.New("at least one limit must be positive")
	}

	cutCount := 0
	paragraphCount := 0
	renderer := ldr.md.Renderer()

	err := ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		// Skip the document node itself
		if node.Kind() == ast.KindDocument {
			return ast.WalkContinue, nil
		}

		// Check for thematic break (--- in markdown) if cutLimit is set
		if cutLimit > 0 && node.Kind() == ast.KindThematicBreak {
			cutCount++
			if cutCount >= cutLimit {
				// Stop when we've reached the Nth cut
				return ast.WalkStop, nil
			}
			// Don't render the --- itself, just count it and continue
			return ast.WalkSkipChildren, nil
		}

		// Count and render paragraphs and block elements
		switch node.Kind() {
		case ast.KindParagraph, ast.KindHeading, ast.KindList, ast.KindBlockquote,
			ast.KindCodeBlock, ast.KindFencedCodeBlock:

			// Check if we've reached the paragraph limit
			if paragraphLimit > 0 && paragraphCount >= paragraphLimit {
				return ast.WalkStop, nil
			}

			// Render this node
			err := renderer.Render(buf, source, node)
			if err != nil {
				return ast.WalkStop, fmt.Errorf("failed to render node: %w", err)
			}

			paragraphCount++
			return ast.WalkSkipChildren, nil
		}

		// Continue walking for other node types
		return ast.WalkContinue, nil
	})

	return err
}
