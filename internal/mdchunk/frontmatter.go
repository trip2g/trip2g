package mdchunk

import "strings"

// StripFrontmatter removes the YAML frontmatter block (--- ... ---) from the
// beginning of markdown content. Returns the content unchanged if no valid
// frontmatter block is found (e.g. no opening ---, missing closing ---, or
// opening --- not immediately followed by a newline).
func StripFrontmatter(content string) string {
	if len(content) < 3 || content[:3] != "---" {
		return content
	}
	rest := content[3:]
	if len(rest) == 0 || (rest[0] != '\n' && rest[0] != '\r') {
		return content
	}
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return content
	}
	after := rest[idx+4:]
	if len(after) > 0 && after[0] == '\r' {
		after = after[1:]
	}
	if len(after) > 0 && after[0] == '\n' {
		after = after[1:]
	}
	return after
}
