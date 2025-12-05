package importtelegramchannel

import (
	"fmt"
	"regexp"
)

var (
	// tgLinkRegex matches [text](https://t.me/channel/123) - any channel.
	tgLinkRegex = regexp.MustCompile(`\[([^\]]*)\]\(https?://t\.me/[^/]+/(\d+)\)`)
	// Custom emoji with tg://emoji?id=123
	customEmojiReplaceRegex = regexp.MustCompile(`!\[([^\]]*)\]\(tg://emoji\?id=(\d+)\)`)
)

func replaceTelegramLinks(content string, postMap map[string]string) string {
	// Replace telegram channel links with wikilinks
	result := tgLinkRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatches := tgLinkRegex.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}

		postID := submatches[2]

		// Look up in map
		if title, ok := postMap[postID]; ok {
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
