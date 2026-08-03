package rendernotepage

import (
	"strings"
	"unicode"

	"trip2g/internal/model"
)

// metaDescriptionMaxLen bounds the derived meta description. ~155 chars is the
// common search-snippet cut-off; a touch of slack avoids mid-word truncation.
const metaDescriptionMaxLen = 155

// deriveMetaDescription builds a fallback SEO meta description from a note's
// first paragraph when it declares no explicit description:. Returns "" when
// nothing usable can be extracted.
//
// Only a note whose body is already anonymous-visible may be summarized this
// way: the description lands in the <head> of every render, including the
// paywall and sign-in wall pages, which are served before access is granted.
// A closed note therefore gets a description only from explicit frontmatter —
// no explicit description means no description, and nothing but the title
// (shown on the wall by design) leaves the note.
func deriveMetaDescription(note *model.NoteView) string {
	if note == nil || note.PartialRenderer == nil || !note.IsAnonymouslyReadable() {
		return ""
	}
	intro := note.PartialRenderer.Introduce()
	text := collapseSpaces(stripHTMLTags(intro.ContentHTML))
	if text == "" {
		return ""
	}
	return truncateForMeta(text, metaDescriptionMaxLen)
}

// stripHTMLTags removes markup, keeping tag text. It is a light scanner (no full
// HTML parse) sufficient for the first-paragraph HTML the renderer produces.
func stripHTMLTags(s string) string {
	if !strings.ContainsRune(s, '<') {
		return s
	}
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// truncateForMeta cuts s to at most max runes, preferring a word boundary and
// appending an ellipsis when it actually shortens the text.
func truncateForMeta(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	cut := runes[:limit]
	if i := lastSpace(cut); i > 0 {
		cut = cut[:i]
	}
	return strings.TrimRightFunc(string(cut), unicode.IsSpace) + "…"
}

func lastSpace(runes []rune) int {
	for i := len(runes) - 1; i >= 0; i-- {
		if unicode.IsSpace(runes[i]) {
			return i
		}
	}
	return -1
}
