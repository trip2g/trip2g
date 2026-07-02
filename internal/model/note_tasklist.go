package model

import (
	"bytes"
	"sort"

	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

// NoteViewTaskItem is one GFM task checkbox extracted from the note AST, in
// document order. Index is the 0-based document-order position and equals the
// position of the rendered <input type=checkbox> in the note HTML, so a client
// can map DOM checkbox i → TaskList[i]. Line is the 1-based line number in the
// raw note source (frontmatter included), Text is that exact source line
// (CR-stripped).
type NoteViewTaskItem struct {
	Index   int
	Line    int
	Checked bool
	Text    string
}

// HasTaskListItems reports whether the note contains GFM task checkboxes.
// Drives the conditional admin task-list mount on note pages.
func (n *NoteView) HasTaskListItems() bool {
	return len(n.TaskList) > 0
}

// extractTaskList walks the AST and collects every GFM task checkbox with its
// source position. Runs during the load pipeline (like extractHeadings), so
// fenced code blocks and plain text never contribute items — only real
// checkbox nodes that goldmark will render do.
//
// If any checkbox lacks source position info the whole list is dropped: a
// partial list would break the index↔DOM mapping and risk patching the wrong
// line, while an empty list just leaves the checkboxes read-only.
func (n *NoteView) extractTaskList() {
	n.TaskList = nil

	if n.ast == nil {
		return
	}

	var lineStarts []int // built lazily on the first checkbox

	incomplete := false
	_ = ast.Walk(n.ast, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		cb, ok := node.(*extast.TaskCheckBox)
		if !ok {
			return ast.WalkContinue, nil
		}

		// The checkbox node carries no source segment; its parent text block
		// does. The first line segment starts at the item's text content —
		// within the source line that holds the "[ ]"/"[x]" marker.
		parent := cb.Parent()
		if parent == nil || parent.Lines().Len() == 0 {
			incomplete = true
			return ast.WalkStop, nil
		}

		if lineStarts == nil {
			lineStarts = buildLineStarts(n.Content)
		}

		offset := parent.Lines().At(0).Start
		lineNo, text := lineAt(n.Content, lineStarts, offset)

		n.TaskList = append(n.TaskList, NoteViewTaskItem{
			Index:   len(n.TaskList),
			Line:    lineNo,
			Checked: cb.IsChecked,
			Text:    text,
		})

		return ast.WalkContinue, nil
	})

	if incomplete {
		n.TaskList = nil
	}
}

// buildLineStarts returns the byte offset of the start of every line in src.
func buildLineStarts(src []byte) []int {
	starts := []int{0}
	for i, b := range src {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// lineAt maps a byte offset to its 1-based line number and the full line text
// (CR-stripped).
func lineAt(src []byte, lineStarts []int, offset int) (int, string) {
	idx := sort.Search(len(lineStarts), func(i int) bool {
		return lineStarts[i] > offset
	}) - 1

	start := lineStarts[idx]
	end := len(src)
	if idx+1 < len(lineStarts) {
		end = lineStarts[idx+1] - 1 // drop the trailing \n
	}
	line := bytes.TrimRight(src[start:end], "\r")

	return idx + 1, string(line)
}
