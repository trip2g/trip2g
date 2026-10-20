package mcp

import (
	"context"
	"strings"
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
		NoteView:        note,
		URL:             note.Permalink,
		Score:           1,
		MatchOrigin:     model.SearchMatchVector,
		ChunkIndex:      &chunkIndex,
		ContentSegments: model.PlainSegments("Лучший способ отомстить - не уподобляться обидчику."),
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
		NoteView:        note,
		URL:             note.Permalink,
		Score:           1,
		MatchOrigin:     model.SearchMatchText,
		ContentSegments: []model.HighlightSegment{{Text: "Лучший способ "}, {Text: "отомстить", Match: true}, {Text: " - не уподобляться обидчику."}},
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
		NoteView:        note,
		URL:             note.Permalink,
		Score:           1,
		MatchOrigin:     model.SearchMatchText,
		ContentSegments: model.PlainSegments("Фрагмент, которого нет в чанках."),
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
		NoteView:        note,
		URL:             note.Permalink,
		Score:           1,
		MatchOrigin:     model.SearchMatchVector,
		ChunkIndex:      &chunkIndex,
		ContentSegments: model.PlainSegments("Текст, которого нет в HTML заметки."),
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

func TestSearchResultItemCarriesTheNoteSizeAndSections(t *testing.T) {
	long := &model.NoteView{
		Path:    "guide.md",
		Content: []byte("---\ntitle: Guide\n---\n" + strings.Repeat("word ", 400)),
		Headings: model.NoteViewHeadings{
			{Level: 2, Text: "Install"},
			{Level: 2, Text: "Configure"},
			{Level: 3, Text: "Keys"},
		},
	}
	short := &model.NoteView{Path: "faq.md", Content: []byte("Short answer.")}
	noteURL := func(*model.NoteView) string { return "" }

	longItem := searchResultItemFromNote(long, 1, noteURL)
	require.Equal(t, 501, longItem.SizeTokens)
	require.Equal(t, 3, longItem.Sections)

	shortItem := searchResultItemFromNote(short, 1, noteURL)
	require.Equal(t, 4, shortItem.SizeTokens)
	require.Equal(t, 0, shortItem.Sections)
}
