package model

import (
	"html"
	"strings"
)

// HighlightSegment is a piece of a search snippet; Match marks a matched word.
type HighlightSegment struct {
	Text  string
	Match bool
}

// PlainSegments wraps text that has no matches; empty text gives no segments.
func PlainSegments(text string) []HighlightSegment {
	if text == "" {
		return nil
	}
	return []HighlightSegment{{Text: text}}
}

// HighlightHTML renders segments the way the bleve html highlighter did:
// escaped text with <mark> around matches.
func HighlightHTML(segments []HighlightSegment) string {
	var sb strings.Builder
	for _, s := range segments {
		if s.Match {
			sb.WriteString("<mark>")
			sb.WriteString(html.EscapeString(s.Text))
			sb.WriteString("</mark>")
			continue
		}
		sb.WriteString(html.EscapeString(s.Text))
	}
	return sb.String()
}

// SegmentsText joins the segments' text.
func SegmentsText(segments []HighlightSegment) string {
	var sb strings.Builder
	for _, s := range segments {
		sb.WriteString(s.Text)
	}
	return sb.String()
}
