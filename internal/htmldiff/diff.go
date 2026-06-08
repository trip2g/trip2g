package htmldiff

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

//nolint:gochecknoglobals // read-only lookup table of HTML void elements
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true,
	"embed": true, "hr": true, "img": true, "input": true,
	"link": true, "meta": true, "source": true, "wbr": true,
}

type tokState struct {
	tok   *html.Tokenizer
	depth int
	nth   int // 1-based index of current top-level child
}

// advance returns the next meaningful token type and its normalized string.
// Skips comments and doctypes. Tracks depth and nth-child at depth==0.
func (s *tokState) advance() (html.TokenType, string) {
	for {
		tt := s.tok.Next()
		switch tt { //nolint:exhaustive // only comments/doctype/error are special-cased here
		case html.CommentToken, html.DoctypeToken:
			continue
		case html.ErrorToken:
			return tt, ""
		}

		t := s.tok.Token()

		switch tt { //nolint:exhaustive // only tag tokens drive depth/nth tracking
		case html.StartTagToken:
			if s.depth == 0 {
				s.nth++
			}
			if !voidElements[t.Data] {
				s.depth++
			}
		case html.EndTagToken:
			if s.depth > 0 {
				s.depth--
			}
		}

		return tt, t.String()
	}
}

// FirstChangedBlock returns a CSS selector for the first top-level block
// element that differs between oldHTML and newHTML, e.g. ".content__body > :nth-child(3)".
// Returns "" if the HTML is identical.
// Both inputs are expected to be inner HTML fragments (not full documents).
func FirstChangedBlock(oldHTML, newHTML string) string {
	old := &tokState{tok: html.NewTokenizer(strings.NewReader(oldHTML))}
	nw := &tokState{tok: html.NewTokenizer(strings.NewReader(newHTML))}

	for {
		oldTT, oldStr := old.advance()
		newTT, newStr := nw.advance()

		if oldTT == html.ErrorToken && newTT == html.ErrorToken {
			return ""
		}

		if oldTT != newTT || oldStr != newStr {
			nth := nw.nth
			if nth == 0 {
				nth = 1
			}
			return fmt.Sprintf(".content__body > :nth-child(%d)", nth)
		}
	}
}
