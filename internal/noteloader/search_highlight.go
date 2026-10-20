package noteloader

import (
	"sort"
	"trip2g/internal/model"
	"unicode/utf8"

	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/highlight"
	simpleFragmenter "github.com/blevesearch/bleve/v2/search/highlight/fragmenter/simple"
)

// Same window and separator as bleve's simple highlighter, so snippets read as
// they did when bleve built them from stored fields.
const (
	highlightFragmentRunes = 200
	highlightSeparator     = "…"
)

// searchText is the plain text a note was indexed with. Bleve keeps only term
// vectors; the snippet is cut from this copy at query time.
type searchText struct {
	Title []byte
	Body  []byte
}

type termSpan struct {
	start, end int
	term       string
}

// highlightText cuts the best fragment of text for the locations the hit has
// in fields (the ru and en views of the same text) and returns it as segments,
// together with every matched span of text in order.
func highlightText(text []byte, locations search.FieldTermLocationMap, fields ...string) ([]model.HighlightSegment, []termSpan) {
	if len(text) == 0 {
		return nil, nil
	}

	perField := make([][]termSpan, 0, len(fields))
	for _, field := range fields {
		perField = append(perField, fieldSpans(text, locations[field]))
	}
	spans := mergeSpans(perField)

	ordered := make(highlight.TermLocations, len(spans))
	for i, s := range spans {
		ordered[i] = &highlight.TermLocation{Term: s.term, Start: s.start, End: s.end}
	}
	fragments := simpleFragmenter.NewFragmenter(highlightFragmentRunes).Fragment(text, ordered)

	// The first fragment with the top score: what bleve's priority queue pops
	// when it is filled in order and popped once.
	var best *highlight.Fragment
	bestScore := -1
	for _, f := range fragments {
		score := fragmentScore(f, perField)
		if score > bestScore {
			best, bestScore = f, score
		}
	}
	if best == nil {
		return nil, spans
	}

	return fragmentSegments(text, best.Start, best.End, spans), spans
}

// fieldSpans flattens one field's locations, sorted by start. Locations that
// do not fit text (the index moved ahead of the text snapshot) are dropped.
func fieldSpans(text []byte, tlm search.TermLocationMap) []termSpan {
	var spans []termSpan
	for term, locations := range tlm {
		for _, loc := range locations {
			start, end := int(loc.Start), int(loc.End)
			if start >= end || end > len(text) || !utf8.RuneStart(text[start]) {
				continue
			}
			if end < len(text) && !utf8.RuneStart(text[end]) {
				continue
			}
			spans = append(spans, termSpan{start: start, end: end, term: term})
		}
	}
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].start != spans[j].start {
			return spans[i].start < spans[j].start
		}
		return spans[i].end < spans[j].end
	})
	return spans
}

// mergeSpans unions the fields' spans: the same word matched by both
// analyzers appears once, and overlapping spans become one.
func mergeSpans(perField [][]termSpan) []termSpan {
	var all []termSpan
	for _, spans := range perField {
		all = append(all, spans...)
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].start < all[j].start
	})

	merged := make([]termSpan, 0, len(all))
	for _, s := range all {
		if n := len(merged); n > 0 && s.start < merged[n-1].end {
			if s.end > merged[n-1].end {
				merged[n-1].end = s.end
			}
			continue
		}
		merged = append(merged, s)
	}
	return merged
}

// fragmentScore counts the distinct terms inside f, per field, as bleve's
// simple scorer does, and takes the best field so the ru and en views of one
// word are not counted twice.
func fragmentScore(f *highlight.Fragment, perField [][]termSpan) int {
	best := 0
	var seen []string
	for _, spans := range perField {
		seen = seen[:0]
		first := sort.Search(len(spans), func(i int) bool {
			return spans[i].start >= f.Start
		})
		for _, s := range spans[first:] {
			if s.start >= f.End {
				break
			}
			if s.end > f.End || containsString(seen, s.term) {
				continue
			}
			seen = append(seen, s.term)
		}
		best = max(best, len(seen))
	}
	return best
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func fragmentSegments(text []byte, start, end int, spans []termSpan) []model.HighlightSegment {
	var segments []model.HighlightSegment
	add := func(s string, match bool) {
		if s == "" {
			return
		}
		if n := len(segments); n > 0 && !match && !segments[n-1].Match {
			segments[n-1].Text += s
			return
		}
		segments = append(segments, model.HighlightSegment{Text: s, Match: match})
	}

	if start != 0 {
		add(highlightSeparator, false)
	}
	curr := start
	for _, s := range spans {
		if s.start < curr {
			continue
		}
		if s.end > end {
			break
		}
		add(string(text[curr:s.start]), false)
		add(string(text[s.start:s.end]), true)
		curr = s.end
	}
	add(string(text[curr:end]), false)
	if end != len(text) {
		add(highlightSeparator, false)
	}
	return segments
}

// matchedTerms lists each matched surface form once, title first.
func matchedTerms(title []byte, titleSpans []termSpan, body []byte, bodySpans []termSpan) []string {
	var terms []string
	collect := func(text []byte, spans []termSpan) {
		for _, s := range spans {
			term := string(text[s.start:s.end])
			if !containsString(terms, term) {
				terms = append(terms, term)
			}
		}
	}
	collect(title, titleSpans)
	collect(body, bodySpans)
	return terms
}
