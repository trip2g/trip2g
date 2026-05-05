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
	require.InEpsilon(t, float64(1), results[0].Score, 1e-9)
	require.NotNil(t, results[0].ChunkIndex)
	require.Equal(t, 2, *results[0].ChunkIndex)
	require.NotNil(t, results[0].HighlightedTitle)
	require.Equal(t, sourceNote.Title, *results[0].HighlightedTitle)
	require.Len(t, results[0].HighlightedContent, 1)
	require.Contains(t, results[0].HighlightedContent[0], "обидчику")
	require.NotContains(t, results[0].HighlightedContent[0], "Книга 06\n\n")
}

func TestBuildSearchPayloadUsesChunkBasedMatchIDs(t *testing.T) {
	chunkIndex := 4
	note := &model.NoteView{
		Path:      "Книги/Книга 06.md",
		PathID:    32,
		Title:     "Книга 06",
		Permalink: "/knigi/kniga_06",
	}

	payload := buildSearchPayload("обида", []model.SearchResult{{
		NoteView:           note,
		URL:                note.Permalink,
		Score:              1,
		MatchOrigin:        model.SearchMatchVector,
		ChunkIndex:         &chunkIndex,
		HighlightedContent: []string{"Лучший способ отомстить - не уподобляться обидчику."},
	}}, func(n *model.NoteView) string { return "https://markavrelii.2pub.me" + n.Permalink }, nil)

	require.Len(t, payload.Results, 1)
	require.Len(t, payload.Results[0].Matches, 1)
	require.Equal(t, "p32:c4", payload.Results[0].Matches[0].MatchID)
	require.Equal(t, 4, payload.Results[0].Matches[0].ChunkIndex)
}

func TestBuildSearchPayloadMapsTextSnippetToNearestChunkWhenClear(t *testing.T) {
	note := &model.NoteView{
		Path:      "Книги/Книга 06.md",
		PathID:    32,
		Title:     "Книга 06",
		Permalink: "/knigi/kniga_06",
	}

	payload := buildSearchPayload("обида", []model.SearchResult{{
		NoteView:           note,
		URL:                note.Permalink,
		Score:              1,
		MatchOrigin:        model.SearchMatchText,
		HighlightedContent: []string{"Лучший способ <mark>отомстить</mark> - не уподобляться обидчику."},
	}}, func(n *model.NoteView) string { return "https://markavrelii.2pub.me" + n.Permalink }, []model.NoteChunk{
		{
			NotePath:   note.Path,
			ChunkIndex: 1,
			Content:    "Книга 06\n\nЛучший способ отомстить - не уподобляться обидчику.",
		},
		{
			NotePath:   note.Path,
			ChunkIndex: 2,
			Content:    "Книга 06\n\nСовсем другой фрагмент.",
		},
	})

	require.Len(t, payload.Results, 1)
	require.Len(t, payload.Results[0].Matches, 1)
	require.Equal(t, "p32:c1", payload.Results[0].Matches[0].MatchID)
	require.Equal(t, 1, payload.Results[0].Matches[0].ChunkIndex)
}

func TestBuildSearchPayloadLeavesTextMatchAsGenericWhenNoClearChunk(t *testing.T) {
	note := &model.NoteView{
		Path:      "Книги/Книга 06.md",
		PathID:    32,
		Title:     "Книга 06",
		Permalink: "/knigi/kniga_06",
	}

	payload := buildSearchPayload("обида", []model.SearchResult{{
		NoteView:           note,
		URL:                note.Permalink,
		Score:              1,
		MatchOrigin:        model.SearchMatchText,
		HighlightedContent: []string{"Фрагмент, которого нет в чанках."},
	}}, func(n *model.NoteView) string { return "https://markavrelii.2pub.me" + n.Permalink }, []model.NoteChunk{
		{
			NotePath:   note.Path,
			ChunkIndex: 1,
			Content:    "Книга 06\n\nЛучший способ отомстить - не уподобляться обидчику.",
		},
	})

	require.Len(t, payload.Results, 1)
	require.Len(t, payload.Results[0].Matches, 1)
	require.Equal(t, "p32:m1", payload.Results[0].Matches[0].MatchID)
	require.Equal(t, 0, payload.Results[0].Matches[0].ChunkIndex)
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
