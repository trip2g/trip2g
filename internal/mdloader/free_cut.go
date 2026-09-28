package mdloader

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"

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

// renderFreeContent renders top-level blocks until any limit is reached.
// Each block that produces output (paragraph, heading, list, table, callout,
// ...) counts once toward paragraphLimit and is rendered whole; each --- counts
// toward cutLimit and is not rendered. A container that holds a --- (e.g. a
// callout) is walked into instead, so the cut still applies inside it.
func (ldr *loader) renderFreeContent(buf *bytes.Buffer, root ast.Node, source []byte, cutLimit int, paragraphLimit int) error {
	if cutLimit <= 0 && paragraphLimit <= 0 {
		return errors.New("at least one limit must be positive")
	}

	fc := freeCut{
		renderer:       ldr.md.Renderer(),
		buf:            buf,
		source:         source,
		cutLimit:       cutLimit,
		paragraphLimit: paragraphLimit,
	}
	_, err := fc.renderBlocks(root)
	return err
}

type freeCut struct {
	renderer       renderer.Renderer
	buf            *bytes.Buffer
	source         []byte
	cutLimit       int
	paragraphLimit int
	cutCount       int
	paragraphCount int
}

// renderBlocks renders the children of parent and reports whether a limit
// was reached.
func (fc *freeCut) renderBlocks(parent ast.Node) (bool, error) {
	for node := parent.FirstChild(); node != nil; node = node.NextSibling() {
		if node.Kind() == ast.KindThematicBreak {
			if fc.cutLimit <= 0 {
				continue
			}
			fc.cutCount++
			if fc.cutCount >= fc.cutLimit {
				return true, nil
			}
			continue
		}

		if fc.cutLimit > 0 && isCutContainer(node) && containsThematicBreak(node) {
			done, err := fc.renderBlocks(node)
			if done || err != nil {
				return done, err
			}
			continue
		}

		if fc.paragraphLimit > 0 && fc.paragraphCount >= fc.paragraphLimit {
			return true, nil
		}

		before := fc.buf.Len()
		err := fc.renderer.Render(fc.buf, fc.source, node)
		if err != nil {
			return true, fmt.Errorf("failed to render node: %w", err)
		}

		// Comments and omitted raw HTML render nothing and don't use up the quota.
		if !isEmptyOutput(fc.buf.Bytes()[before:]) {
			fc.paragraphCount++
		}
	}

	return false, nil
}

// isCutContainer reports whether a --- inside node should cut the preview.
// Lists and blockquotes are rendered whole, as they always were.
func isCutContainer(node ast.Node) bool {
	switch node.Kind() {
	case ast.KindList, ast.KindBlockquote:
		return false
	}
	return node.HasChildren() && node.Type() == ast.TypeBlock
}

func containsThematicBreak(node ast.Node) bool {
	found := false
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && n.Kind() == ast.KindThematicBreak {
			found = true
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return found
}

// isEmptyOutput reports whether rendered HTML shows nothing: blank, or only
// goldmark's placeholder for omitted raw HTML.
func isEmptyOutput(out []byte) bool {
	trimmed := bytes.TrimSpace(out)
	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("<!-- raw HTML omitted -->"))
}
