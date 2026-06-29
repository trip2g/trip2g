package layoutloader

import (
	"github.com/CloudyKit/jet/v6"
	"github.com/CloudyKit/jet/v6/utils"
)

// Identifiers/fields whose use makes a layout's output depend on the viewer.
const (
	identCurrentUser       = "currentUser"
	identNote              = "note"
	fieldLastEditedBy      = "LastEditedBy"
	fieldLastEditedByLabel = "LastEditedByLabel"
)

// personalizationFinder walks a Jet template AST and flags references to
// viewer/role-dependent helpers, which make the rendered output depend on WHO
// is viewing. A layout that uses any of them must never be served from the
// anonymous page cache.
//
// Triggers:
//   - any access to the `currentUser` namespace (e.g. currentUser.IsAdmin()).
//   - note.LastEditedBy / note.LastEditedByLabel (the admin-only byline).
//
// For the anonymous page cache this list is defense-in-depth only: the cache is
// keyed and gated to anon viewers, so a missed trigger here cannot serve one
// viewer's personalized output to another — it would at worst cache an anon
// render that need not have been cached.
type personalizationFinder struct{ found bool }

func (w *personalizationFinder) Visit(vc utils.VisitorContext, node jet.Node) {
	switch n := node.(type) {
	case *jet.IdentifierNode:
		// A bare `currentUser` reference is enough; the namespace exists only
		// to gate role-specific output.
		if n.Ident == identCurrentUser {
			w.found = true
		}
	case *jet.ChainNode:
		if id, ok := n.Node.(*jet.IdentifierNode); ok {
			switch id.Ident {
			case identCurrentUser:
				w.found = true
			case identNote:
				for _, field := range n.Field {
					if field == fieldLastEditedBy || field == fieldLastEditedByLabel {
						w.found = true
					}
				}
			}
		}
	}
	vc.Visit(node)
}

// detectPersonalized reports whether view — or any template it imports —
// references viewer-dependent helpers. content is the source template text,
// used to resolve {{ import }} paths the AST walker does not recurse into
// (mirroring how assetFinder/blockFinder follow imports in processTemplates).
func (jl *jetLoader) detectPersonalized(sourceID, content string, view *jet.Template) bool {
	finder := &personalizationFinder{}
	safeWalk(view, finder)
	if finder.found {
		return true
	}

	if views, ok := jl.sets[sourceID]; ok {
		for _, importPath := range extractImportPaths(content) {
			imported, err := views.GetTemplate(importPath)
			if err != nil {
				continue
			}
			safeWalk(imported, finder)
			if finder.found {
				return true
			}
		}
	}

	return finder.found
}
