package model_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/graph/model"
)

func TestComputeDiff_Modified(t *testing.T) {
	d := model.ComputeDiff("old line\n", "new line\n")
	require.Equal(t, 1, d.AddedLines)
	require.Equal(t, 1, d.RemovedLines)
	require.Contains(t, d.Unified, "--- released")
	require.Contains(t, d.Unified, "+++ latest")
	require.Contains(t, d.Word, "{-old-}")
	require.Contains(t, d.Word, "{+new+}")
}

func TestComputeDiff_Identical(t *testing.T) {
	d := model.ComputeDiff("same content\n", "same content\n")
	require.Equal(t, 0, d.AddedLines)
	require.Equal(t, 0, d.RemovedLines)
	require.Equal(t, 0, d.ChangedWords)
	require.Empty(t, d.Unified)
	require.Equal(t, "same content", d.Word)
}

func TestComputeDiff_Added(t *testing.T) {
	d := model.ComputeDiff("", "hello world\n")
	require.Equal(t, 1, d.AddedLines)
	require.Equal(t, 0, d.RemovedLines)
	require.Contains(t, d.Word, "{+hello+}")
	require.Contains(t, d.Word, "{+world+}")
}

func TestComputeDiff_Removed(t *testing.T) {
	d := model.ComputeDiff("goodbye\n", "")
	require.Equal(t, 0, d.AddedLines)
	require.Equal(t, 1, d.RemovedLines)
	require.Contains(t, d.Word, "{-goodbye-}")
}

func TestComputeDiff_BothEmpty(t *testing.T) {
	d := model.ComputeDiff("", "")
	require.Equal(t, 0, d.AddedLines)
	require.Equal(t, 0, d.RemovedLines)
	require.Equal(t, 0, d.ChangedWords)
	require.Empty(t, d.Unified)
	require.Empty(t, d.Word)
}
