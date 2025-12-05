package importtelegramchannel

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	customEmojiRegex         = regexp.MustCompile(`!\[[^\]]*\]\((tg://emoji\?id=\d+|https://ce\.trip2g\.com/\d+\.webp)\)`)
	malformedEmojiRegex      = regexp.MustCompile(`!\[[^\]]*\]\(tg://emoji\?id=\d+\)>[^<]*</u>`)
	numberedEmojiPrefixRegex = regexp.MustCompile(`^!\[[^\]]*\]\([^)]+\)[\.\s]*`)
	markdownLinkRegex        = regexp.MustCompile(`\[([^\]]*)\]\([^)]+\)`)
	htmlTagRegex             = regexp.MustCompile(`</?[a-zA-Z][^>]*>`)
	timecodeRegex            = regexp.MustCompile(`\d{1,2}:\d{2}(?::\d{2})?\s*`)
	// leadingJunkRegex matches emojis and punctuation at the start of text.
	leadingJunkRegex = regexp.MustCompile(
		`^[\x{1F300}-\x{1F9FF}\x{1F3FB}-\x{1F3FF}\x{2600}-\x{26FF}` +
			`\x{2700}-\x{27BF}\x{25A0}-\x{25FF}\x{2B00}-\x{2BFF}` +
			`\x{FE00}-\x{FE0F}\x{200D}\s\-–—•·°№#@!?\.,;:\*"'«»„"'']+`,
	)
	// safeFilenameRegex matches only characters safe for filenames on all platforms
	// Allowed: a-z, A-Z, 0-9, Cyrillic (а-яА-ЯёЁ), space, hyphen, underscore, period
	safeFilenameRegex = regexp.MustCompile(`[^a-zA-Z0-9\p{Cyrillic} \-_.]`)
)

func extractTitle(content string) string {
	text := content

	// Remove malformed custom emoji first
	text = malformedEmojiRegex.ReplaceAllString(text, "")

	// Remove custom emoji markdown
	text = customEmojiRegex.ReplaceAllString(text, "")

	// Remove HTML tags
	text = htmlTagRegex.ReplaceAllString(text, "")

	// Convert markdown links to just text
	text = markdownLinkRegex.ReplaceAllString(text, "$1")

	// Remove markdown formatting
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "__", "")
	text = strings.ReplaceAll(text, "_", "")
	text = strings.ReplaceAll(text, "`", "")

	// Remove timecodes
	text = timecodeRegex.ReplaceAllString(text, "")

	// Get first non-empty line
	var firstParagraph string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			firstParagraph = line
			break
		}
	}

	// Remove numbered emoji prefix
	firstParagraph = numberedEmojiPrefixRegex.ReplaceAllString(firstParagraph, "")

	// Strip leading junk repeatedly
	for {
		cleaned := leadingJunkRegex.ReplaceAllString(firstParagraph, "")
		cleaned = strings.TrimSpace(cleaned)
		if cleaned == firstParagraph {
			break
		}
		firstParagraph = cleaned
	}

	// Take first 7 words
	words := strings.Fields(firstParagraph)
	if len(words) > 7 {
		words = words[:7]
	}

	title := strings.Join(words, " ")

	// Keep only safe filename characters (whitelist approach)
	title = safeFilenameRegex.ReplaceAllString(title, "")

	// Collapse multiple spaces
	title = strings.Join(strings.Fields(title), " ")

	// Strip trailing punctuation
	title = strings.TrimRight(title, ".,;:!?…-–—")

	return strings.TrimSpace(title)
}

func generateFilename(title string, messageID int, usedFilenames map[string]bool) string {
	baseFilename := title + ".md"

	if !usedFilenames[baseFilename] {
		return baseFilename
	}

	return fmt.Sprintf("%s (%d).md", title, messageID)
}
