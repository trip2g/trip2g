package notepath

import (
	pathpkg "path"
	"strings"
)

// Normalize strips leading "/" and "./" from a note path, then path.Clean-resolves
// "."/".." segments. Returns "" when the path is empty, contains a NUL byte or
// resolves outside the vault root. Backslashes and surrounding whitespace are
// rejected rather than rewritten.
func Normalize(p string) string {
	if p != strings.TrimSpace(p) || strings.ContainsRune(p, 0) || strings.ContainsRune(p, '\\') {
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
