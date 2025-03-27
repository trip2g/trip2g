package mdloader

import (
	"bytes"

	"go.abhg.dev/goldmark/wikilink"
)

type myLinkResolver struct{}

const _html = ".html"
const _hash = "#"

func (r *myLinkResolver) ResolveWikilink(n *wikilink.Node) ([]byte, error) {
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
	return dest[:i], nil
}
