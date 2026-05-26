package layoutloader

import (
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
