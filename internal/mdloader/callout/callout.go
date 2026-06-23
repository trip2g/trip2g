// Package callout renders Obsidian callouts (`> [!type]` blockquotes) as
// styled callout blocks. It implements a goldmark extension that rewrites
// matching blockquotes into Callout AST nodes and renders them as semantic
// HTML (a <div> for static callouts, <details>/<summary> for foldable ones).
package callout

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Extension is the goldmark extension for Obsidian callouts.
var Extension goldmark.Extender = &extension{}

type extension struct{}

func (e *extension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&transformer{}, 100),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(newRenderer(), 100),
		),
	)
}
