package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWriteHolder_AcquireRelease(t *testing.T) {
	h := NewWriteHolder(true)

	require.Empty(t, h.Snapshot(), "fresh holder must be free")

	release := h.Acquire("tx pushNotes")
	infos := h.Snapshot()
	require.Len(t, infos, 1)
	require.Equal(t, "tx pushNotes", infos[0].Label)
	require.Contains(t, infos[0].Stack, "TestWriteHolder_AcquireRelease",
		"stack must point at the acquiring goroutine")
	require.GreaterOrEqual(t, infos[0].HeldFor, time.Duration(0))

	release()
	require.Empty(t, h.Snapshot(), "release must clear the entry")

	// double release is a no-op
	release()
	require.Empty(t, h.Snapshot())
}

func TestWriteHolder_OldestFirst(t *testing.T) {
	h := NewWriteHolder(true)

	rel1 := h.Acquire("first")
	time.Sleep(time.Millisecond)
	rel2 := h.Acquire("second")

	infos := h.Snapshot()
	require.Len(t, infos, 2)
	require.Equal(t, "first", infos[0].Label, "oldest entry (the likely holder) comes first")
	require.Equal(t, "second", infos[1].Label)

	rel1()
	infos = h.Snapshot()
	require.Len(t, infos, 1)
	require.Equal(t, "second", infos[0].Label)
	rel2()
}

func TestWriteHolder_Disabled(t *testing.T) {
	h := NewWriteHolder(false)

	release := h.Acquire("tx x")
	require.Empty(t, h.Snapshot(), "disabled holder must not capture anything")
	release()
}

func TestWriteHolder_NilSafe(t *testing.T) {
	var h *WriteHolder

	release := h.Acquire("x")
	release()
	require.Empty(t, h.Snapshot())
}
