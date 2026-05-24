package handletgnavigationupdate

import (
	"regexp"
	"strings"
)

// wikilinkRe matches ![[embed]] and [[target]] / [[target|display]].
// The embed variant is captured so we can skip it.
var wikilinkRe = regexp.MustCompile(`!\[\[[^\]]+\]\]|\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

var embedRe = regexp.MustCompile(`!\[\[[^\]]*\]\]`)

// Wikilink represents a resolved [[target]] or [[target|display]] link.
type Wikilink struct {
	Target  string // note basename, e.g. "My Note"
	Display string // display text; empty means use Target
}

// ParseWikilinks extracts unique wikilinks from markdown content, preserving order.
// Embeds (![[...]]) are ignored.
func ParseWikilinks(content string) []Wikilink {
	var links []Wikilink
	seen := make(map[string]bool)
	for _, m := range wikilinkRe.FindAllStringSubmatch(content, -1) {
		target := m[1]
		if target == "" { // embed match
			continue
		}
		target = strings.TrimSpace(target)
		if seen[target] {
			continue
		}
		seen[target] = true
		links = append(links, Wikilink{
			Target:  target,
			Display: strings.TrimSpace(m[2]),
		})
	}
	return links
}

// StripWikilinks replaces [[target|display]] with display text and [[target]] with target.
// Embeds are removed entirely.
func StripWikilinks(content string) string {
	return wikilinkRe.ReplaceAllStringFunc(content, func(m string) string {
		sub := wikilinkRe.FindStringSubmatch(m)
		if sub[1] == "" {
			return "" // embed
		}
		if sub[2] != "" {
			return sub[2]
		}
		return sub[1]
	})
}

// StripFrontmatter removes YAML frontmatter (content between leading --- delimiters).
func StripFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	idx := strings.Index(content[3:], "---")
	if idx < 0 {
		return content
	}
	return strings.TrimLeft(content[3+idx+3:], "\n\r")
}

// StripEmbeds removes ![[...]] embed syntax.
func StripEmbeds(content string) string {
	return embedRe.ReplaceAllString(content, "")
}

// TruncateText truncates to maxLen runes, appending "…" if truncated.
func TruncateText(content string, maxLen int) string {
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	return string(runes[:maxLen]) + "…"
}
