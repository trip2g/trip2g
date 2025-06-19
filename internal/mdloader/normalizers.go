package mdloader

import (
	"strings"
)

// NormalizeWikilinks replaces [[Wikilink]] with [[Wikilink|wikilink]]
// It skips wikilinks that start a sentence (at position 0 or after a dot)
func NormalizeWikilinks(content []byte) []byte {
	// Process the content byte by byte to handle sentence start correctly
	var result []byte
	i := 0

	for i < len(content) {
		// Look for [[
		if i+1 < len(content) && content[i] == '[' && content[i+1] == '[' {
			// Check if this starts a sentence (at beginning or after dot + optional whitespace)
			skipThis := false
			if i == 0 {
				// At the very beginning of content
				skipThis = true
			} else {
				// Look backwards for a dot
				for j := i - 1; j >= 0; j-- {
					if content[j] == '.' {
						skipThis = true
						break
					} else if content[j] != ' ' && content[j] != '\t' && content[j] != '\n' && content[j] != '\r' {
						// Found a non-whitespace, non-dot character
						break
					}
				}
			}

			// Find the closing ]]
			closeIdx := -1
			for j := i + 2; j < len(content)-1; j++ {
				if content[j] == ']' && content[j+1] == ']' {
					closeIdx = j
					break
				}
			}

			if closeIdx != -1 && !skipThis {
				// Extract link text
				linkText := string(content[i+2 : closeIdx])

				// Skip empty links
				if linkText == "" {
					// Just copy the empty link as-is
					result = append(result, content[i:closeIdx+2]...)
					i = closeIdx + 2
					continue
				}

				// Check if it already has a pipe
				if !strings.Contains(linkText, "|") {
					// Replace with [[Link|link]]
					result = append(result, content[i:i+2]...)
					result = append(result, []byte(linkText)...)
					result = append(result, '|')
					result = append(result, []byte(strings.ToLower(linkText))...)
					result = append(result, content[closeIdx:closeIdx+2]...)
					i = closeIdx + 2
					continue
				}
			}
		}

		// Copy current byte
		result = append(result, content[i])
		i++
	}

	return result
}
