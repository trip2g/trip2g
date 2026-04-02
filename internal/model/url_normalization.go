package model

import (
	"net/url"
	"strings"

	"trip2g/internal/russkayalatinica"
	rl2 "trip2g/internal/russkayalatinica2"
	"trip2g/internal/translit"
)

// URLNormalizationMethod controls how Cyrillic note paths are transliterated into URL slugs.
type URLNormalizationMethod string

const (
	// URLNormCyrillic keeps Cyrillic characters as-is (browser percent-encodes them).
	URLNormCyrillic URLNormalizationMethod = "cyrillic"

	// URLNormSimpleTranslit uses basic ASCII transliteration (е→e, ж→zh, щ→sch).
	URLNormSimpleTranslit URLNormalizationMethod = "simple_translit"

	// URLNormRusskayaLatinica uses the old v1 transliteration package.
	// Deprecated: kept only for backward-compatible redirects. Not user-selectable.
	URLNormRusskayaLatinica URLNormalizationMethod = "russkayalatinica"

	// URLNormRusskayaLatinica2 uses the new v2 transliteration (context-aware, bidirectional).
	URLNormRusskayaLatinica2 URLNormalizationMethod = "russkayalatinica2"

	// DefaultURLNormalizationMethod is the default when no config is set.
	DefaultURLNormalizationMethod = URLNormRusskayaLatinica2
)

// Valid returns true if the method is user-selectable.
// russkayalatinica v1 is deprecated and not a valid user choice.
func (m URLNormalizationMethod) Valid() bool {
	switch m {
	case URLNormCyrillic, URLNormSimpleTranslit, URLNormRusskayaLatinica2:
		return true
	case URLNormRusskayaLatinica:
		// no-op: deprecated, not user-selectable
	}
	return false
}

// Apply transliterates permalinkOriginal using the chosen method.
func (m URLNormalizationMethod) Apply(permalinkOriginal string) string {
	switch m {
	case URLNormCyrillic:
		return permalinkOriginal
	case URLNormSimpleTranslit:
		return translit.ToASCII(permalinkOriginal)
	case URLNormRusskayaLatinica:
		return russkayalatinica.Translit(permalinkOriginal)
	case URLNormRusskayaLatinica2:
		return rl2.Translit(permalinkOriginal)
	default:
		return rl2.Translit(permalinkOriginal)
	}
}

// AllURLVariants computes all transliteration variants for a given permalinkOriginal.
// Returns a map from method to the resulting URL string.
// Entries that produce the same string as another method are still included
// (deduplication happens at registration time).
func AllURLVariants(permalinkOriginal string) map[URLNormalizationMethod]string {
	return map[URLNormalizationMethod]string{
		URLNormCyrillic:          URLNormCyrillic.Apply(permalinkOriginal),
		URLNormSimpleTranslit:    URLNormSimpleTranslit.Apply(permalinkOriginal),
		URLNormRusskayaLatinica:  URLNormRusskayaLatinica.Apply(permalinkOriginal),
		URLNormRusskayaLatinica2: URLNormRusskayaLatinica2.Apply(permalinkOriginal),
	}
}

// PermalinkEncode percent-encodes each segment of a decoded unicode path
// for use in HTTP output (Location headers, sitemap, template hrefs).
func PermalinkEncode(permalink string) string {
	parts := strings.Split(permalink, "/")
	encoded := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			encoded = append(encoded, "")
			continue
		}
		encoded = append(encoded, url.PathEscape(p))
	}
	return strings.Join(encoded, "/")
}
