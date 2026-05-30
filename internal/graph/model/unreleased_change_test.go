package model_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/graph/model"
)

func strPtr(s string) *string { return &s }

func TestDiff_Modified_LineAndWordStats(t *testing.T) {
	ch := &model.UnreleasedChange{
		ChangeType: model.NoteChangeTypeModified,
		OldContent: strPtr("old line\n"),
		NewContent: strPtr("new line\n"),
	}

	d := ch.Diff()
	require.Equal(t, 1, d.AddedLines)
	require.Equal(t, 1, d.RemovedLines)
	require.Contains(t, d.Unified, "--- released")
	require.Contains(t, d.Unified, "+++ latest")
	require.Contains(t, d.Word, "{-old-}")
	require.Contains(t, d.Word, "{+new+}")
}

func TestDiff_NilOldContent_Added(t *testing.T) {
	ch := &model.UnreleasedChange{
		ChangeType: model.NoteChangeTypeAdded,
		OldContent: nil,
		NewContent: strPtr("hello world\n"),
	}

	d := ch.Diff()
	require.Equal(t, 1, d.AddedLines)
	require.Equal(t, 0, d.RemovedLines)
	require.Contains(t, d.Word, "{+hello+}")
}

func TestDiff_NilNewContent_Removed(t *testing.T) {
	ch := &model.UnreleasedChange{
		ChangeType: model.NoteChangeTypeRemoved,
		OldContent: strPtr("goodbye\n"),
		NewContent: nil,
	}

	d := ch.Diff()
	require.Equal(t, 0, d.AddedLines)
	require.Equal(t, 1, d.RemovedLines)
	require.Contains(t, d.Word, "{-goodbye-}")
}

func TestDiff_CalledOnlyOnce(t *testing.T) {
	var calls atomic.Int64
	ch := &model.UnreleasedChange{
		ChangeType: model.NoteChangeTypeModified,
		OldContent: strPtr("a\n"),
		NewContent: strPtr("b\n"),
	}

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch.Diff()
			calls.Add(1)
		}()
	}
	wg.Wait()

	d1 := ch.Diff()
	d2 := ch.Diff()
	require.Same(t, d1, d2)
}
