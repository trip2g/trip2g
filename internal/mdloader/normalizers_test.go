package mdloader

import (
	"testing"
)

func TestNormalizeWikilinks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple wikilink",
			input:    "This is a [[Wikilink]] test",
			expected: "This is a [[Wikilink|wikilink]] test",
		},
		{
			name:     "multiple wikilinks",
			input:    "Text [[First]] and [[Second]] links",
			expected: "Text [[First|first]] and [[Second|second]] links",
		},
		{
			name:     "wikilink with spaces",
			input:    "Text [[Link With Spaces]]",
			expected: "Text [[Link With Spaces|link with spaces]]",
		},
		{
			name:     "wikilink with mixed case",
			input:    "Text [[MixedCaseLink]]",
			expected: "Text [[MixedCaseLink|mixedcaselink]]",
		},
		{
			name:     "skip wikilink with dot before",
			input:    "Skip this .[[Wikilink]] one",
			expected: "Skip this .[[Wikilink]] one",
		},
		{
			name:     "skip wikilink at start of content",
			input:    "[[StartLink]] is here",
			expected: "[[StartLink]] is here",
		},
		{
			name:     "wikilink with existing pipe",
			input:    "[[Already|Has|Pipe]]",
			expected: "[[Already|Has|Pipe]]",
		},
		{
			name:     "mixed scenarios",
			input:    "Text [[Normal]] and .[[Skipped]] and [[Another]]",
			expected: "Text [[Normal|normal]] and .[[Skipped]] and [[Another|another]]",
		},
		{
			name:     "skip wikilink with numbers at start",
			input:    "[[Link123]]",
			expected: "[[Link123]]",
		},
		{
			name:     "wikilink with special chars",
			input:    "Text [[Link-With_Special.Chars]]",
			expected: "Text [[Link-With_Special.Chars|link-with_special.chars]]",
		},
		{
			name:     "wikilink with numbers mid-sentence",
			input:    "Here is [[Link123]] test",
			expected: "Here is [[Link123|link123]] test",
		},
		{
			name:     "multiple dots before wikilink",
			input:    "Skip ...[[Wikilink]] this",
			expected: "Skip ...[[Wikilink]] this",
		},
		{
			name:     "skip wikilink after newline (sentence start)",
			input:    "Line one.\n[[NewLine]]",
			expected: "Line one.\n[[NewLine]]",
		},
		{
			name:     "skip wikilink with cyrillic at start",
			input:    "[[Привет]] world",
			expected: "[[Привет]] world",
		},
		{
			name:     "empty wikilink",
			input:    "[[]]",
			expected: "[[]]",
		},
		{
			name:     "no wikilinks",
			input:    "Just plain text",
			expected: "Just plain text",
		},
		{
			name:     "incomplete wikilink",
			input:    "[[Incomplete",
			expected: "[[Incomplete",
		},
		{
			name:     "nested brackets",
			input:    "Text [[Outer [[Inner]] Link]]",
			expected: "Text [[Outer [[Inner|outer [[inner]] Link]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeWikilinks([]byte(tt.input))
			if string(result) != tt.expected {
				t.Errorf("normalizeWikilinks(%q) = %q, want %q", tt.input, string(result), tt.expected)
			}
		})
	}
}