package sitesearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trip2g/internal/features"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"
	"trip2g/internal/openai"
	"trip2g/internal/usertoken"
)

// rerankEnv is a minimal Env exposing only what rerankResults touches
// (Features and Logger). Every other method panics so an unexpected call is loud.
type rerankEnv struct {
	feats features.Features
}

func (e *rerankEnv) Features() features.Features { return e.feats }
func (e *rerankEnv) Logger() logger.Logger       { return &logger.DummyLogger{} }

func (e *rerankEnv) SearchLatestNotes(string) ([]appmodel.SearchResult, error) { panic("unused") }
func (e *rerankEnv) SearchLiveNotes(string) ([]appmodel.SearchResult, error)   { panic("unused") }
func (e *rerankEnv) CurrentUserToken(context.Context) (*usertoken.Data, error) { panic("unused") }
func (e *rerankEnv) CanReadNote(context.Context, *appmodel.NoteView) (bool, error) {
	panic("unused")
}
func (e *rerankEnv) SiteConfig(context.Context) appmodel.SiteConfig { panic("unused") }
func (e *rerankEnv) OpenAI() *openai.Client                         { panic("unused") }
func (e *rerankEnv) LatestNoteViews() *appmodel.NoteViews           { panic("unused") }
func (e *rerankEnv) LiveNoteViews() *appmodel.NoteViews             { panic("unused") }
func (e *rerankEnv) LatestNoteChunks() []appmodel.NoteChunk         { panic("unused") }
func (e *rerankEnv) LiveNoteChunks() []appmodel.NoteChunk           { panic("unused") }

func rerankerFeatures(w float64, url string) features.Features {
	var f features.Features
	f.VectorSearch.Reranker = features.RerankerConfig{
		Enabled:     true,
		BaseURL:     url,
		Model:       "test",
		TopN:        50,
		OutputK:     20,
		BlendWeight: w,
	}
	return f
}

// ceServer returns a rerank server that scores each doc via scoreFor, letting
// tests drive the cross-encoder ranking. Request/response shapes match TEI:
// request has "texts" (not "documents"), response is a bare array [{index, score}].
func ceServer(t *testing.T, scoreFor func(doc string) float64) *httptest.Server {
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
		var out []result
		for i, d := range req.Texts {
			out = append(out, result{Index: i, Score: scoreFor(d)})
		}
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(out))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRerankBlend_CENudgesButRRFDominatesAtLowWeight(t *testing.T) {
	// Stage-1 (RRF) order: A > B. CE strongly prefers B.
	// With a low blend weight, A must stay on top (CE only nudges).
	results := []appmodel.SearchResult{
		makeResult("/a", 0.030, "A"),
		makeResult("/b", 0.020, "B"),
	}
	passages := map[string]string{"/a": "alpha passage", "/b": "beta passage"}

	srv := ceServer(t, func(doc string) float64 {
		if strings.Contains(doc, "beta") {
			return 0.99
		}
		return 0.01
	})
	env := &rerankEnv{feats: rerankerFeatures(0.2, srv.URL)}

	out := rerankResults(context.Background(), env, "q", results, passages, wantRerank(true))
	require.Equal(t, "/a", out[0].URL, "low weight: strong RRF prior must win")
}

func TestRerankBlend_CEOverridesAtHighWeight(t *testing.T) {
	// Same close RRF order, but a high blend weight lets a confident CE flip it.
	results := []appmodel.SearchResult{
		makeResult("/a", 0.021, "A"),
		makeResult("/b", 0.020, "B"),
	}
	passages := map[string]string{"/a": "alpha passage", "/b": "beta passage"}

	srv := ceServer(t, func(doc string) float64 {
		if strings.Contains(doc, "beta") {
			return 0.99
		}
		return 0.01
	})
	env := &rerankEnv{feats: rerankerFeatures(0.9, srv.URL)}

	out := rerankResults(context.Background(), env, "q", results, passages, wantRerank(true))
	require.Equal(t, "/b", out[0].URL, "high weight: confident CE flips a near-tie")
}

