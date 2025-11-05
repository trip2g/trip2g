package telegram

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

// HandleRateLimit checks if error is "Too Many Requests" and returns retry delay.
func HandleRateLimit(err error) (bool, time.Duration) {
	if err == nil {
		return false, 0
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Too Many Requests") {
		return false, 0
	}

	// Try to parse "retry after X" from error message
	re := regexp.MustCompile(`retry after (\d+)`)
	matches := re.FindStringSubmatch(errMsg)

	seconds := 10 // default delay
	if len(matches) > 1 {
		parsed, parseErr := strconv.Atoi(matches[1])
		if parseErr == nil {
			seconds = parsed
		}
	}

	// Add +1 second to the delay
	return true, time.Duration(seconds+1) * time.Second
}

// GetTelegramLength returns the length of text as counted by Telegram.
// Telegram counts message length in UTF-16 code units, not bytes.
func GetTelegramLength(text string) int {
	// Convert string to []rune (Unicode code points)
	runes := []rune(text)

	// Encode to UTF-16
	utf16Encoded := utf16.Encode(runes)

	// Return the number of UTF-16 code units
	return len(utf16Encoded)
}

// TruncateContent truncates content to Telegram limits.
// Text messages: 4096 chars, Photo captions: 1024 chars.
// Reserve 3 chars for '...' if truncation is needed.
// Removes unclosed HTML tags after truncation.
func TruncateContent(content string, hasImages bool) string {
	maxLength := 4096
	if hasImages {
		maxLength = 1024
	}

	// Reserve 3 chars for ellipsis
	maxLength -= 3

	if GetTelegramLength(content) <= maxLength {
		return content
	}

	// Truncate by runes to avoid cutting in the middle of a character
	runes := []rune(content)
	utf16Encoded := utf16.Encode(runes)

	// Truncate to maxLength UTF-16 code units
	if len(utf16Encoded) > maxLength {
		utf16Encoded = utf16Encoded[:maxLength]
	}

	// Decode back to runes
	truncatedRunes := utf16.Decode(utf16Encoded)
	truncated := string(truncatedRunes)

	// Remove all unclosed HTML tags (may need multiple passes for nested tags)
	for {
		cleaned := removeUnclampedTags(truncated)
		if cleaned == truncated {
			// No more unclosed tags
			break
		}
		truncated = cleaned
	}

	// Convert back to string and add ellipsis
	return truncated + "..."
}

// tagPos represents an open HTML tag position.
type tagPos struct {
	name  string
	start int
}

// removeUnclampedTags removes unclosed HTML tags from the end of the string.
func removeUnclampedTags(content string) string {
	// Check if content ends with an incomplete tag (e.g., "<b", "<code", etc.)
	lastLt := strings.LastIndex(content, "<")
	if lastLt == -1 {
		return content
	}

	lastGt := strings.LastIndex(content, ">")

	// If last '<' is after last '>', we have an incomplete tag - remove it
	if lastLt > lastGt {
		content = content[:lastLt]
	}

	// Track open tags
	var openTags []tagPos
	i := 0

	for i < len(content) {
		if content[i] == '<' {
			nextI, found := processTag(content, i, &openTags)
			if !found {
				break
			}
			i = nextI
		} else {
			i++
		}
	}

	// If there are unclosed tags, remove the last one
	if len(openTags) > 0 {
		lastTag := openTags[len(openTags)-1]
		content = content[:lastTag.start]
	}

	return content
}

// processTag processes a single HTML tag and updates the openTags slice.
// Returns the next index to continue from and whether processing should continue.
func processTag(content string, i int, openTags *[]tagPos) (int, bool) {
	// Find the end of tag
	end := strings.IndexByte(content[i:], '>')
	if end == -1 {
		// Incomplete tag at the end
		return i, false
	}
	end += i

	tagContent := content[i+1 : end]

	// Skip self-closing tags and special tags
	if strings.HasPrefix(tagContent, "!") ||
		strings.HasSuffix(tagContent, "/") ||
		strings.HasPrefix(tagContent, "?") {
		return end + 1, true
	}

	// Closing tag
	if strings.HasPrefix(tagContent, "/") {
		tagName := tagContent[1:]
		// Find matching opening tag
		for j := len(*openTags) - 1; j >= 0; j-- {
			if (*openTags)[j].name == tagName {
				// Remove this tag from open tags
				*openTags = append((*openTags)[:j], (*openTags)[j+1:]...)
				break
			}
		}
		return end + 1, true
	}

	// Opening tag - extract tag name (before space or >)
	tagName := tagContent
	if spaceIdx := strings.IndexAny(tagContent, " \t\n\r"); spaceIdx != -1 {
		tagName = tagContent[:spaceIdx]
	}

	// Add to open tags
	*openTags = append(*openTags, tagPos{name: tagName, start: i})
	return end + 1, true
}
