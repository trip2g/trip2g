package handletgcanvasupdate

import (
	"regexp"
	"strings"
)

var (
	frontmatterRe      = regexp.MustCompile(`(?s)\A---\r?\n.*?\r?\n---\r?\n*`)
	frontmatterBlockRe = regexp.MustCompile(`(?s)\A---\r?\n(.*?)\r?\n---\r?\n*`)
	frontmatterImageRe = regexp.MustCompile(`(?m)^image:\s*"?([^"\r\n]+?)"?\s*$`)
	embedRe            = regexp.MustCompile(`!\[\[([^\]\n]+)\]\]\n?`)
	imageExtRe         = regexp.MustCompile(`(?i)\.(png|jpg|jpeg|gif|webp|bmp|svg)$`)
)

// extractFirstImage returns the first image reference from note content and
// the body with frontmatter and consumed image embed stripped.
// Used as fallback when NoteView doesn't have a pre-extracted first image.
func extractFirstImage(content string) (image, body string) {
	if m := frontmatterBlockRe.FindStringSubmatch(content); len(m) > 1 {
		if im := frontmatterImageRe.FindStringSubmatch(m[1]); len(im) > 1 {
			image = strings.TrimSpace(im[1])
		}
	}

	body = strings.TrimSpace(frontmatterRe.ReplaceAllString(content, ""))

	if image != "" {
		return image, body
	}

	loc := embedRe.FindAllStringSubmatchIndex(body, -1)
	for _, m := range loc {
		target := body[m[2]:m[3]]
		if !imageExtRe.MatchString(target) {
			continue
		}
		image = target
		body = strings.TrimSpace(body[:m[0]] + body[m[1]:])
		break
	}
	return image, body
}
