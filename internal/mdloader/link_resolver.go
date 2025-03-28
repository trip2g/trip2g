package mdloader

import (
	"bytes"
	"strings"
	"trip2g/internal/logger"

	"go.abhg.dev/goldmark/wikilink"
)

type myLinkResolver struct {
	log logger.Logger

	pages map[string]*Page

	currentPage *Page
}

const _html = ".html"
const _hash = "#"

func (r *myLinkResolver) ResolveWikilink(n *wikilink.Node) ([]byte, error) {
	// Remove .html extension if present in the target
	target := n.Target
	if bytes.HasSuffix(target, []byte(_html)) {
		target = target[:len(target)-len(_html)]
	}

	currentParts := strings.Split(r.currentPage.Permalink, "/")
	pageFound := false

	for i := len(currentParts) - 1; i >= 0; i-- {
		targetPermalink := strings.Join(currentParts[:i], "/") + "/" + string(target)

		targetPage, ok := r.pages[targetPermalink]
		if ok {
			target = []byte(targetPage.Permalink)
			pageFound = true
			break
		}
	}

	if !pageFound {
		r.log.Warn("Page not found", "target", string(target), "page", r.currentPage.Permalink)
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
