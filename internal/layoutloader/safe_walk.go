package layoutloader

import (
	"fmt"

	"github.com/CloudyKit/jet/v6"
	"github.com/CloudyKit/jet/v6/utils"
)

// safeWalk is a drop-in for utils.Walk that guards against known Jet AST
// walker pitfalls so individual visitors don't need to handle them:
//
//   - IncludeNode triggers infinite recursion in Jet's walker — skipped.
//   - YieldNode.Parameters may be nil for {{ yield content }} nodes — initialised.
//   - Jet's walker panics with "unexpected node" on {{ try }}/{{ catch }},
//     {{ return }} and the _ identifier — the walker descends into them itself.
func safeWalk(tmpl *jet.Template, v utils.Visitor) {
	utils.Walk(tmpl, &guardedWalker{inner: v})
}

// walkContained runs safeWalk and recovers panics from Jet's AST walker
// (e.g. a node type added in a newer Jet). Used where one
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
	switch n := node.(type) {
	case *jet.IncludeNode, *jet.UnderscoreNode:
		return
	case *jet.YieldNode:
		if n.Parameters == nil {
			n.Parameters = &jet.BlockParameterList{}
		}
	case *jet.ReturnNode:
		g.Visit(vc, n.Value)
		return
	case *jet.TryNode:
		g.Visit(vc, n.List)
		if n.Catch != nil && n.Catch.List != nil {
			g.Visit(vc, n.Catch.List)
		}
		return
	}
	g.inner.Visit(vc, node)
}
