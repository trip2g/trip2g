package mcp

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/model"
)

func TestVectorResultsFromChunksUsesBestMatchingChunk(t *testing.T) {
	sourceNote := &model.NoteView{
		Path:      "Книги/Книга 06.md",
		PathID:    32,
		Title:     "Книга 06",
		Permalink: "/knigi/kniga_06",
	}
	otherNote := &model.NoteView{
		Path:      "Книги/Книга 01.md",
		PathID:    11,
		Title:     "Книга 01",
		Permalink: "/knigi/kniga_01",
	}
	noteViews := model.NewNoteViews()
	noteViews.RegisterNote(sourceNote)
	noteViews.RegisterNote(otherNote)

	chunks := []model.NoteChunk{
		{
			NotePath:   otherNote.Path,
			ChunkIndex: 0,
			Content:    "Книга 01\n\nНе самый подходящий фрагмент.",
			Embedding:  []float32{0, 1},
		},
		{
			NotePath:   sourceNote.Path,
			ChunkIndex: 2,
			Content:    "Книга 06\n\nЛучший способ отомстить - не уподобляться обидчику.",
			Embedding:  []float32{1, 0},
		},
	}

	results := vectorResultsFromChunks([]float32{1, 0}, chunks, noteViews, 10)

	require.Len(t, results, 2)
	require.Equal(t, sourceNote.Path, results[0].NoteView.Path)
	require.Equal(t, sourceNote.Permalink, results[0].URL)
	require.Equal(t, float64(1), results[0].Score)
	require.NotNil(t, results[0].HighlightedTitle)
	require.Equal(t, sourceNote.Title, *results[0].HighlightedTitle)
	require.Len(t, results[0].HighlightedContent, 1)
	require.Contains(t, results[0].HighlightedContent[0], "обидчику")
	require.NotContains(t, results[0].HighlightedContent[0], "Книга 06\n\n")
}

func TestMergeResultsUsesRRF(t *testing.T) {
	textResults := []model.SearchResult{
		{URL: "/a", Score: 100, NoteView: &model.NoteView{Permalink: "/a", Title: "A"}},
		{URL: "/b", Score: 90, NoteView: &model.NoteView{Permalink: "/b", Title: "B"}},
	}
	vectorResults := []model.SearchResult{
		{URL: "/b", Score: 0.9, NoteView: &model.NoteView{Permalink: "/b", Title: "B"}},
		{URL: "/c", Score: 0.8, NoteView: &model.NoteView{Permalink: "/c", Title: "C"}},
	}

	results := mergeResults(textResults, vectorResults)

	require.Len(t, results, 3)
	require.Equal(t, "/b", results[0].URL)
	require.InDelta(t, 1.0/61.0+1.0/62.0, results[0].Score, 1e-10)
	require.InDelta(t, 1.0/61.0, results[1].Score, 1e-10)
	require.InDelta(t, 1.0/62.0, results[2].Score, 1e-10)
}
