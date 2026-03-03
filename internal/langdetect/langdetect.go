package langdetect

import (
	"sort"
	"strconv"
	"strings"
)

// DetectPreferred returns the user's preferred language.
// Priority: cookie "lang" > Accept-Language header > empty string.
func DetectPreferred(cookieValue string, acceptLanguage string) string {
	if cookieValue != "" {
		return cookieValue
	}
	return ParseAcceptLanguage(acceptLanguage)
}

// ParseAcceptLanguage extracts the most preferred language from
// the Accept-Language header value.
// Example: "en-US,en;q=0.9,ru;q=0.8" -> "en"
// Returns empty string on empty or malformed input.
func ParseAcceptLanguage(header string) string {
	if header == "" {
		return ""
	}

	type langQ struct {
		lang    string
		quality float64
	}

	var langs []langQ

	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		tag := part
		quality := 1.0

		if idx := strings.Index(part, ";"); idx >= 0 {
			tag = strings.TrimSpace(part[:idx])
			param := strings.TrimSpace(part[idx+1:])
			if strings.HasPrefix(param, "q=") {
				q, err := strconv.ParseFloat(strings.TrimPrefix(param, "q="), 64)
				if err == nil {
					quality = q
				}
			}
		}

		// Ignore wildcard.
		if tag == "*" {
			continue
		}

		// Extract primary language tag (before '-').
		primary := tag
		if idx := strings.Index(tag, "-"); idx >= 0 {
			primary = tag[:idx]
		}
		primary = strings.ToLower(strings.TrimSpace(primary))
		if primary == "" {
			continue
		}

		langs = append(langs, langQ{lang: primary, quality: quality})
	}

	if len(langs) == 0 {
		return ""
	}

	sort.SliceStable(langs, func(i, j int) bool {
		return langs[i].quality > langs[j].quality
	})

	return langs[0].lang
}
