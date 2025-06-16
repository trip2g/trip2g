package mdloader

import (
	"bytes"
	"fmt"
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
	assetPath, ok := r.currentPage.AssetReplaces[string(n.Target)]
	if ok {
		return []byte(assetPath), nil
	}

	// fmt.Println("Resolving wikilink:", string(n.Target))

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

	if len(r.version) > 0 && r.version != DefaultVersion {
		// parse url and add ?version= to the end
		u, err := url.Parse(string(dest[:i]))
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL: %w", err)
		}

		query := u.Query()
		query.Set("version", r.version)
		u.RawQuery = query.Encode()

		return []byte(u.String()), nil
	}

	return dest[:i], nil
}
