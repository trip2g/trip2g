package gitapi

import (
	"path"
	"strings"
)

// resolveAssetPath maps an asset link as written in a note to a file in the
// pushed tree, following the same order as the sync plugin's CLI resolver
// (obsidian-sync src/sync/resolve.ts): explicit paths as given, then a bare
// name at the root, under assets/, next to the note, and finally anywhere by
// the shortest-path rule. Returns "" when no file matches.
func resolveAssetPath(files map[string]struct{}, link, notePath string) string {
	exists := func(p string) bool {
		_, ok := files[p]
		return ok
	}

	switch {
	case strings.HasPrefix(link, "./"):
		p := path.Join(path.Dir(notePath), link[2:])
		if exists(p) {
			return p
		}
		return ""
	case strings.HasPrefix(link, "/"):
		p := link[1:]
		if exists(p) {
			return p
		}
		return ""
	case strings.Contains(link, "/"):
		if exists(link) {
			return link
		}
		return ""
	}

	if exists(link) {
		return link
	}

	if p := path.Join("assets", link); exists(p) {
		return p
	}

	if dir := path.Dir(notePath); dir != "." {
		if p := path.Join(dir, link); exists(p) {
			return p
		}
	}

	return pickByBasename(files, link)
}

// pickByBasename picks the file named name (case-insensitive) with the fewest
// path segments, ties broken lexicographically.
func pickByBasename(files map[string]struct{}, name string) string {
	wanted := strings.ToLower(name)

	best := ""
	bestDepth := 0
	for p := range files {
		if strings.ToLower(path.Base(p)) != wanted {
			continue
		}
		depth := strings.Count(p, "/") + 1
		if best == "" || depth < bestDepth || (depth == bestDepth && p < best) {
			best = p
			bestDepth = depth
		}
	}

	return best
}
