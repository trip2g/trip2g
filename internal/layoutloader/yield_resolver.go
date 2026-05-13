package layoutloader

import (
	"fmt"
	"trip2g/internal/model"

	"github.com/CloudyKit/jet/v6"
	"github.com/CloudyKit/jet/v6/utils"
)

// blockNameFinder collects {{block}} definition names.
type blockNameFinder struct{ names []string }

func (w *blockNameFinder) Visit(vc utils.VisitorContext, node jet.Node) {
	if node == nil {
		return
	}
	// jet's visitIncludeNode re-visits the IncludeNode itself instead of its children,
	// causing infinite recursion. Skip it — include nodes contain no blocks or yields.
	if _, ok := node.(*jet.IncludeNode); ok {
		return
	}
	// Fix jet panic: YieldNode.Parameters can be nil for partially-parsed templates.
	if y, ok := node.(*jet.YieldNode); ok && y.Parameters == nil {
		y.Parameters = &jet.BlockParameterList{}
	}
	if b, ok := node.(*jet.BlockNode); ok {
		w.names = append(w.names, b.Name)
	}
	vc.Visit(node)
}

// yieldNameFinder collects {{yield name()}} names (excluding {{yield content}}).
type yieldNameFinder struct{ names []string }

func (w *yieldNameFinder) Visit(vc utils.VisitorContext, node jet.Node) {
	if node == nil {
		return
	}
	if _, ok := node.(*jet.IncludeNode); ok {
		return
	}
	if y, ok := node.(*jet.YieldNode); ok {
		if y.Parameters == nil {
			y.Parameters = &jet.BlockParameterList{}
		}
		if !y.IsContent {
			w.names = append(w.names, y.Name)
		}
	}
	vc.Visit(node)
}

// resolveNeededFiles BFS-walks yield deps from the page template.
// Returns ordered list of sourceIDs to inline, flattened block names, and warnings.
func resolveNeededFiles( //nolint:nonamedreturns // named returns used throughout function body
	views *jet.Set,
	pageView *jet.Template,
	registry map[string]blockRegistryEntry,
) (neededFileIDs []string, inlinedBlockNames []string, warnings []model.NoteWarning) {
	visited := make(map[string]bool)
	addedFiles := make(map[string]bool)

	yf := &yieldNameFinder{}
	utils.Walk(pageView, yf)
	queue := yf.names

	for len(queue) > 0 {
		blockName := queue[0]
		queue = queue[1:]

		if visited[blockName] {
			continue
		}
		visited[blockName] = true

		entry, ok := registry[blockName]
		if !ok {
			warnings = append(warnings, model.NoteWarning{
				Level:   model.NoteWarningWarning,
				Message: fmt.Sprintf("yield %q has no matching block in component files", blockName),
			})
			continue
		}

		if !addedFiles[entry.SourceID] {
			addedFiles[entry.SourceID] = true
			neededFileIDs = append(neededFileIDs, entry.SourceID)

			if t, err := views.GetTemplate(entry.SourceID); err == nil && t != nil {
				bf := &blockNameFinder{}
				utils.Walk(t, bf)
				inlinedBlockNames = append(inlinedBlockNames, bf.names...)

				yf2 := &yieldNameFinder{}
				utils.Walk(t, yf2)
				queue = append(queue, yf2.names...)
			}
		}
	}
	return neededFileIDs, inlinedBlockNames, warnings
}
