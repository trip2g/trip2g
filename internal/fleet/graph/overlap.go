// Package graph derives the static fleet dependency graph from parsed role
// notes: which role's writes can trigger or feed which other role. Pure
// functions, no IO — the caller supplies roles and (optionally) registry
// markers.
package graph

import "strings"

// maxBraceExpansions caps brace expansion; beyond it the pattern is treated as
// unknown (conservative overlap = true).
const maxBraceExpansions = 64

// Overlap reports whether two doublestar patterns can match a common path,
// i.e. ∃ path: doublestar.Match(p, path) && doublestar.Match(q, path).
// Exact for the glob subset used in role frontmatter (literals, *, ?, [...],
// ** segments, {a,b} braces); unknown or pathological constructs return true
// (sound toward true: a spurious edge is a review nuisance, a missing edge
// hides a live trigger chain).
func Overlap(p, q string) bool {
	ps, ok := expandBraces(p)
	if !ok {
		return true
	}
	qs, ok := expandBraces(q)
	if !ok {
		return true
	}
	for _, pe := range ps {
		for _, qe := range qs {
			if overlapExpanded(pe, qe) {
				return true
			}
		}
	}
	return false
}

// overlapExpanded checks overlap of two brace-free patterns by segment
// recursion, treating "**" as zero-or-more segments.
func overlapExpanded(p, q string) bool {
	ps := strings.Split(p, "/")
	qs := strings.Split(q, "/")
	type key struct{ i, j int }
	memo := map[key]bool{}
	var rec func(i, j int) bool
	rec = func(i, j int) bool {
		k := key{i, j}
		if v, seen := memo[k]; seen {
			return v
		}
		memo[k] = false // cycle guard; overwritten below
		var res bool
		switch {
		case i == len(ps) && j == len(qs):
			res = true
		case i < len(ps) && ps[i] == "**":
			res = rec(i+1, j) || (j < len(qs) && rec(i, j+1))
		case j < len(qs) && qs[j] == "**":
			res = rec(i, j+1) || (i < len(ps) && rec(i+1, j))
		case i == len(ps) || j == len(qs):
			res = false // one exhausted, other head is a non-** segment
		default:
			res = segOverlap(ps[i], qs[j]) && rec(i+1, j+1)
		}
		memo[k] = res
		return res
	}
	return rec(0, 0)
}

// expandBraces expands {a,b} alternatives (recursively) into brace-free
// patterns. Returns ok=false when expansion explodes or braces are malformed.
func expandBraces(p string) ([]string, bool) {
	open := strings.IndexByte(p, '{')
	if open == -1 || isEscaped(p, open) {
		// No (unescaped) brace: also check for a stray '}' escape-neutrally —
		// doublestar treats it literally, and so do we by returning as-is.
		return []string{p}, true
	}
	depth := 0
	var alts []string
	start := open + 1
	for i := open; i < len(p); i++ {
		if isEscaped(p, i) {
			continue
		}
		switch p[i] {
		case '{':
			depth++
		case ',':
			if depth == 1 {
				alts = append(alts, p[start:i])
				start = i + 1
			}
		case '}':
			depth--
			if depth == 0 {
				alts = append(alts, p[start:i])
				var out []string
				for _, alt := range alts {
					sub, ok := expandBraces(p[:open] + alt + p[i+1:])
					if !ok {
						return nil, false
					}
					out = append(out, sub...)
					if len(out) > maxBraceExpansions {
						return nil, false
					}
				}
				return out, true
			}
		}
	}
	return nil, false // unmatched '{' — unknown construct
}

// isEscaped reports whether p[i] is preceded by an odd number of backslashes.
func isEscaped(p string, i int) bool {
	n := 0
	for j := i - 1; j >= 0 && p[j] == '\\'; j-- {
		n++
	}
	return n%2 == 1
}

// segToken is one within-segment pattern element: a star (any run of chars) or
// a one-character set.
type segToken struct {
	star bool
	set  charSet
}

// charSet is a one-character matcher: ? (any), a literal rune, or a [...] class.
type charSet struct {
	any     bool
	lit     rune
	isClass bool
	negated bool
	ranges  [][2]rune
}

