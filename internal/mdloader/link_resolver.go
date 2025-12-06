package mdloader

import (
	"bytes"
	"net/url"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"go.abhg.dev/goldmark/wikilink"
)

type myLinkResolver struct {
	log logger.Logger

	nvs *model.NoteViews

	currentPage *model.NoteView

	version string
}

const _html = ".html"
const _hash = "#"

// DefaultVersion does not add ?version= to the URL.
const DefaultVersion = "live"

func (r *myLinkResolver) ResolveWikilink(n *wikilink.Node) ([]byte, error) {
	assetReplace, ok := r.currentPage.AssetReplaces[string(n.Target)]
	if ok && assetReplace != nil {
		return []byte(assetReplace.URL), nil
	}

	// Remove .html extension if present in the target
	target := n.Target
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

	// TODO: don't resolve links to assets, not only images
	if len(r.version) > 0 && r.version != DefaultVersion && !resolveAsImage(n) {
		// Add ?version= to the end
		// Wikilinks can contain special chars like %, so we need to build URL manually
		destStr := string(dest[:i])
		encoded := url.PathEscape(destStr) + "?version=" + url.QueryEscape(r.version)

		return []byte(encoded), nil
	}

	return dest[:i], nil
}
