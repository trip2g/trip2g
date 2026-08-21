package sitesearch_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trip2g/internal/appreq"
	"trip2g/internal/case/sitesearch"
	"trip2g/internal/features"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"
	"trip2g/internal/openai"
	"trip2g/internal/usertoken"
)

// Golden retrieval tests: fixed fixtures with hand-computed expected rankings,
// characterising the shared Retrieve core (and Resolve's ACL/capping layer)
// against the behavior both transports had before the PR #176 consolidation.
// If a retrieval change reorders these goldens, that is a ranking change for
// BOTH the site search and the MCP search — update them deliberately.
//
// Fixture: query embedding [1,0]; text lane returns [B, A, E]; vector chunks
// score C=1.0, D=0.9, A=0.8 (dot product), E is stale (3-dim vs 2-dim query)
// and F has an empty embedding — both must be skipped by the dim guard.
//
// RRF (k=60, 1-indexed ranks):
//
//	A = 1/62 (text#2) + 1/63 (vector#3) ≈ 0.032002
//	B = 1/61 (text#1) ≈ 0.016393   ┐ exact tie, broken
//	C = 1/61 (vector#1) ≈ 0.016393 ┘ by URL: /b < /c
//	D = 1/62 (vector#2) ≈ 0.016129
//	E = 1/63 (text#3) ≈ 0.015873
func goldenEnv(t *testing.T, embURL string) *EnvMock {
	t.Helper()

	mk := func(path, permalink, title string) *appmodel.NoteView {
		return &appmodel.NoteView{
			Path: path, Permalink: permalink, Title: title,
			Content: []byte("content of " + title),
		}
	}
	notes := map[string]*appmodel.NoteView{
		"a.md": mk("a.md", "/a", "A"),
		"b.md": mk("b.md", "/b", "B"),
		"c.md": mk("c.md", "/c", "C"),
		"d.md": mk("d.md", "/d", "D"),
		"e.md": mk("e.md", "/e", "E"),
		"f.md": mk("f.md", "/f", "F"),
	}

	return &EnvMock{
		SearchLiveNotesFunc: func(string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{
				{NoteView: notes["b.md"], URL: "/b", HighlightedContent: []string{"hit b"}},
				{NoteView: notes["a.md"], URL: "/a", HighlightedContent: []string{"hit a"}},
				{NoteView: notes["e.md"], URL: "/e", HighlightedContent: []string{"hit e"}},
			}, nil
		},
		LiveNoteChunksFunc: func() []appmodel.NoteChunk {
			return []appmodel.NoteChunk{
				{NotePath: "c.md", ChunkIndex: 0, Content: "C\n\nbody c", Embedding: []float32{1, 0}},
				{NotePath: "d.md", ChunkIndex: 0, Content: "D\n\nbody d", Embedding: []float32{0.9, 0.435889894}},
				{NotePath: "a.md", ChunkIndex: 0, Content: "A\n\nbody a", Embedding: []float32{0.8, 0.6}},
				{NotePath: "e.md", ChunkIndex: 0, Content: "E\n\nbody e", Embedding: []float32{1, 0, 0}}, // stale dims
				{NotePath: "f.md", ChunkIndex: 0, Content: "F\n\nbody f", Embedding: []float32{}},        // empty
			}
		},
		LiveNoteViewsFunc: func() *appmodel.NoteViews {
			return &appmodel.NoteViews{PathMap: notes}
		},
		FeaturesFunc: func() features.Features {
			return features.Features{VectorSearch: features.VectorSearchConfig{Enabled: true}}
		},
		OpenAIFunc: func() *openai.Client { return openai.New("test-key", "test-model", embURL+"/v1") },
		LoggerFunc: func() logger.Logger { return &logger.DummyLogger{} },
	}
}

func urlsOf(results []appmodel.SearchResult) []string {
	urls := make([]string, len(results))
	for i, r := range results {
		urls[i] = r.URL
	}
	return urls
}

func TestGoldenRetrieve_TextOnly(t *testing.T) {
	srv := newEmbeddingServer(t, []float32{1, 0})
	defer srv.Close()

	env := goldenEnv(t, srv.URL)
	env.FeaturesFunc = func() features.Features { return features.Features{} } // vector disabled

	results, merged, err := sitesearch.Retrieve(context.Background(), env, "q", false, nil)
	require.NoError(t, err)
	require.False(t, merged)
	require.Equal(t, []string{"/b", "/a", "/e"}, urlsOf(results))
}

