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

	// Convert back to string and add ellipsis
	return string(truncatedRunes) + "..."
}
