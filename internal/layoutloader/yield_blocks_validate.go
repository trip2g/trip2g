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
	if action, ok := node.(*jet.ActionNode); ok {
		if action.Pipe != nil && len(action.Pipe.Cmds) > 0 {
			cmd := action.Pipe.Cmds[0]
			if ident, identOk := cmd.BaseExpr.(*jet.IdentifierNode); identOk && ident.Ident == "yield_blocks" {
				if len(cmd.Exprs) > 0 {
					if strNode, nodeOk := cmd.Exprs[0].(*jet.StringNode); nodeOk {
						pattern := strNode.Text
						if len(pattern) >= 2 && pattern[0] == '/' && pattern[len(pattern)-1] == '/' {
							if _, err := regexp.Compile(pattern[1 : len(pattern)-1]); err != nil {
								w.warnings = append(w.warnings, model.NoteWarning{
									Level:   model.NoteWarningCritical,
									Message: fmt.Sprintf("yield_blocks: invalid regexp %q: %v", pattern, err),
								})
							}
						}
					}
				}
			}
		}
	}
	vc.Visit(node)
}
