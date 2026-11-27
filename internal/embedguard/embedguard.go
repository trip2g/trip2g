// Package embedguard marks every note HTML that is injected into another
// note's HTML, and removes the marked blocks a given viewer may not read at
// response time.
package embedguard

import (
	"bytes"
	"strconv"
	"strings"
)

const (
	openPrefix  = "<!--t2e:"
	closePrefix = "<!--/t2e:"
	markerEnd   = "-->"
)

// Wrap marks html as the content of the note with the given id.
func Wrap(id int64, html string) string {
	idText := strconv.FormatInt(id, 10)

	return openPrefix + idText + markerEnd + html + closePrefix + idText + markerEnd
}

// HasMarkers reports whether html carries any embed marker.
func HasMarkers(html string) bool {
	return strings.Contains(html, openPrefix) || strings.Contains(html, closePrefix)
}

// HasMarkersBytes is HasMarkers for a byte slice, without copying it.
func HasMarkersBytes(html []byte) bool {
	return bytes.Contains(html, []byte(openPrefix)) || bytes.Contains(html, []byte(closePrefix))
}

// Decide answers for one embedded note: keep its block, or replace the block
// with the returned HTML.
type Decide func(id int64) (keep bool, replacement string)

type marker struct {
	start int
	end   int
	id    int64
	open  bool
}

// Strip removes every marker and replaces each block whose note Decide
// refuses. A refused block takes everything nested inside it with it, and a
// block left open is refused to the end of the text.
func Strip(html string, decide Decide) string {
	if !HasMarkers(html) {
		return html
	}

	var out strings.Builder

	pos := 0

	for {
		m, ok := nextMarker(html, pos)
		if !ok {
			out.WriteString(html[pos:])
			return out.String()
		}

		out.WriteString(html[pos:m.start])
		pos = m.end

		if !m.open {
			continue
		}

		keep, replacement := decide(m.id)
		if keep {
			continue
		}

		out.WriteString(replacement)
		pos = skipBlock(html, pos)
	}
}

func skipBlock(html string, pos int) int {
	depth := 1

	for depth > 0 {
		m, ok := nextMarker(html, pos)
		if !ok {
			return len(html)
		}

		pos = m.end

		if m.open {
			depth++
		} else {
			depth--
		}
	}

	return pos
}

func nextMarker(html string, from int) (marker, bool) {
	for from < len(html) {
		idx := strings.Index(html[from:], "<!--")
		if idx < 0 {
			return marker{}, false
		}

		start := from + idx

		m, ok := parseMarker(html, start)
		if ok {
			return m, true
		}

		from = start + len("<!--")
	}

	return marker{}, false
}

func parseMarker(html string, start int) (marker, bool) {
	rest := html[start:]

	open := strings.HasPrefix(rest, openPrefix)
	if !open && !strings.HasPrefix(rest, closePrefix) {
		return marker{}, false
	}

	prefix := closePrefix
	if open {
		prefix = openPrefix
	}

	tail := rest[len(prefix):]

	digits := 0
	for digits < len(tail) && tail[digits] >= '0' && tail[digits] <= '9' {
		digits++
	}

	if digits == 0 || !strings.HasPrefix(tail[digits:], markerEnd) {
		return marker{}, false
	}

	id, err := strconv.ParseInt(tail[:digits], 10, 64)
	if err != nil {
		return marker{}, false
	}

	return marker{
		start: start,
		end:   start + len(prefix) + digits + len(markerEnd),
		id:    id,
		open:  open,
	}, true
}
