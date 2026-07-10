// Package russkayalatinica2 implements bidirectional Cyrillic↔Latin
// transliteration using the Russkaya Latinica system.
//
// Faithfully ported from LinguisticKit (Swift) ScriptTable.ru.
// Key design decisions matching the reference implementation:
//   - Element types derived from cell table (not a separate map)
//   - Context matching via prefix/postfix element types
//   - Multi-pass case resolution algorithm
//   - Bidirectional engine: same cell table drives both directions
package russkayalatinica2

import (
	"strings"
	"unicode"
)

// Element & context types

type elementType uint8

const (
	consonant elementType = iota + 1
	vowel
	other
	nonLetter
)

type contextType uint8

const (
	ctxAny          contextType = iota // matches everything (zero value)
	ctxConsonant                       // == consonant
	ctxVowel                           // == vowel
	ctxOther                           // == other
	ctxNonLetter                       // == nonLetter
	ctxNonConsonant                    // != consonant
	ctxNonVowel                        // != vowel
	ctxNonOther                        // != other
	ctxLetter                          // != nonLetter
)

func (c contextType) matches(et elementType) bool {
	switch c {
	case ctxAny:
		return true
	case ctxConsonant:
		return et == consonant
	case ctxVowel:
		return et == vowel
	case ctxOther:
		return et == other
	case ctxNonLetter:
		return et == nonLetter
	case ctxNonConsonant:
		return et != consonant
	case ctxNonVowel:
		return et != vowel
	case ctxNonOther:
		return et != other
	case ctxLetter:
		return et != nonLetter
	}
	return false
}

// Engine

type engine struct {
	cyrlCells map[rune][]cell      // lowercase Cyrillic rune → cells (forward)
	cyrlTypes map[rune]elementType // Cyrillic rune → type (first cell wins)

	latnCells  map[string][]cell      // lowercase Latin key → cells (reverse)
	latnTypes  map[string]elementType // Latin key → type (first cell wins)
	maxLatnLen int                    // longest Latin key (for greedy match)
}

//nolint:gochecknoglobals
var global = newEngine()

func newEngine() *engine {
	e := &engine{
		cyrlCells: make(map[rune][]cell, 40),
		cyrlTypes: make(map[rune]elementType, 40),
		latnCells: make(map[string][]cell, 50),
		latnTypes: make(map[string]elementType, 50),
	}

	for i := range ruTable {
		c := &ruTable[i]

		e.cyrlCells[c.cyrl] = append(e.cyrlCells[c.cyrl], *c)
		if _, ok := e.cyrlTypes[c.cyrl]; !ok {
			e.cyrlTypes[c.cyrl] = c.typ
		}

		e.latnCells[c.latn] = append(e.latnCells[c.latn], *c)
		if _, ok := e.latnTypes[c.latn]; !ok {
			e.latnTypes[c.latn] = c.typ
		}

		if len(c.latn) > e.maxLatnLen {
			e.maxLatnLen = len(c.latn)
		}
	}

	return e
}

// runeType returns the element type of a Cyrillic rune (lowercase lookup).
func (e *engine) runeType(r rune) elementType {
	if t, ok := e.cyrlTypes[unicode.ToLower(r)]; ok {
		return t
	}
	if unicode.IsLetter(r) {
		return other
	}
	return nonLetter
}

// nextLatnType returns the element type of the first Latin element starting
// at runes[0], using longest-match lookup.
func (e *engine) nextLatnType(runes []rune) elementType {
	limit := e.maxLatnLen
	if limit > len(runes) {
		limit = len(runes)
	}
	for l := limit; l >= 1; l-- {
		key := strings.ToLower(string(runes[:l]))
		if t, ok := e.latnTypes[key]; ok {
			return t
		}
	}
	if len(runes) > 0 && unicode.IsLetter(runes[0]) {
		return other
	}
	return nonLetter
}

// latnElemType returns the element type for a Latin source string.
func (e *engine) latnElemType(s string) elementType {
	if t, ok := e.latnTypes[strings.ToLower(s)]; ok {
		return t
	}
	runes := []rune(s)
	if len(runes) > 0 && unicode.IsLetter(runes[0]) {
		return other
	}
	return nonLetter
}

// Forward: Cyrillic → Latin

type fwdElem struct {
	srcRune rune
	target  string // lowercase from table
}

// Translit converts Cyrillic text to Latin using Russkaya Latinica.
func Translit(s string) string { return global.translit(s) }

func (e *engine) translit(input string) string {
	if input == "" {
		return ""
	}

	runes := []rune(input)
	n := len(runes)
	elems := make([]fwdElem, 0, n)

	for i := 0; i < n; i++ {
		r := runes[i]
		key := unicode.ToLower(r)

		cells, ok := e.cyrlCells[key]
		if !ok {
			elems = append(elems, fwdElem{srcRune: r, target: string(r)})
			continue
		}

		prefixType := nonLetter
		if i > 0 {
			prefixType = e.runeType(runes[i-1])
		}

		postfixType := nonLetter
		if i+1 < n {
			postfixType = e.runeType(runes[i+1])
		}

		target := string(r) // fallback
		for ci := range cells {
			if cells[ci].prefix.matches(prefixType) && cells[ci].postfix.matches(postfixType) {
				target = cells[ci].latn
				break
			}
		}

		elems = append(elems, fwdElem{srcRune: r, target: target})
	}

	applyCaseFwd(elems)

	var b strings.Builder
	b.Grow(n * 2)
	for _, el := range elems {
		b.WriteString(el.target)
	}
	return b.String()
}

// Reverse: Latin → Cyrillic

type revElem struct {
	source string // original Latin (preserves case)
	target string // Cyrillic lowercase from table
}

