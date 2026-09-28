package enclavefix

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	youtubeIDPattern  = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	bilibiliIDPattern = regexp.MustCompile(`^(?:BV[A-Za-z0-9]{10}|av[0-9]+)$`)
	imageSizePattern  = regexp.MustCompile(`^[0-9]+(?:%|px|rem)?$`)
)

// ValidEmbedID reports whether an embed identifier is safe for the provider's
// downstream HTML/JavaScript template. mdloader renders the enclave nodes.
func ValidEmbedID(provider, id string) bool {
	switch provider {
	case "youtube":
		return youtubeIDPattern.MatchString(id)
	case "bilibili":
		return bilibiliIDPattern.MatchString(id)
	case "tradingview":
		if id == "" || strings.ContainsAny(id, `"\<>`) {
			return false
		}
		for _, r := range id {
			if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// SafeImageDimension returns a dimension only when it is safe to place in an
// HTML attribute or inline style. Invalid values are dropped.
func SafeImageDimension(value string) string {
	if imageSizePattern.MatchString(value) {
		return value
	}
	return ""
}
