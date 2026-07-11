package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/model"
)

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
	}}, func(n *model.NoteView) string { return "https://markavrelii.2pub.me" + n.Permalink }, nil, MaxSearchLimit, MaxSearchLimit)

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
	}, MaxSearchLimit, MaxSearchLimit)

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
	}, MaxSearchLimit, MaxSearchLimit)

	require.Len(t, payload.Results, 1)
	require.Len(t, payload.Results[0].Matches, 1)
	// No chunk resolved: match_id is omitted instead of an unusable "m"-form id.
	require.Empty(t, payload.Results[0].Matches[0].MatchID)
	require.Equal(t, 0, payload.Results[0].Matches[0].ChunkIndex)
}

func TestBuildSearchPayloadFirstChunkGetsBreadcrumbFallback(t *testing.T) {
	chunkIndex := 0
	note := &model.NoteView{
		Path:      "Книги/Книга 06.md",
		PathID:    32,
		Title:     "Книга 06",
		Permalink: "/knigi/kniga_06",
		HTML: `<div data-header="Интро"><h1>Интро</h1><p>Вступление.</p></div>` +
			`<div data-header="Раздел"><h1>Раздел</h1><p>Что-то другое.</p></div>`,
	}

	payload := buildSearchPayload("обида", []model.SearchResult{{
		NoteView:           note,
		URL:                note.Permalink,
		Score:              1,
		MatchOrigin:        model.SearchMatchVector,
		ChunkIndex:         &chunkIndex,
		HighlightedContent: []string{"Текст, которого нет в HTML заметки."},
	}}, func(n *model.NoteView) string { return "https://markavrelii.2pub.me" + n.Permalink }, []model.NoteChunk{
		{
			NotePath:   note.Path,
			ChunkIndex: 0,
			Content:    "Книга 06 > Раздел\n\nТекст, которого нет в HTML заметки.",
		},
	}, MaxSearchLimit, MaxSearchLimit)

	require.Len(t, payload.Results, 1)
	require.Len(t, payload.Results[0].Matches, 1)
	require.Equal(t, "p32:c0", payload.Results[0].Matches[0].MatchID)
	// Chunk 0 is a real chunk: its breadcrumb resolves the toc_path instead of
	// falling back to the note's first section.
	require.Equal(t, []string{"Раздел"}, payload.Results[0].Matches[0].TOCPath)
}

func TestMCPAPIKeyContext(t *testing.T) {
	ctx := context.Background()

	require.False(t, mcpAPIKeyAuthed(ctx))
	require.False(t, mcpAdminToolsEnabled(ctx))

	ctx = contextWithMCPAPIKeyAuth(ctx, false)
	require.True(t, mcpAPIKeyAuthed(ctx))
	require.False(t, mcpAdminToolsEnabled(ctx))

	ctx = contextWithMCPAPIKeyAuth(ctx, true)
	require.True(t, mcpAPIKeyAuthed(ctx))
	require.True(t, mcpAdminToolsEnabled(ctx))
}
