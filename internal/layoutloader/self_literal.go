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
// of its own expanded @lid/@did names. Such literals silently break the rename
// guarantee: the placeholders are optional, so a hardcoded "mesh-bar" survives a
// file rename while "@did" would follow it. Escaped @@lid/@@did stay literal in the
// raw source (they read "@@lid"), so they never match the expanded name.
//
// Rules:
//   - Only the file's OWN lid/did. Referencing another component's expanded name
//     (e.g. index.html yielding mesh_bar()) is correct composition, never flagged.
//   - JS/CSS strings are in scope: querySelector('.mesh-bar__nav') is exactly the
//     drift to catch — the expansion is textual everywhere.
//   - did matches with a BEM/CSS boundary so "mesh-bar" does not match inside
//     "mesh-barometer".
//   - lid matches only call/definition forms so "mesh_art" does not match inside
//     "mesh_article".
//   - Matches inside an HTML comment are skipped: comment prose (e.g. "the
//     kanban app") isn't an identifier, it's English text that happens to
//     contain the word.
//   - Matches inside an absolute URL (scheme://...) are skipped: the URL names
//     an external artifact (e.g. a release asset in another repo) whose name
//     must not track this file's name.
var (
	htmlCommentRe = regexp.MustCompile(`<!--[\s\S]*?-->`)
	absoluteURLRe = regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9+.-]*://[^\s'"<>()]+`)
)

// byteRange is a [start, end) byte offset range within scanned content.
type byteRange struct{ start, end int }

// skipRanges computes byte ranges that self-literal matches should ignore:
// HTML comments and absolute URL tokens (see scanSelfLiteral's Rules).
func skipRanges(content string) []byteRange {
	var ranges []byteRange
	for _, m := range htmlCommentRe.FindAllStringIndex(content, -1) {
		ranges = append(ranges, byteRange{m[0], m[1]})
	}
	for _, m := range absoluteURLRe.FindAllStringIndex(content, -1) {
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
	lid, did := derivePlaceholderIDs(sourceID)
	if lid == "" && did == "" {
		return nil
	}

	type hit struct {
		line int
		kind string // "@lid" or "@did"
		text string
	}
	var hits []hit

	skip := skipRanges(content)

	addHits := func(re *regexp.Regexp, kind, text string) {
		for _, m := range re.FindAllStringSubmatchIndex(content, -1) {
			start := m[2]
			if start < 0 || inSkipRange(skip, start) {
				continue
			}
			line := 1 + strings.Count(content[:start], "\n")
			hits = append(hits, hit{line: line, kind: kind, text: text})
		}
	}

	if did != "" {
		// non-word (or start) before; BEM/CSS boundary (or end) after.
		didRe := regexp.MustCompile(`(?:^|[^\w])(` + regexp.QuoteMeta(did) + `)(?:__|--|[\s'"` + "`" + `.#:\[)\{,;]|$)`)
		addHits(didRe, "@did", did)
	}

	if lid != "" {
		q := regexp.QuoteMeta(lid)
		// <lid>_ru( — RU variant call/definition (checked first so the _ru form wins).
		addHits(regexp.MustCompile(`(?:^|[^\w])(`+q+`)_ru\(`), "@lid", lid)
		// <lid>( — call or block definition.
		addHits(regexp.MustCompile(`(?:^|[^\w])(`+q+`)\(`), "@lid", lid)
		// _style_<lid> — the paired CSS block, boundary (or end) after.
		addHits(regexp.MustCompile(`_style_(`+q+`)(?:[^\w]|$)`), "@lid", lid)
		// yield <lid> — a yield reference, boundary (or end) after.
		addHits(regexp.MustCompile(`yield\s+(`+q+`)(?:[^\w]|$)`), "@lid", lid)
	}

	if len(hits) == 0 {
		return nil
	}

	// Dedupe by (line, kind): one warning per line per placeholder kind.
	seen := make(map[string]struct{})
	unique := hits[:0]
	for _, h := range hits {
		key := h.kind + ":" + strconv.Itoa(h.line)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, h)
	}

	sort.Slice(unique, func(i, j int) bool {
		if unique[i].line != unique[j].line {
			return unique[i].line < unique[j].line
		}
		return unique[i].kind < unique[j].kind
	})

	warnings := make([]model.NoteWarning, 0, len(unique))
	for _, h := range unique {
		warnings = append(warnings, model.NoteWarning{
			Level: model.NoteWarningWarning,
			Message: "layout " + sourceID + " line " + strconv.Itoa(h.line) +
				`: literal "` + h.text + `" matches this file's ` + h.kind +
				`; replace with ` + h.kind + " so renames keep working",
		})
	}
	return warnings
}
