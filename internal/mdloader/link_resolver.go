package mdloader

import (
	"bytes"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"go.abhg.dev/goldmark/wikilink"
)

type myLinkResolver struct {
	log logger.Logger

	nvs *model.NoteViews

	currentPage *model.NoteView

	// domainRenderNotes maps domain-specific paths to NoteViews during
	// domain re-render. Used by linkRenderer to find notes for data-pid
	// and paywall classes when the href is a domain path (not a permalink).
	// nil during normal rendering.
	domainRenderNotes map[string]*model.NoteView
}

const _html = ".html"
const _hash = "#"

func (r *myLinkResolver) ResolveWikilink(n *wikilink.Node) ([]byte, error) {
	assetReplace, ok := r.currentPage.AssetReplaces[string(n.Target)]
	if ok && assetReplace != nil {
		return []byte(assetReplace.URL), nil
	}

	// Check if this link was resolved in extractInLinks
	// This allows us to avoid mutating the AST, which breaks caching
	target := n.Target
	if resolved, found := r.currentPage.ResolvedLinks[string(n.Target)]; found {
		target = []byte(resolved)
	}

	// Remove .html extension if present in the target
	if bytes.HasSuffix(target, []byte(_html)) {
		target = target[:len(target)-len(_html)]
	}

	dest := make([]byte, len(target)+len(_hash)+len(n.Fragment))
	var i int
	if len(target) > 0 {
		i += copy(dest, target)
	}
	if len(n.Fragment) > 0 {
		i += copy(dest[i:], _hash)
		i += copy(dest[i:], n.Fragment)
	}

	return dest[:i], nil
}
