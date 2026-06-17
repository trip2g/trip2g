package model

import "strings"

// OGImageURL resolves an explicit social-preview image from frontmatter.
// It reads the "og_image" key (falling back to "cover"), which holds a vault
// [[link]] or a plain asset path, and resolves it to a presigned asset URL via
// AssetReplaces — mirroring the chart frontmatter-asset resolution
// (mdloader.chartRenderer.resolveFrontmatterSrc). Returns "" when no key is set
// or the asset cannot be resolved.
//
// The renderer prefers this over the first body image (see buildOGTags), so an
// author can set a social image without placing it in the note body.
func (n *NoteView) OGImageURL() string {
	target := n.frontmatterAssetTarget("og_image")
	if target == "" {
		target = n.frontmatterAssetTarget("cover")
	}
	if target == "" {
		return ""
	}

	if ar, ok := n.AssetReplaces[target]; ok && ar != nil {
		return ar.URL
	}
	// Frontmatter links may be resolved into ResolvedLinks during link extraction.
	if resolved, ok := n.ResolvedLinks[target]; ok {
		if ar, ok2 := n.AssetReplaces[resolved]; ok2 && ar != nil {
			return ar.URL
		}
	}
	return ""
}

// frontmatterAssetTarget extracts a single asset target from a frontmatter key
// holding a string or [[wikilink]] (or a list, taking the first item),
// stripping the wikilink wrapper and any |alias.
func (n *NoteView) frontmatterAssetTarget(key string) string {
	switch v := n.RawMeta[key].(type) {
	case string:
		return stripMetaWikilink(v)
	case []interface{}:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return stripMetaWikilink(s)
			}
		}
	}
	return ""
}

func stripMetaWikilink(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[[")
	s = strings.TrimSuffix(s, "]]")
	if i := strings.IndexByte(s, '|'); i != -1 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
