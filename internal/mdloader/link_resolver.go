package mdloader

import (
	"bytes"
	"fmt"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"go.abhg.dev/goldmark/wikilink"
)

type myLinkResolver struct {
	log logger.Logger

	nvs *model.NoteViews

	currentPage *model.NoteView
}

const _html = ".html"
const _hash = "#"

func (r *myLinkResolver) ResolveWikilink(n *wikilink.Node) ([]byte, error) {
	assetPath, ok := r.currentPage.AssetReplaces[string(n.Target)]
	fmt.Println("assetPath", string(n.Target), ok, assetPath, r.currentPage.AssetReplaces)
	if ok {
		return []byte(assetPath), nil
	}

	// Remove .html extension if present in the target
	target := n.Target
	if bytes.HasSuffix(target, []byte(_html)) {
		target = target[:len(target)-len(_html)]
	}

	// currentParts := strings.Split(r.currentPage.Permalink, "/")
	// pageFound := false
	//
	// for i := len(currentParts) - 1; i >= 0; i-- {
	// 	targetPermalink := strings.Join(currentParts[:i], "/") + "/" + string(target)
	//
	// 	targetPage, ok := r.pages[targetPermalink]
	// 	if ok {
	// 		target = []byte(targetPage.Permalink)
	// 		pageFound = true
	// 		break
	// 	}
	// }
	//
	// if !pageFound {
	// 	r.log.Warn("Page not found", "target", string(target), "page", r.currentPage.Permalink)
	// }

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
