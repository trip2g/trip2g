package layoutloader

import (
	"fmt"

	"github.com/CloudyKit/jet/v6"
	"github.com/CloudyKit/jet/v6/utils"
)

// safeWalk is a drop-in for utils.Walk that guards against two known Jet AST
// walker pitfalls so individual visitors don't need to handle them:
//
//   - IncludeNode triggers infinite recursion in Jet's walker — skipped.
//   - YieldNode.Parameters may be nil for {{ yield content }} nodes — initialised.
func safeWalk(tmpl *jet.Template, v utils.Visitor) {
	utils.Walk(tmpl, &guardedWalker{inner: v})
}

// walkContained runs safeWalk and recovers panics from Jet's AST walker
// (e.g. "unexpected node _" for underscore range vars). Used where one
// template's walk must not take down analysis of others (block registry,
// imported-template walks). Returns the panic message, or "" on success.
//
//nolint:nonamedreturns // named return required for defer/recover to set it
func walkContained(tmpl *jet.Template, v utils.Visitor) (panicMsg string) {
	defer func() {
		if r := recover(); r != nil {
			panicMsg = fmt.Sprint(r)
		}
	}()
	safeWalk(tmpl, v)
	return ""
}

type guardedWalker struct{ inner utils.Visitor }

func (g *guardedWalker) Visit(vc utils.VisitorContext, node jet.Node) {
	if node == nil {
		return
	}
	if _, ok := node.(*jet.IncludeNode); ok {
		return
	}
	if y, ok := node.(*jet.YieldNode); ok && y.Parameters == nil {
		y.Parameters = &jet.BlockParameterList{}
	}
	g.inner.Visit(vc, node)
}
