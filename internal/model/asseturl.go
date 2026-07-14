package model

import "net/url"

// AssetRoutePrefix is the stable, content-addressed note-asset route prefix.
// Assets are served as /_system/assets/{sha256}/{fileName}.
const AssetRoutePrefix = "/_system/assets/"

// NoteAssetURLPath builds the stable public URL path for a note asset.
// The URL is content-addressed (sha256), so it never expires and is safe to
// cache immutably; the serving route resolves the hash back to storage.
func NoteAssetURLPath(sha256Hash, fileName string) string {
	return AssetRoutePrefix + sha256Hash + "/" + url.PathEscape(fileName)
}
