package layoutloader

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"trip2g/internal/model"
)

// derivePlaceholderIDs derives the @lid/@did expansions from a component's source ID.
// @lid = lodash id (underscores), used for Jet block names.
// @did = dash id (hyphens), used for BEM CSS class names.
// Examples: "/mesh/bar" → lid="mesh_bar", did="mesh-bar".
func derivePlaceholderIDs(sourceID string) (string, string) {
	base := strings.TrimPrefix(sourceID, "/")
	if idx := strings.LastIndex(base, "."); idx != -1 {
		base = base[:idx]
	}
	lid := strings.ReplaceAll(base, "/", "_")
	did := strings.ReplaceAll(base, "/", "-")
	return lid, did
}

// scanSelfLiteral scans a component's PRE-expansion source for literal occurrences
// of its own expanded @lid name. Such literals silently break the rename
// guarantee: the placeholder is optional, so a hardcoded "mesh_bar" survives a
// file rename while "@lid" would follow it. Escaped @@lid stays literal in the
// raw source (it reads "@@lid"), so it never matches the expanded name.
//
// @did is deliberately NOT checked. A dash id is an ordinary word in HTML, so
// every layout named after an element flagged its own markup — "/rss.html"
// warned about the RSS root element `<rss version="2.0">`, and table/main/nav
// layouts would do the same. The placeholder itself still expands as before.
//
// Rules:
//   - Only the file's OWN lid. Referencing another component's expanded name
//     (e.g. index.html yielding mesh_bar()) is correct composition, never flagged.
//   - lid matches only call/definition forms so "mesh_art" does not match inside
//     "mesh_article".
//   - Matches inside an HTML comment are skipped: usage documented in a comment
//     is prose, not a real call site.
var htmlCommentRe = regexp.MustCompile(`<!--[\s\S]*?-->`)

// byteRange is a [start, end) byte offset range within scanned content.
type byteRange struct{ start, end int }

// skipRanges computes byte ranges that self-literal matches should ignore:
// HTML comments (see scanSelfLiteral's Rules).
func skipRanges(content string) []byteRange {
	var ranges []byteRange
	for _, m := range htmlCommentRe.FindAllStringIndex(content, -1) {
		ranges = append(ranges, byteRange{m[0], m[1]})
	}
	return ranges
}

func inSkipRange(ranges []byteRange, pos int) bool {
	for _, r := range ranges {
		if pos >= r.start && pos < r.end {
			return true
		}
	}
	return false
}

func scanSelfLiteral(content, sourceID string) []model.NoteWarning {
	lid, _ := derivePlaceholderIDs(sourceID)
	if lid == "" {
		return nil
	}

	var lines []int
	skip := skipRanges(content)

	addHits := func(re *regexp.Regexp) {
		for _, m := range re.FindAllStringSubmatchIndex(content, -1) {
			start := m[2]
			if start < 0 || inSkipRange(skip, start) {
				continue
			}
			lines = append(lines, 1+strings.Count(content[:start], "\n"))
		}
	}

	q := regexp.QuoteMeta(lid)
	// <lid>_ru( — RU variant call/definition (checked first so the _ru form wins).
	addHits(regexp.MustCompile(`(?:^|[^\w])(` + q + `)_ru\(`))
	// <lid>( — call or block definition.
	addHits(regexp.MustCompile(`(?:^|[^\w])(` + q + `)\(`))
	// _style_<lid> — the paired CSS block, boundary (or end) after.
	addHits(regexp.MustCompile(`_style_(` + q + `)(?:[^\w]|$)`))
	// yield <lid> — a yield reference, boundary (or end) after.
	addHits(regexp.MustCompile(`yield\s+(` + q + `)(?:[^\w]|$)`))

	if len(lines) == 0 {
		return nil
	}

	// One warning per line, however many literals it holds.
	sort.Ints(lines)
	warnings := make([]model.NoteWarning, 0, len(lines))
	for i, line := range lines {
		if i > 0 && line == lines[i-1] {
			continue
		}
		warnings = append(warnings, model.NoteWarning{
			Level: model.NoteWarningWarning,
			Message: "layout " + sourceID + " line " + strconv.Itoa(line) +
				`: literal "` + lid + `" matches this file's @lid; replace with @lid so renames keep working`,
		})
	}
	return warnings
}