// segOverlap reports whether two within-segment patterns can match a common
// string. Unknown constructs (unclosed class, trailing backslash) → true.
func segOverlap(a, b string) bool {
	at, ok := tokenize(a)
	if !ok {
		return true
	}
	bt, ok := tokenize(b)
	if !ok {
		return true
	}
	type key struct{ i, j int }
	memo := map[key]bool{}
	var rec func(i, j int) bool
	rec = func(i, j int) bool {
		k := key{i, j}
		if v, seen := memo[k]; seen {
			return v
		}
		memo[k] = false
		var res bool
		switch {
		case i == len(at) && j == len(bt):
			res = true
		case i < len(at) && at[i].star:
			res = rec(i+1, j) || (j < len(bt) && rec(i, j+1))
		case j < len(bt) && bt[j].star:
			res = rec(i, j+1) || (i < len(at) && rec(i+1, j))
		case i == len(at) || j == len(bt):
			res = false
		default:
			res = setsIntersect(at[i].set, bt[j].set) && rec(i+1, j+1)
		}
		memo[k] = res
		return res
	}
	return rec(0, 0)
}

// tokenize parses a within-segment pattern into tokens. ok=false on unknown
// constructs. A non-full-segment "**" behaves like "*" (doublestar semantics).
func tokenize(s string) ([]segToken, bool) {
	var out []segToken
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		switch rs[i] {
		case '*':
			if len(out) == 0 || !out[len(out)-1].star {
				out = append(out, segToken{star: true})
			}
		case '?':
			out = append(out, segToken{set: charSet{any: true}})
		case '\\':
			if i+1 >= len(rs) {
				return nil, false
			}
			i++
			out = append(out, segToken{set: charSet{lit: rs[i]}})
		case '[':
			set, next, ok := parseClass(rs, i)
			if !ok {
				return nil, false
			}
			out = append(out, segToken{set: set})
			i = next
		default:
			out = append(out, segToken{set: charSet{lit: rs[i]}})
		}
	}
	return out, true
}

// parseClass parses a [...] class starting at rs[i]=='['. Returns the set and
// the index of the closing ']'.
func parseClass(rs []rune, i int) (charSet, int, bool) {
	set := charSet{isClass: true}
	j := i + 1
	if j < len(rs) && (rs[j] == '!' || rs[j] == '^') {
		set.negated = true
		j++
	}
	first := true
	for ; j < len(rs); j++ {
		r := rs[j]
		if r == ']' && !first {
			if len(set.ranges) == 0 {
				return charSet{}, 0, false // empty class — treat as unknown
			}
			return set, j, true
		}
		first = false
		if r == '\\' {
			if j+1 >= len(rs) {
				return charSet{}, 0, false
			}
			j++
			r = rs[j]
		}
		lo, hi := r, r
		if j+2 < len(rs) && rs[j+1] == '-' && rs[j+2] != ']' {
			hi = rs[j+2]
			if rs[j+2] == '\\' {
				if j+3 >= len(rs) {
					return charSet{}, 0, false
				}
				hi = rs[j+3]
				j++
			}
			j += 2
		}
		set.ranges = append(set.ranges, [2]rune{lo, hi})
	}
	return charSet{}, 0, false // unclosed class
}

// setsIntersect reports whether two one-character sets share a rune.
// Conservative (true) when both are negated classes.
func setsIntersect(a, b charSet) bool {
	if a.any || b.any {
		return true
	}
	if !a.isClass && !b.isClass {
		return a.lit == b.lit
	}
	if !a.isClass {
		return classMatches(b, a.lit)
	}
	if !b.isClass {
		return classMatches(a, b.lit)
	}
	switch {
	case !a.negated && !b.negated:
		for _, ra := range a.ranges {
			for _, rb := range b.ranges {
				if ra[0] <= rb[1] && rb[0] <= ra[1] {
					return true
				}
			}
		}
		return false
	case a.negated && b.negated:
		return true // two negated classes virtually always intersect
	case a.negated:
		return positiveEscapesNegated(b, a)
	default:
		return positiveEscapesNegated(a, b)
	}
}

// classMatches reports whether class c matches rune r.
func classMatches(c charSet, r rune) bool {
	in := false
	for _, rg := range c.ranges {
		if rg[0] <= r && r <= rg[1] {
			in = true
			break
		}
	}
	return in != c.negated
}

// positiveEscapesNegated reports whether positive class pos contains a rune the
// negated class neg also matches (i.e. a rune outside neg's underlying ranges).
func positiveEscapesNegated(pos, neg charSet) bool {
	for _, rg := range pos.ranges {
		if rg[1]-rg[0] > 256 {
			return true // huge range certainly escapes any realistic exclusion
		}
		for r := rg[0]; r <= rg[1]; r++ {
			if classMatches(neg, r) {
				return true
			}
		}
	}
	return false
}
