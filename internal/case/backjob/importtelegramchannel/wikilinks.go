package importtelegramchannel

import (
	"fmt"
	"regexp"
)

var (
	// tgLinkRegex matches telegram links in two formats:
	// - Public channels: [text](https://t.me/channel_name/123)
	// - Private channels: [text](https://t.me/c/1234567890/123)
	// Captures the LAST number in the path (message ID).
	tgLinkRegex = regexp.MustCompile(`\[([^\]]*)\]\(https?://t\.me/(?:[^/]+/)*(\d+)\)`)
	// Custom emoji with tg://emoji?id=123
	customEmojiReplaceRegex = regexp.MustCompile(`!\[([^\]]*)\]\(tg://emoji\?id=(\d+)\)`)
	// ceEmojiURLRegex matches ![alt](https://ce.trip2g.com/{id}.webp) format.
	ceEmojiURLRegex = regexp.MustCompile(`!\[([^\]]*)\]\(https://ce\.trip2g\.com/(\d+)\.webp\)`)
)

func replaceTelegramLinks(content string, postMap map[string]string) string {
	// Replace telegram channel links with wikilinks
	result := tgLinkRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatches := tgLinkRegex.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}

		linkText := submatches[1]
		postID := submatches[2]

		// Look up in map
		if title, ok := postMap[postID]; ok {
			// Use alias if link text differs from title
			if linkText != "" && linkText != title {
				return fmt.Sprintf("[[%s|%s]]", title, linkText)
			}
			return fmt.Sprintf("[[%s]]", title)
		}

		// Not found - keep original link
		return match
	})

	// Replace custom emoji tg://emoji?id=... with https://ce.trip2g.com/{id}.webp
	result = customEmojiReplaceRegex.ReplaceAllStringFunc(result, func(match string) string {
		submatches := customEmojiReplaceRegex.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		altText := submatches[1]
		emojiID := submatches[2]
		return fmt.Sprintf("![%s](https://ce.trip2g.com/%s.webp)", altText, emojiID)
	})

	return result
}

// extractCustomEmojiIDs extracts unique custom emoji IDs from markdown content.
// Returns a slice of unique emoji IDs found in ce.trip2g.com URLs.
func extractCustomEmojiIDs(content string) []string {
	matches := ceEmojiURLRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	// Deduplicate
	seen := make(map[string]bool)
	var result []string
	for _, match := range matches {
		if len(match) >= 3 {
			emojiID := match[2]
			if !seen[emojiID] {
				seen[emojiID] = true
				result = append(result, emojiID)
			}
		}
	}
	return result
}

// replaceCustomEmojiURLs replaces ce.trip2g.com URLs with local asset paths.
// downloadedEmojis maps emojiID -> local filename (e.g., "tg_ce_123.webp").
func replaceCustomEmojiURLs(content string, downloadedEmojis map[string]string) string {
	return ceEmojiURLRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatches := ceEmojiURLRegex.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		altText := submatches[1]
		emojiID := submatches[2]

		if localFilename, ok := downloadedEmojis[emojiID]; ok {
			return fmt.Sprintf("![%s](./assets/%s)", altText, localFilename)
		}
		// Not downloaded - keep original URL
		return match
	})
}
