package layoutloader

import (
	"testing"

	"github.com/CloudyKit/jet/v6"
	"github.com/CloudyKit/jet/v6/utils"
	"github.com/stretchr/testify/require"
)

// paramAccessVisitor accesses YieldNode.Parameters.List without a nil guard —
// exactly what user-written visitors do. Panics when Parameters is nil.
type paramAccessVisitor struct{ count int }

func (p *paramAccessVisitor) Visit(vc utils.VisitorContext, node jet.Node) {
	if node == nil {
		return
	}
	if _, ok := node.(*jet.IncludeNode); ok {
		return
	}
	if y, ok := node.(*jet.YieldNode); ok {
		// Accessing .List on a nil *BlockParameterList panics.
		p.count += len(y.Parameters.List)
	}
	vc.Visit(node)
}

// TestSafeWalk_NilYieldParams shows that {{ yield content }} produces a YieldNode
// with nil Parameters in Jet's AST. A visitor that accesses Parameters.List panics
// under utils.Walk. safeWalk initialises Parameters first so the same visitor is safe.
func TestSafeWalk_NilYieldParams(t *testing.T) {
	// {{ yield content }} inside a block sets Parameters=nil in Jet's AST.
	loader := jet.NewInMemLoader()
	loader.Set("/layout",
		`{{ block shell() }}<html>{{ yield content }}</html>{{ end }}`+
			`{{ yield shell() content }}hello{{ end }}`)

	set := jet.NewSet(loader, jet.DevelopmentMode(true))
	view, err := set.GetTemplate("/layout")
	require.NoError(t, err)
	require.NotNil(t, view)

	// utils.Walk + paramAccessVisitor panics on the nil-Parameters YieldNode.
	// If this assertion starts failing, Jet fixed the bug upstream — safeWalk and
	// its nil-Parameters guard can then be removed.
	require.Panics(t, func() {
		utils.Walk(view, &paramAccessVisitor{})
	}, "visitor accessing YieldNode.Parameters.List should panic when Parameters is nil")

	// safeWalk guards nil Parameters before delivering the node to the visitor.
	require.NotPanics(t, func() {
		safeWalk(view, &paramAccessVisitor{})
	}, "safeWalk must initialise Parameters so the visitor never sees nil")
}
