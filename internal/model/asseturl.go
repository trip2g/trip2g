package model

import (
	"net/url"
	"strings"
)

// AssetRoutePrefix is the stable, content-addressed note-asset route prefix.
// Assets are served as /_system/assets/{sha256}/{fileName}.
const AssetRoutePrefix = "/_system/assets/"

// NoteAssetURLPath builds the stable public URL path for a note asset.
// The URL is content-addressed (sha256), so it never expires and is safe to
// cache immutably; the serving route resolves the hash back to storage. It is
// intentionally relative — an in-page reference is same-origin and a relative
// path is more cache-friendly (works unmodified behind any CDN/host alias).
// Consumers that emit the URL somewhere it will be dereferenced from a
// different origin (Telegram, og:image, JSON-LD, RSS, ...) MUST pass it
// through AbsoluteURL first — a relative URL is meaningless off-site.
func NoteAssetURLPath(sha256Hash, fileName string) string {
	return AssetRoutePrefix + sha256Hash + "/" + url.PathEscape(fileName)
}

// AbsoluteURL turns a URL that may be relative (as produced by
// NoteAssetURLPath, or any other site-relative path) into an absolute one
// using baseURL (typically the site's configured PublicURL). Already-absolute
// URLs (containing "://") and the empty string pass through unchanged, so
// it's always safe to call even when the input's origin is uncertain.
func AbsoluteURL(baseURL, path string) string {
	if path == "" || strings.Contains(path, "://") {
		return path
	}
	base := strings.TrimRight(baseURL, "/")
	if !strings.HasPrefix(path, "/") {
		return base + "/" + path
	}
	return base + path
}

// AbsolutizeAssetURLsInHTML rewrites quoted /_system/assets/... references
// inside pre-rendered note HTML to absolute URLs. Note bodies embed relative
// asset URLs (img src, link href, ...) because that HTML is normally served
// same-origin; but some consumers (RSS content:encoded, emailed content, ...)
// republish that HTML on a different origin, where a relative URL breaks. The
// route prefix is a stable, unambiguous marker, so a direct string replace is
// sufficient — no HTML parsing needed. baseURL=="" is a no-op (nothing to
// absolutize against).
func AbsolutizeAssetURLsInHTML(html, baseURL string) string {
	if baseURL == "" || !strings.Contains(html, AssetRoutePrefix) {
		return html
	}
	base := strings.TrimRight(baseURL, "/")
	return strings.ReplaceAll(html, `"`+AssetRoutePrefix, `"`+base+AssetRoutePrefix)
}
