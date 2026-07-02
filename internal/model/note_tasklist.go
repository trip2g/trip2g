package model

import (
	"bytes"
	"strings"
)

// HasTaskListItems reports whether the note's raw markdown contains GFM task
// list markers (- [ ] or - [x]). Drives conditional loading of the admin
// task-list widget script and the mount-point meta tag.
func (n *NoteView) HasTaskListItems() bool {
	return countTaskListMarkers(n.Content) > 0
}

// CountTaskListMarkers returns the number of GFM task-list markers in src,
// skipping markers inside fenced code blocks.
//
// A GFM task marker is a list item whose first non-space content is "[ ]" or
// "[x]" / "[X]" (the markdown spec allows any character, but trip2g only
// produces and toggles the two canonical forms).
func CountTaskListMarkers(src []byte) int {
	return countTaskListMarkers(src)
}

func countTaskListMarkers(src []byte) int {
	var count int
	inFence := false
	var fenceChar byte
	fenceLen := 0

	lines := bytes.Split(src, []byte("\n"))
	for _, rawLine := range lines {
		line := rawLine
		// Trim trailing \r so Windows line endings don't break matching.
		line = bytes.TrimRight(line, "\r")

		// Detect fenced code block open/close (``` or ~~~).
		stripped := strings.TrimLeft(string(line), " \t")
		if !inFence {
			if len(stripped) >= 3 {
				ch := stripped[0]
				if ch == '`' || ch == '~' {
					run := 0
					for run < len(stripped) && stripped[run] == ch {
						run++
					}
					if run >= 3 {
						inFence = true
						fenceChar = ch
						fenceLen = run
						continue
					}
				}
			}
		} else {
			// Inside fence: look for the closing fence (same char, >= same length, no trailing non-space).
			if len(stripped) >= fenceLen {
				ch := stripped[0]
				if ch == fenceChar {
					run := 0
					for run < len(stripped) && stripped[run] == ch {
						run++
					}
					if run >= fenceLen && strings.TrimLeft(stripped[run:], " \t") == "" {
						inFence = false
					}
				}
			}
			continue
		}

		// Outside a fence: match a task-list marker.
		// Allowed list prefixes: "- ", "* ", "+ " (with 0–3 leading spaces).
		sp := 0
		for sp < len(stripped) && (stripped[sp] == ' ' || stripped[sp] == '\t') {
			sp++
		}
		// stripped is already without leading whitespace (from TrimLeft above).
		rest := stripped
		if len(rest) < 5 {
			continue
		}
		// Must start with a list bullet.
		bullet := rest[0]
		if bullet != '-' && bullet != '*' && bullet != '+' {
			continue
		}
		if rest[1] != ' ' && rest[1] != '\t' {
			continue
		}
		// Skip any extra spaces after the bullet.
		i := 2
		for i < len(rest) && rest[i] == ' ' {
			i++
		}
		if i+3 > len(rest) {
			continue
		}
		// Match "[ ]" or "[x]" / "[X]".
		if rest[i] == '[' {
			inner := rest[i+1]
			if rest[i+2] == ']' && (inner == ' ' || inner == 'x' || inner == 'X') {
				count++
			}
		}
	}

	return count
}

// TaskListMarkerIndex maps a 0-based DOM checkbox index to the raw markdown
// source line that contains the corresponding task-list marker.
//
// Returns ("", -1) when n is out of range.
// Returns ("", -1) when src task-marker count != domCount (safety guard: mismatch
// means the client DOM and server source are out of sync — do NOT patch).
//
// The caller supplies domCount so the guard can be checked here; it must equal
// the number of <input type=checkbox> elements found in the rendered HTML.
func TaskListMarkerIndex(src []byte, domCount, n int) (line string, lineNumber int) {
	markers := collectTaskListMarkers(src)
	if len(markers) != domCount {
		return "", -1
	}
	if n < 0 || n >= len(markers) {
		return "", -1
	}
	return markers[n].line, markers[n].lineNo
}

type taskMarker struct {
	line   string // the complete raw line
	lineNo int    // 1-based line number in src
}

func collectTaskListMarkers(src []byte) []taskMarker {
	var out []taskMarker
	inFence := false
	var fenceChar byte
	fenceLen := 0

	lines := bytes.Split(src, []byte("\n"))
	for lineNo, rawLine := range lines {
		line := rawLine
		line = bytes.TrimRight(line, "\r")
		rawStr := string(rawLine)
		// Preserve original line (with \r if any) for find/replace fidelity.
		_ = rawStr

		stripped := strings.TrimLeft(string(line), " \t")
		if !inFence {
			if len(stripped) >= 3 {
				ch := stripped[0]
				if ch == '`' || ch == '~' {
					run := 0
					for run < len(stripped) && stripped[run] == ch {
						run++
					}
					if run >= 3 {
						inFence = true
						fenceChar = ch
						fenceLen = run
						continue
					}
				}
			}
		} else {
			if len(stripped) >= fenceLen {
				ch := stripped[0]
				if ch == fenceChar {
					run := 0
					for run < len(stripped) && stripped[run] == ch {
						run++
					}
					if run >= fenceLen && strings.TrimLeft(stripped[run:], " \t") == "" {
						inFence = false
					}
				}
			}
			continue
		}

		rest := stripped
		if len(rest) < 5 {
			continue
		}
		bullet := rest[0]
		if bullet != '-' && bullet != '*' && bullet != '+' {
			continue
		}
		if rest[1] != ' ' && rest[1] != '\t' {
			continue
		}
		i := 2
		for i < len(rest) && rest[i] == ' ' {
			i++
		}
		if i+3 > len(rest) {
			continue
		}
		if rest[i] == '[' {
			inner := rest[i+1]
			if rest[i+2] == ']' && (inner == ' ' || inner == 'x' || inner == 'X') {
				// Preserve original bytes (CR-stripped) as the canonical line.
				out = append(out, taskMarker{
					line:   string(line),
					lineNo: lineNo + 1,
				})
			}
		}
	}

	return out
}
