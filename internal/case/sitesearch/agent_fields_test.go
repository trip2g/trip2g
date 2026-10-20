package sitesearch_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trip2g/internal/case/sitesearch"
	"trip2g/internal/features"
	appmodel "trip2g/internal/model"
)

// textEmbeddingServer embeds each input by exact text, so a test can give the
// query and each agent field a vector of its own. It records every input.
func textEmbeddingServer(t *testing.T, vectors map[string][]float32) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var inputs []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Input []string `json:"input"`
		}
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&req)) {
			return
		}
		data := make([]map[string]any, 0, len(req.Input))
		for i, in := range req.Input {
			vector, ok := vectors[in]
			assert.True(t, ok, "unexpected embedding input %q", in)
			data = append(data, map[string]any{"object": "embedding", "index": i, "embedding": vector})
		}
		mu.Lock()
		inputs = append(inputs, req.Input...)
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data":   data,
			"usage":  map[string]any{"total_tokens": 3},
		})
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), inputs...)
	}
}

// textLaneByQuery answers the text lane per query text, the way bleve's AND
// match does: a long question finds B only, its two key words find A.
func textLaneByQuery(env *EnvMock, hits map[string][]string) {
	views := env.LiveNoteViewsFunc()
	env.SearchLiveNotesFunc = func(query string, _, _ int) ([]appmodel.SearchResult, error) {
		var out []appmodel.SearchResult
		for _, path := range hits[query] {
			note := views.PathMap[path]
			out = append(out, appmodel.SearchResult{
				NoteView: note, URL: note.Permalink, ContentSegments: appmodel.PlainSegments("hit " + path),
			})
		}
		return out, nil
	}
}

func searchedQueries(env *EnvMock) []string {
	var out []string
	for _, c := range env.SearchLiveNotesCalls() {
		out = append(out, c.Query)
	}
	return out
}

func TestRetrieve_ShortQueryTextOnly(t *testing.T) {
	const long = "how do I pin a tab so it stays open"

	tests := []struct {
		name       string
		short      string
		want       []string
		wantSearch []string
	}{
		{
			name:       "no short query keeps the text order",
			want:       []string{"/b"},
			wantSearch: []string{long},
		},
		{
			// B = 1/61 (text#1), A = 1/61 (short#1): a tie broken by URL.
			name:       "short query adds the note the long query misses",
			short:      "pin tab",
			want:       []string{"/a", "/b"},
			wantSearch: []string{long, "pin tab"},
		},
		{
			// B = 1/61 + 1/62, A = 1/61.
			name:       "a note both lanes find ranks first",
			short:      "pinned tab",
			want:       []string{"/b", "/a"},
			wantSearch: []string{long, "pinned tab"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := goldenEnv(t, "http://unused")
			env.FeaturesFunc = func() features.Features { return features.Features{} }
			textLaneByQuery(env, map[string][]string{
				long:         {"b.md"},
				"pin tab":    {"a.md"},
				"pinned tab": {"a.md", "b.md"},
			})

			req := sitesearch.Request{Query: long, ShortQuery: tt.short}
			results, merged, err := sitesearch.Retrieve(context.Background(), env, req)
			require.NoError(t, err)
			require.False(t, merged)
			require.Equal(t, tt.want, urlsOf(results))
			require.Equal(t, tt.wantSearch, searchedQueries(env))
			require.Empty(t, env.OpenAICalls())
		})
	}
}

func TestRetrieve_VectorFieldsIgnoredWithoutVectors(t *testing.T) {
	env := goldenEnv(t, "http://unused")
	env.FeaturesFunc = func() features.Features { return features.Features{} }

	req := sitesearch.Request{Query: "q", RephrasedQuery: "r", ExpectedAnswer: "e"}
	results, merged, err := sitesearch.Retrieve(context.Background(), env, req)
	require.NoError(t, err)
	require.False(t, merged)
	require.Equal(t, []string{"/b", "/a", "/e"}, urlsOf(results))
	require.Equal(t, []string{"q"}, searchedQueries(env))
	require.Empty(t, env.OpenAICalls())
}

// The query embeds to [1,0] (vector lane C, D, A) and both vector fields to
// [0,1] (A 0.6, D 0.436, C 0): an extra vector list ranks A, D, C. The text
// lane gives B, A, E for "q" and D alone for the short query "s".
//
//	base:      A = 1/62+1/63, B = 1/61, C = 1/61, D = 1/62, E = 1/63
//	one field: A += 1/61, D += 1/62, C += 1/63 → A, C (≈0.03227), D (≈0.03226), B, E
//	short "s": D += 1/61 → D (≈0.0325) beats A (≈0.0320)
//	all three: A ≈ 0.06479 stays just ahead of D ≈ 0.06478
func TestRetrieve_AgentFieldsWithVectors(t *testing.T) {
	tests := []struct {
		name      string
		req       sitesearch.Request
		want      []string
		wantEmbed []string
	}{
		{
			name:      "no fields",
			req:       sitesearch.Request{Query: "q"},
			want:      []string{"/a", "/b", "/c", "/d", "/e"},
			wantEmbed: []string{"q"},
		},
		{
			name:      "rephrased query",
			req:       sitesearch.Request{Query: "q", RephrasedQuery: "r"},
			want:      []string{"/a", "/c", "/d", "/b", "/e"},
			wantEmbed: []string{"q", "r"},
		},
		{
			name:      "expected answer",
			req:       sitesearch.Request{Query: "q", ExpectedAnswer: "e"},
			want:      []string{"/a", "/c", "/d", "/b", "/e"},
			wantEmbed: []string{"q", "e"},
		},
		{
			name:      "short query goes to the text lane only",
			req:       sitesearch.Request{Query: "q", ShortQuery: "s"},
			want:      []string{"/d", "/a", "/b", "/c", "/e"},
			wantEmbed: []string{"q"},
		},
		{
			name:      "all three",
			req:       sitesearch.Request{Query: "q", ShortQuery: "s", RephrasedQuery: "r", ExpectedAnswer: "e"},
			want:      []string{"/a", "/d", "/c", "/b", "/e"},
			wantEmbed: []string{"q", "r", "e"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, embedded := textEmbeddingServer(t, map[string][]float32{
				"q": {1, 0},
				"r": {0, 1},
				"e": {0, 1},
			})
			env := goldenEnv(t, srv.URL)
			textLaneByQuery(env, map[string][]string{
				"q": {"b.md", "a.md", "e.md"},
				"s": {"d.md"},
			})

			results, merged, err := sitesearch.Retrieve(context.Background(), env, tt.req)
			require.NoError(t, err)
			require.True(t, merged)
			require.Equal(t, tt.want, urlsOf(results))
			require.Equal(t, tt.wantEmbed, embedded())
		})
	}
}