func TestGoldenRetrieve_HybridRRF(t *testing.T) {
	srv := newEmbeddingServer(t, []float32{1, 0})
	defer srv.Close()

	results, merged, err := sitesearch.Retrieve(context.Background(), goldenEnv(t, srv.URL), "q", false, nil)
	require.NoError(t, err)
	require.True(t, merged)
	require.Equal(t, []string{"/a", "/b", "/c", "/d", "/e"}, urlsOf(results))

	// /b and /c tie exactly on RRF (both 1/61); order comes from the URL tie-break.
	require.InDelta(t, results[1].Score, results[2].Score, 1e-12, "/b and /c must tie on RRF score")
	// Stale (3-dim) and empty embeddings never surface as candidates.
	require.NotContains(t, urlsOf(results), "/f")
}

// goldenCEServer scores rerank docs by substring, TEI wire shape.
func goldenCEServer(t *testing.T, scoreFor func(doc string) float64) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Texts []string `json:"texts"`
		}
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&req)) {
			return
		}
		type result struct {
			Index int     `json:"index"`
			Score float64 `json:"score"`
		}
		out := make([]result, 0, len(req.Texts))
		for i, d := range req.Texts {
			out = append(out, result{Index: i, Score: scoreFor(d)})
		}
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(out))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// Reranker ON (BlendWeight 0.5): CE strongly prefers D (0.95) over C (0.2) and
// A (0.1); B and E have no vector passage so they blend against the neutral CE
// midpoint 0.5. Blended scores: D≈0.508, A=0.500, B≈0.266, E=0.250, C≈0.075.
func TestGoldenRetrieve_RerankerBlends(t *testing.T) {
	embSrv := newEmbeddingServer(t, []float32{1, 0})
	defer embSrv.Close()

	ceScores := map[string]float64{"body a": 0.1, "body c": 0.2, "body d": 0.95}
	ceSrv := goldenCEServer(t, func(doc string) float64 {
		for sub, score := range ceScores {
			if strings.Contains(doc, sub) {
				return score
			}
		}
		t.Fatalf("unexpected rerank doc: %q", doc)
		return 0
	})

	env := goldenEnv(t, embSrv.URL)
	env.FeaturesFunc = func() features.Features {
		return features.Features{VectorSearch: features.VectorSearchConfig{
			Enabled: true,
			Reranker: features.RerankerConfig{
				Enabled:     true,
				Default:     true, // this instance reranks unless the caller opts out
				BaseURL:     ceSrv.URL + "/rerank",
				TopN:        10,
				BlendWeight: 0.5,
			},
		}}
	}

	results, _, err := sitesearch.Retrieve(context.Background(), env, "q", false, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"/d", "/a", "/b", "/e", "/c"}, urlsOf(results))
}

// hybridOrder is the plain RRF order from TestGoldenRetrieve_HybridRRF — what a
// search must return whenever the cross-encoder stage does not run.
var hybridOrder = []string{"/a", "/b", "/c", "/d", "/e"} //nolint:gochecknoglobals // test fixture

// wantRerank expresses an explicit caller preference; nil means no preference.
func wantRerank(v bool) *bool { return &v }

// A caller can decline the second stage on an instance that reranks by default:
// the CE server is wired up and would reorder, and must not be consulted.
func TestGoldenRetrieve_RerankOptOut(t *testing.T) {
	embSrv := newEmbeddingServer(t, []float32{1, 0})
	defer embSrv.Close()

	ceSrv := goldenCEServer(t, func(doc string) float64 {
		t.Fatalf("reranker must not be called when the caller passed rerank=false, got %q", doc)
		return 0
	})

	env := goldenEnv(t, embSrv.URL)
	env.FeaturesFunc = func() features.Features {
		return features.Features{VectorSearch: features.VectorSearchConfig{
			Enabled: true,
			Reranker: features.RerankerConfig{
				Enabled: true, Default: true,
				BaseURL: ceSrv.URL + "/rerank", TopN: 10, BlendWeight: 0.5,
			},
		}}
	}

	results, _, err := sitesearch.Retrieve(context.Background(), env, "q", false, wantRerank(false))
	require.NoError(t, err)
	require.Equal(t, hybridOrder, urlsOf(results))
}

