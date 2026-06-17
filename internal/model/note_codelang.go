package model

import (
	"strings"

	"github.com/yuin/goldmark/ast"
)

// extractCodeLanguages walks the AST and records the language of every fenced
// code block into n.CodeLanguages (lowercased, first info-string token only).
// The set lets the renderer decide which per-language client widgets a page
// needs (e.g. load mermaid only when a `mermaid` block is present).
func (n *NoteView) extractCodeLanguages() {
	n.CodeLanguages = nil

	if n.ast == nil {
		return
	}

	_ = ast.Walk(n.ast, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		fcb, ok := node.(*ast.FencedCodeBlock)
		if !ok {
			return ast.WalkContinue, nil
		}

		lang := fcb.Language(n.Content)
		if len(lang) == 0 {
			return ast.WalkContinue, nil
		}

		key := strings.ToLower(string(lang))
		if n.CodeLanguages == nil {
			n.CodeLanguages = map[string]bool{}
		}
		n.CodeLanguages[key] = true

		return ast.WalkContinue, nil
	})
}

// HasCodeLanguage reports whether the note contains a fenced code block with the
// given language (case-insensitive). Nil-safe.
func (n *NoteView) HasCodeLanguage(lang string) bool {
	if n.CodeLanguages == nil {
		return false
	}
	return n.CodeLanguages[strings.ToLower(lang)]
}
