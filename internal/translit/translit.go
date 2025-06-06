// Package translit implements non-standard one-way string transliteration from Cyrillic to Latin.
package translit

import (
	"strings"
	"unicode"
)

// DefaultTable maps Russian Cyrillic runes to rough ASCII Latin equivalents.
var DefaultTable = map[rune]string{ //nolint:gochecknoglobals // it's ok
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e",
	'ё': "yo", 'ж': "zh", 'з': "z", 'и': "i", 'й': "j", 'к': "k",
	'л': "l", 'м': "m", 'н': "n", 'о': "o", 'п': "p", 'р': "r",
	'с': "s", 'т': "t", 'у': "u", 'ф': "f", 'х': "h", 'ц': "c",
	'ч': "ch", 'ш': "sh", 'щ': "sch", 'ъ': "'", 'ы': "y", 'ь': "",
	'э': "e", 'ю': "ju", 'я': "ja",
}

// ToLatin returns a transliterated string using the given mapping table.
// Unmapped runes are copied as-is.
func ToLatin(input string, table map[rune]string) string {
	var b strings.Builder
	runes := []rune(input)

	for i, r := range runes {
		tr, ok := table[unicode.ToLower(r)]
		if !ok {
			b.WriteRune(r)
			continue
		}
		if tr == "" {
			continue
		}

		// Preserve case for upper-case input
		if unicode.IsUpper(r) {
			nextIsLower := i+1 < len(runes) && !unicode.IsUpper(runes[i+1])
			if nextIsLower {
				t := []rune(tr)
				t[0] = unicode.ToUpper(t[0])
				b.WriteString(string(t))
			} else {
				b.WriteString(strings.ToUpper(tr))
			}
		} else {
			b.WriteString(tr)
		}
	}

	return b.String()
}

// ToASCII transliterates the input string using DefaultTable (Cyrillic → rough ASCII).
func ToASCII(s string) string {
	return ToLatin(s, DefaultTable)
}