// Mirror image: an opt-in instance leaves the order alone until a caller asks,
// and reorders when one does.
func TestGoldenRetrieve_RerankOptIn(t *testing.T) {
	embSrv := newEmbeddingServer(t, []float32{1, 0})
	defer embSrv.Close()

	ceScores := map[string]float64{"body a": 0.1, "body c": 0.2, "body d": 0.95}
	ceSrv := goldenCEServer(t, func(doc string) float64 {
		for sub, score := range ceScores {
			if strings.Contains(doc, sub) {
				return score
			}
		}
		t.Fatalf("unexpected rerank doc: %q", doc)
		return 0
	})

	env := goldenEnv(t, embSrv.URL)
	env.FeaturesFunc = func() features.Features {
		return features.Features{VectorSearch: features.VectorSearchConfig{
			Enabled: true,
			Reranker: features.RerankerConfig{
				Enabled: true, Default: false,
				BaseURL: ceSrv.URL + "/rerank", TopN: 10, BlendWeight: 0.5,
			},
		}}
	}

	silent, _, err := sitesearch.Retrieve(context.Background(), env, "q", false, nil)
	require.NoError(t, err)
	require.Equal(t, hybridOrder, urlsOf(silent), "no preference on an opt-in instance must not rerank")

	asked, _, err := sitesearch.Retrieve(context.Background(), env, "q", false, wantRerank(true))
	require.NoError(t, err)
	require.Equal(t, []string{"/d", "/a", "/b", "/e", "/c"}, urlsOf(asked), "an explicit ask must rerank")
}

func anonSiteEnv(env *EnvMock) *EnvMock {
	env.CurrentUserTokenFunc = func(context.Context) (*usertoken.Data, error) {
		return &usertoken.Data{}, nil
	}
	env.SiteConfigFunc = func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} }
	return env
}

// ACL layer: unreadable notes become "Закрытый материал." placeholders pushed
// to the end, in fused-rank order, after the readable results.
func TestGoldenResolve_ACLPlaceholders(t *testing.T) {
	srv := newEmbeddingServer(t, []float32{1, 0})
	defer srv.Close()

	env := anonSiteEnv(goldenEnv(t, srv.URL))
	env.CanReadNoteFunc = func(_ context.Context, nv *appmodel.NoteView) (bool, error) {
		return nv.Path != "a.md" && nv.Path != "c.md", nil
	}

	ctx := appreq.NewContext(context.Background(), &appreq.Request{})
	conn, err := sitesearch.Resolve(ctx, env, model.SearchInput{Query: "q"})
	require.NoError(t, err)

	require.Equal(t, []string{"/b", "/d", "/e", "/a", "/c"}, urlsOf(conn.Nodes))
	for _, hidden := range conn.Nodes[3:] {
		require.Nil(t, hidden.NoteView)
		require.Equal(t, []string{"Закрытый материал."}, hidden.HighlightedContent)
	}
}

// Capping layer: the hybrid cap (20) applies AFTER permission filtering, so
// with 24 fused candidates whose top 4 are unreadable, exactly the 20 readable
// ones surface and the placeholders are cut, not the readable tail.
func TestGoldenResolve_PostACLCap(t *testing.T) {
	srv := newEmbeddingServer(t, []float32{1, 0})
	defer srv.Close()

	var textResults []appmodel.SearchResult
	pathMap := map[string]*appmodel.NoteView{}
	for i := 1; i <= 24; i++ {
		note := &appmodel.NoteView{
			Path:      fmt.Sprintf("n%02d.md", i),
			Permalink: fmt.Sprintf("/n%02d", i),
			Title:     fmt.Sprintf("N%02d", i),
		}
		pathMap[note.Path] = note
		textResults = append(textResults, appmodel.SearchResult{NoteView: note, URL: note.Permalink})
	}

	env := anonSiteEnv(goldenEnv(t, srv.URL))
	env.SearchLiveNotesFunc = func(string) ([]appmodel.SearchResult, error) { return textResults, nil }
	env.LiveNoteViewsFunc = func() *appmodel.NoteViews { return &appmodel.NoteViews{PathMap: pathMap} }
	env.LiveNoteChunksFunc = func() []appmodel.NoteChunk {
		// One chunk for the top text hit: triggers the hybrid merge (and its cap)
		// without reordering anything else.
		return []appmodel.NoteChunk{
			{NotePath: "n01.md", ChunkIndex: 0, Content: "N01\n\nbody", Embedding: []float32{1, 0}},
		}
	}
	env.CanReadNoteFunc = func(_ context.Context, nv *appmodel.NoteView) (bool, error) {
		return nv.Path > "n04.md", nil // n01..n04 unreadable
	}

	ctx := appreq.NewContext(context.Background(), &appreq.Request{})
	conn, err := sitesearch.Resolve(ctx, env, model.SearchInput{Query: "q"})
	require.NoError(t, err)

	expected := make([]string, 0, 20)
	for i := 5; i <= 24; i++ {
		expected = append(expected, fmt.Sprintf("/n%02d", i))
	}
	require.Equal(t, expected, urlsOf(conn.Nodes))
	for _, n := range conn.Nodes {
		require.NotNil(t, n.NoteView, "placeholders must be cut by the cap, not readable results")
	}
}
