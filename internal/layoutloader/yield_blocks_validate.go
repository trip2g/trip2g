package layoutloader

import (
	"fmt"
	"regexp"
	"trip2g/internal/model"

	"github.com/CloudyKit/jet/v6"
	"github.com/CloudyKit/jet/v6/utils"
)

type yieldBlocksValidator struct {
	warnings []model.NoteWarning
}

func (w *yieldBlocksValidator) Visit(vc utils.VisitorContext, node jet.Node) {
	if node == nil {
		return
	}
	w.validateYieldBlocksPattern(node)
	vc.Visit(node)
}

func (w *yieldBlocksValidator) validateYieldBlocksPattern(node jet.Node) {
	action, ok := node.(*jet.ActionNode)
	if !ok || action.Pipe == nil || len(action.Pipe.Cmds) == 0 {
		return
	}
	cmd := action.Pipe.Cmds[0]
	ident, identOk := cmd.BaseExpr.(*jet.IdentifierNode)
	if !identOk || ident.Ident != "yield_blocks" || len(cmd.Exprs) == 0 {
		return
	}
	strNode, nodeOk := cmd.Exprs[0].(*jet.StringNode)
	if !nodeOk {
		return
	}
	pattern := strNode.Text
	if len(pattern) < 2 || pattern[0] != '/' || pattern[len(pattern)-1] != '/' {
		return
	}
	if _, err := regexp.Compile(pattern[1 : len(pattern)-1]); err != nil {
		w.warnings = append(w.warnings, model.NoteWarning{
			Level:   model.NoteWarningCritical,
			Message: fmt.Sprintf("yield_blocks: invalid regexp %q: %v", pattern, err),
		})
	}
}
