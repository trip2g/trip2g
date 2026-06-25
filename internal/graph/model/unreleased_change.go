package model

import (
	"strings"
	"sync"

	"github.com/pmezard/go-difflib/difflib"
)

// DiffResult is computed once per UnreleasedChange via sync.Once.
// Shared by the Stats, UnifiedDiff, and WordDiff resolver fields.
type DiffResult struct {
	Unified      string
	Word         string
	AddedLines   int
	RemovedLines int
	ChangedWords int
}

// UnreleasedChange is defined manually (not generated) to embed sync.Once for lazy
// single-computation diff. All three lazy fields call Diff() — it runs at most once.
type UnreleasedChange struct {
	Path            string
	PathID          int64
	Title           string
	ChangeType      NoteChangeType
	LiveVersionID   *int64
	LatestVersionID *int64
	OldContent      *string // nil when ADDED
	NewContent      *string // nil when REMOVED

	once sync.Once
	diff *DiffResult
}

// UnreleasedChangesConnection wraps the list of changes with aggregate metadata.
// Defined manually so Nodes can use pointer elements (required by sync.Once in UnreleasedChange).
type UnreleasedChangesConnection struct {
	TotalCount int
	Nodes      []*UnreleasedChange
}

// Diff computes and caches the diff result. Safe for concurrent resolver calls.
func (u *UnreleasedChange) Diff() *DiffResult {
	u.once.Do(func() {
		u.diff = computeDiff(derefStr(u.OldContent), derefStr(u.NewContent))
	})
	return u.diff
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ComputeDiff computes a diff between two strings, returning a DiffResult with
// unified diff, word diff, and change statistics.
func ComputeDiff(old, nw string) *DiffResult {
	return computeDiff(old, nw)
}

func computeDiff(old, nw string) *DiffResult {
	unified, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(old),
		B:        difflib.SplitLines(nw),
		FromFile: "released",
		ToFile:   "latest",
		Context:  3,
	})

	oldLines := difflib.SplitLines(old)
	newLines := difflib.SplitLines(nw)
	lm := difflib.NewMatcher(oldLines, newLines)
	var addedLines, removedLines int
	for _, op := range lm.GetOpCodes() {
		switch op.Tag {
		case 'i':
			addedLines += op.J2 - op.J1
		case 'd':
			removedLines += op.I2 - op.I1
		case 'r':
			addedLines += op.J2 - op.J1
			removedLines += op.I2 - op.I1
		}
	}

	oldWords := strings.Fields(old)
	newWords := strings.Fields(nw)
	wm := difflib.NewMatcher(oldWords, newWords)
	var changedWords int
	var sb strings.Builder
	for _, op := range wm.GetOpCodes() {
		switch op.Tag {
		case 'e':
			for _, w := range oldWords[op.I1:op.I2] {
				sb.WriteString(w)
				sb.WriteByte(' ')
			}
		case 'r':
			for _, w := range oldWords[op.I1:op.I2] {
				sb.WriteString("{-")
				sb.WriteString(w)
				sb.WriteString("-} ")
			}
			for _, w := range newWords[op.J1:op.J2] {
				sb.WriteString("{+")
				sb.WriteString(w)
				sb.WriteString("+} ")
			}
			changedWords += op.J2 - op.J1 + op.I2 - op.I1
		case 'i':
			for _, w := range newWords[op.J1:op.J2] {
				sb.WriteString("{+")
				sb.WriteString(w)
				sb.WriteString("+} ")
			}
			changedWords += op.J2 - op.J1
		case 'd':
			for _, w := range oldWords[op.I1:op.I2] {
				sb.WriteString("{-")
				sb.WriteString(w)
				sb.WriteString("-} ")
			}
			changedWords += op.I2 - op.I1
		}
	}

	return &DiffResult{
		Unified:      unified,
		Word:         strings.TrimSpace(sb.String()),
		AddedLines:   addedLines,
		RemovedLines: removedLines,
		ChangedWords: changedWords,
	}
}