func TestRerankBlend_NoPassageKeepsStageOne(t *testing.T) {
	// A text-only candidate without a window-sized passage must not be dropped.
	results := []appmodel.SearchResult{
		makeResult("/a", 0.030, "A"),
		makeResult("/b", 0.020, "B"),
		makeResult("/c", 0.010, "C"),
	}
	passages := map[string]string{"/a": "alpha passage", "/b": "beta passage"}

	srv := ceServer(t, func(string) float64 { return 0.5 })
	env := &rerankEnv{feats: rerankerFeatures(0.5, srv.URL)}

	out := rerankResults(context.Background(), env, "q", results, passages, wantRerank(true))
	require.Len(t, out, 3)
	urls := map[string]bool{}
	for _, r := range out {
		urls[r.URL] = true
	}
	require.True(t, urls["/c"], "text-only candidate must survive")
}

func TestRerankBlend_DisabledIsPassthrough(t *testing.T) {
	results := []appmodel.SearchResult{makeResult("/a", 0.03, "A"), makeResult("/b", 0.02, "B")}
	env := &rerankEnv{feats: features.Features{}} // reranker disabled
	out := rerankResults(context.Background(), env, "q", results, nil, wantRerank(true))
	require.Equal(t, results, out)
}

// wantRerank expresses an explicit caller preference for the second stage.
// A nil *bool means "no preference", which is what most call sites pass.
func wantRerank(v bool) *bool { return &v }

// deadCEServer stands in for a reranker that must never be reached: any request
// to it fails the test, which is how these cases prove the stage was skipped
// rather than merely producing the same order by luck.
func deadCEServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("reranker was called when it should have been skipped")
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRerankResults_SkippedUnlessAsked(t *testing.T) {
	results := []appmodel.SearchResult{
		makeResult("/a", 0.030, "A"),
		makeResult("/b", 0.020, "B"),
	}
	passages := map[string]string{"/a": "alpha passage", "/b": "beta passage"}

	cases := []struct {
		name    string
		enabled bool
		byDflt  bool
		want    *bool
	}{
		{name: "capability off, no preference", enabled: false, byDflt: false, want: nil},
		{name: "capability off, caller asks anyway", enabled: false, byDflt: false, want: wantRerank(true)},
		{name: "opt-in instance, no preference", enabled: true, byDflt: false, want: nil},
		{name: "opt-in instance, caller declines", enabled: true, byDflt: false, want: wantRerank(false)},
		{name: "default-on instance, caller declines", enabled: true, byDflt: true, want: wantRerank(false)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			feats := rerankerFeatures(0.9, deadCEServer(t).URL)
			feats.VectorSearch.Reranker.Enabled = tc.enabled
			feats.VectorSearch.Reranker.Default = tc.byDflt

			out := rerankResults(context.Background(), &rerankEnv{feats: feats}, "q", results, passages, tc.want)
			require.Equal(t, []string{"/a", "/b"}, []string{out[0].URL, out[1].URL},
				"stage-1 order must survive untouched")
		})
	}
}

func TestRerankResults_DefaultOnRuns(t *testing.T) {
	// Same near-tie as the blend tests: with the CE consulted, /b wins.
	results := []appmodel.SearchResult{
		makeResult("/a", 0.021, "A"),
		makeResult("/b", 0.020, "B"),
	}
	passages := map[string]string{"/a": "alpha passage", "/b": "beta passage"}

	srv := ceServer(t, func(doc string) float64 {
		if strings.Contains(doc, "beta") {
			return 0.99
		}
		return 0.01
	})
	feats := rerankerFeatures(0.9, srv.URL)
	feats.VectorSearch.Reranker.Default = true

	out := rerankResults(context.Background(), &rerankEnv{feats: feats}, "q", results, passages, nil)
	require.Equal(t, "/b", out[0].URL, "a default-on instance reranks without being asked")
}
