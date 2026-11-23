package notepath

import (
	pathpkg "path"
	"strings"
)

// Normalize strips leading "/" and "./" from a note path, then path.Clean-resolves
// "."/".." segments. Returns "" when the path is empty, contains a NUL byte or
// resolves outside the vault root.
func Normalize(p string) string {
	p = strings.TrimSpace(p)
	if strings.ContainsRune(p, 0) {
		return ""
	}
	for strings.HasPrefix(p, "./") || strings.HasPrefix(p, "/") {
		p = strings.TrimPrefix(strings.TrimPrefix(p, "./"), "/")
	}
	if p == "" {
		return ""
	}
	c := pathpkg.Clean(p)
	if c == "." || c == ".." || strings.HasPrefix(c, "../") {
		return ""
	}
	return c
}