// RevertTranslit converts Latin text back to Cyrillic.
func RevertTranslit(s string) string { return global.revert(s) }

func (e *engine) revert(input string) string {
	if input == "" {
		return ""
	}

	runes := []rune(input)
	n := len(runes)
	elems := make([]revElem, 0, n)

	for i := 0; i < n; {
		matched := false
		limit := e.maxLatnLen
		if limit > n-i {
			limit = n - i
		}

		for tryLen := limit; tryLen >= 1; tryLen-- {
			src := string(runes[i : i+tryLen])
			srcLower := strings.ToLower(src)

			cells, ok := e.latnCells[srcLower]
			if !ok {
				continue
			}

			prefixType := nonLetter
			if len(elems) > 0 {
				prefixType = e.latnElemType(elems[len(elems)-1].source)
			}

			postfixType := nonLetter
			if i+tryLen < n {
				postfixType = e.nextLatnType(runes[i+tryLen:])
			}

			for ci := range cells {
				if cells[ci].prefix.matches(prefixType) && cells[ci].postfix.matches(postfixType) {
					elems = append(elems, revElem{source: src, target: string(cells[ci].cyrl)})
					i += tryLen
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}

		if !matched {
			elems = append(elems, revElem{source: string(runes[i]), target: string(runes[i])})
			i++
		}
	}

	applyCaseRev(elems)

	var b strings.Builder
	b.Grow(n)
	for _, el := range elems {
		b.WriteString(el.target)
	}
	return b.String()
}

// Case resolution (Swift multi-pass algorithm)

type charCase int8

const (
	caseLower   charCase = iota // all letters lowercase
	caseUpper                   // all letters uppercase
	caseCapital                 // first letter uppercased, rest lower
	caseAmbig                   // single uppercase letter (needs resolution)
	caseUncased                 // non-letter / pass-through
)

func detectRuneCase(r rune) charCase {
	if !unicode.IsLetter(r) {
		return caseUncased
	}
	if unicode.IsUpper(r) {
		return caseAmbig
	}
	return caseLower
}

func detectStringCase(s string) charCase {
	runes := []rune(s)
	if len(runes) == 0 {
		return caseUncased
	}
	first := runes[0]
	if !unicode.IsLetter(first) {
		return caseUncased
	}
	if unicode.IsUpper(first) && len(runes) == 1 {
		return caseAmbig
	}

	allLower, allUpper, hasLower := true, true, false
	for _, r := range runes {
		if !unicode.IsLetter(r) {
			continue
		}
		if !unicode.IsLower(r) {
			allLower = false
		}
		if !unicode.IsUpper(r) {
			allUpper = false
		}
		if unicode.IsLower(r) {
			hasLower = true
		}
	}

	switch {
	case allLower:
		return caseLower
	case allUpper:
		return caseUpper
	case hasLower:
		return caseCapital
	default:
		return caseUncased
	}
}

// resolveCases implements the Swift multi-pass case resolution.
//
// Pass 1: Interior ambiguous chars — resolve by next neighbor.
// Pass 2: Consecutive upper/ambig sequences of length > 1 → all upper.
// Pass 3: Remaining ambiguous — look at nearest non-ambiguous cased neighbors.
func resolveCases(cases []charCase) {
	n := len(cases)
	if n == 0 {
		return
	}

	// Pass 1
	for i := 1; i < n-1; i++ {
		if cases[i] != caseAmbig {
			continue
		}
		switch cases[i+1] {
		case caseUncased:
			// leave ambiguous
		case caseLower:
			cases[i] = caseCapital
		case caseUpper, caseCapital, caseAmbig:
			cases[i] = caseUpper
		}
	}

	// Pass 2
	i := 0
	for i < n {
		if cases[i] != caseUpper && cases[i] != caseAmbig {
			i++
			continue
		}
		j := i + 1
		for j < n && (cases[j] == caseUpper || cases[j] == caseAmbig) {
			j++
		}
		if j-i > 1 {
			for k := i; k < j; k++ {
				cases[k] = caseUpper
			}
		}
		i = j
	}

	// Pass 3
	for i := 0; i < n; i++ {
		if cases[i] != caseAmbig {
			continue
		}
		hasUpper := false
		for j := i - 1; j >= 0; j-- {
			if cases[j] != caseAmbig && cases[j] != caseUncased {
				hasUpper = cases[j] == caseUpper
				break
			}
		}
		if !hasUpper {
			for j := i + 1; j < n; j++ {
				if cases[j] != caseAmbig && cases[j] != caseUncased {
					hasUpper = cases[j] == caseUpper
					break
				}
			}
		}
		if hasUpper {
			cases[i] = caseUpper
		} else {
			cases[i] = caseCapital
		}
	}
}

func applyCaseStr(s string, c charCase) string {
	switch c {
	case caseUpper:
		return strings.ToUpper(s)
	case caseCapital:
		runes := []rune(s)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
		}
		return string(runes)
	case caseLower, caseAmbig, caseUncased:
		// no-op: already lowercase or uncased
	}
	return s
}

func applyCaseFwd(elems []fwdElem) {
	n := len(elems)
	if n == 0 {
		return
	}
	cases := make([]charCase, n)
	for i := range elems {
		cases[i] = detectRuneCase(elems[i].srcRune)
	}
	resolveCases(cases)
	for i := range elems {
		elems[i].target = applyCaseStr(elems[i].target, cases[i])
	}
}

func applyCaseRev(elems []revElem) {
	n := len(elems)
	if n == 0 {
		return
	}
	cases := make([]charCase, n)
	for i := range elems {
		cases[i] = detectStringCase(elems[i].source)
	}
	resolveCases(cases)
	for i := range elems {
		elems[i].target = applyCaseStr(elems[i].target, cases[i])
	}
}
