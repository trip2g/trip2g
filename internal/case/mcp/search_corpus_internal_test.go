package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/features"
	"trip2g/internal/model"
)

// fedCorpusEnv extends the panicking fedGQLEnv stub with just what an
// anonymous-shaped search needs. SearchLatestNotes stays a panic, so touching
// the latest (draft) corpus fails loudly.
type fedCorpusEnv struct {
	fedGQLEnv
	live *model.NoteView
}

func (e *fedCorpusEnv) SearchLiveNotes(string) ([]model.SearchResult, error) {
	return []model.SearchResult{{
		NoteView: e.live, URL: e.live.Permalink,
		HighlightedContent: []string{"published snippet"},
	}}, nil
}
func (e *fedCorpusEnv) LiveNoteChunks() []model.NoteChunk { return nil }
func (e *fedCorpusEnv) Features() features.Features       { return features.Features{} }
func (e *fedCorpusEnv) NoteURL(n *model.NoteView) string  { return "https://x.test" + n.Permalink }

// Federation-JWT clients must search the LIVE corpus, like anonymous clients —
// only API-key (admin) auth gets the latest (draft) corpus.
func TestSearchFederationJWTUsesLiveCorpus(t *testing.T) {
	env := &fedCorpusEnv{
		// Free note: readable through the federation subgraph ACL without extra deps.
		live: &model.NoteView{Path: "published.md", PathID: 2, Title: "Published", Permalink: "/published", Free: true},
	}

	ctx := contextWithFederationAuth(context.Background(), "kid1", []string{"subA"})
	require.False(t, mcpAPIKeyAuthed(ctx), "federation auth must not count as API-key auth")

	resp := handleSearch(ctx, env, 1, json.RawMessage(`{"query":"x"}`))

	require.Nil(t, resp.Error)
	result, ok := resp.Result.(CallToolResult)
	require.True(t, ok)
	payload, ok := result.StructuredContent.(SearchResultPayload)
	require.True(t, ok)
	require.Len(t, payload.Results, 1)
	require.Equal(t, "Published", payload.Results[0].Title)
}
