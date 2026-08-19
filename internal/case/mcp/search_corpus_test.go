package mcp_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/appreq"
	"trip2g/internal/case/mcp"
	"trip2g/internal/case/sitesearch"
	"trip2g/internal/db"
	"trip2g/internal/features"
	graphmodel "trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/metrics"
	appmodel "trip2g/internal/model"
	"trip2g/internal/openai"
	"trip2g/internal/usertoken"
)

func searchToolCall(t *testing.T) mcp.Request {
	t.Helper()
	params, err := json.Marshal(mcp.CallToolParams{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"x"}`),
	})
	require.NoError(t, err)
	return mcp.Request{JSONRPC: "2.0", Method: "tools/call", Params: params, ID: 1}
}

// corpusEnv returns an EnvMock whose latest and live corpora contain different
// notes, so tests can tell which corpus a search consulted.
func corpusEnv() *EnvMock {
	latest := &appmodel.NoteView{Path: "draft.md", PathID: 1, Title: "Draft Only", Permalink: "/draft"}
	live := &appmodel.NoteView{Path: "published.md", PathID: 2, Title: "Published", Permalink: "/published"}

	return &EnvMock{
		FederatedFanoutTimeoutFunc: func() time.Duration { return 2 * time.Second },
		MCPMetricsFunc:             func() *metrics.MCPMetrics { return nil },
		SearchLatestNotesFunc: func(string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{{
				NoteView: latest, URL: latest.Permalink, Score: 1,
				HighlightedContent: []string{"draft snippet"},
			}}, nil
		},
		SearchLiveNotesFunc: func(string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{{
				NoteView: live, URL: live.Permalink, Score: 1,
				HighlightedContent: []string{"published snippet"},
			}}, nil
		},
		LatestNoteChunksFunc: func() []appmodel.NoteChunk { return nil },
		LiveNoteChunksFunc:   func() []appmodel.NoteChunk { return nil },
		FeaturesFunc:         func() features.Features { return features.Features{} },
		NoteURLFunc:          func(n *appmodel.NoteView) string { return "https://x.test" + n.Permalink },
		LoggerFunc:           func() logger.Logger { return &logger.DummyLogger{} },
		CanReadNoteFunc:      func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil },
		SiteConfigFunc:       func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
	}
}

func searchPayload(t *testing.T, resp mcp.Response) mcp.SearchResultPayload {
	t.Helper()
	require.Nil(t, resp.Error)
	result, ok := resp.Result.(mcp.CallToolResult)
	require.True(t, ok)
	return decodePayload[mcp.SearchResultPayload](t, result)
}

// Anonymous MCP clients must search the LIVE corpus, like anonymous site
// visitors — not draft (latest) content of public notes.
func TestSearchAnonymousUsesLiveCorpus(t *testing.T) {
	env := corpusEnv()

	resp := callMCP(t, env, searchToolCall(t))
	payload := searchPayload(t, resp)

	require.Len(t, payload.Results, 1)
	require.Equal(t, "Published", payload.Results[0].Title)
	require.Empty(t, env.SearchLatestNotesCalls(),
		"anonymous search must not touch the latest (draft) corpus")
}

// Instances that show draft versions to site visitors (ShowDraftVersions)
// must search the latest corpus for anonymous MCP clients too — the corpus
// choice mirrors the site search rule, not just the auth method. Regression
// test for the outage where anonymous federated search returned zero results
// on such instances because the live index is nearly empty.
func TestSearchAnonymousWithShowDraftVersionsUsesLatestCorpus(t *testing.T) {
	env := corpusEnv()
	env.SiteConfigFunc = func(context.Context) appmodel.SiteConfig {
		return appmodel.SiteConfig{ShowDraftVersions: true}
	}

	resp := callMCP(t, env, searchToolCall(t))
	payload := searchPayload(t, resp)

	require.Len(t, payload.Results, 1)
	require.Equal(t, "Draft Only", payload.Results[0].Title)
	require.Empty(t, env.SearchLiveNotesCalls(),
		"ShowDraftVersions instance must search the latest corpus, even anonymously")
}

// API-key (admin) clients keep searching the latest corpus, drafts included.
func TestSearchAPIKeyUsesLatestCorpus(t *testing.T) {
	env := corpusEnv()
	env.ResolveAPIKeyFunc = func(context.Context, string, string) (*db.ApiKey, error) {
		return &db.ApiKey{ID: 1}, nil
	}
	env.MCPMetricsFunc = func() *metrics.MCPMetrics { return nil }

	body, err := json.Marshal(searchToolCall(t))
	require.NoError(t, err)
	fasthttpCtx := buildMCPFasthttpCtx(body, "")
	fasthttpCtx.Request.Header.Set("X-API-Key", "any-key")

	req := wiredRequest(fasthttpCtx, env, nil)
	defer appreq.Release(req)

	_, err = (&mcp.Endpoint{}).Handle(req)
	require.NoError(t, err)

	var resp mcp.Response
	require.NoError(t, json.Unmarshal(fasthttpCtx.Response.Body(), &resp))
	require.Nil(t, resp.Error)

	var result struct {
		StructuredContent mcp.SearchResultPayload `json:"structuredContent"`
	}
	raw, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &result))

	require.Len(t, result.StructuredContent.Results, 1)
	require.Equal(t, "Draft Only", result.StructuredContent.Results[0].Title)
	require.Empty(t, env.SearchLiveNotesCalls(),
		"API-key search must use the latest corpus")
}

// Stale chunks whose embedding dimensionality differs from the query embedding
// must be skipped, not scored ~0 into an arbitrary top-K.
func TestSearchSkipsDimMismatchedChunks(t *testing.T) {
	srv := embeddingServer(t, []float32{1, 0, 0}) // 3-dim query
	env := corpusEnv()
	env.SearchLiveNotesFunc = func(string) ([]appmodel.SearchResult, error) { return nil, nil }
	env.FeaturesFunc = func() features.Features {
		return features.Features{VectorSearch: features.VectorSearchConfig{Enabled: true}}
	}
	env.OpenAIFunc = func() *openai.Client { return openai.New("k", "m", srv.URL+"/v1") }
	env.LiveNoteViewsFunc = func() *appmodel.NoteViews {
		return &appmodel.NoteViews{PathMap: map[string]*appmodel.NoteView{
			"stale.md": {Path: "stale.md", PathID: 3, Title: "Stale", Permalink: "/stale"},
		}}
	}
	env.LiveNoteChunksFunc = func() []appmodel.NoteChunk {
		return []appmodel.NoteChunk{
			{NotePath: "stale.md", Content: "Stale\n\nbody", Embedding: []float32{1, 0}}, // 2-dim
		}
	}

	resp := callMCP(t, env, searchToolCall(t))
	payload := searchPayload(t, resp)
	require.Empty(t, payload.Results,
		"dim-mismatched chunks must not surface as vector candidates")
}

// Both transports must rank identically for the same query and corpus: the MCP
// search and the site (GraphQL) search now share one retrieval core.
func TestSearchRankingMatchesSiteSearch(t *testing.T) {
	srv := embeddingServer(t, []float32{1, 0})

	notes := []*appmodel.NoteView{
		{Path: "a.md", PathID: 1, Title: "A", Permalink: "/a", Content: []byte("content a")},
		{Path: "b.md", PathID: 2, Title: "B", Permalink: "/b", Content: []byte("content b")},
		{Path: "c.md", PathID: 3, Title: "C", Permalink: "/c", Content: []byte("content c")},
		{Path: "d.md", PathID: 4, Title: "D", Permalink: "/d", Content: []byte("content d")},
	}
	pathMap := map[string]*appmodel.NoteView{}
	for _, n := range notes {
		pathMap[n.Path] = n
	}

	// Text lane: b then a. Vector lane: c, d, a — so /a is hybrid, and /b vs /c
	// tie on RRF score (rank 0 text vs rank 0 vector), exercising the
	// deterministic tie-break in both transports.
	textResults := func(string) ([]appmodel.SearchResult, error) {
		return []appmodel.SearchResult{
			{NoteView: pathMap["b.md"], URL: "/b", HighlightedContent: []string{"hit b"}},
			{NoteView: pathMap["a.md"], URL: "/a", HighlightedContent: []string{"hit a"}},
		}, nil
	}
	chunks := []appmodel.NoteChunk{
		{NotePath: "c.md", ChunkIndex: 0, Content: "C\n\nbody c", Embedding: []float32{1, 0}},
		{NotePath: "d.md", ChunkIndex: 0, Content: "D\n\nbody d", Embedding: []float32{0.9, 0.435889894}},
		{NotePath: "a.md", ChunkIndex: 0, Content: "A\n\nbody a", Embedding: []float32{0.8, 0.6}},
	}
	feats := func() features.Features {
		return features.Features{VectorSearch: features.VectorSearchConfig{Enabled: true}}
	}

	env := corpusEnv()
	env.SearchLiveNotesFunc = textResults
	env.FeaturesFunc = feats
	env.OpenAIFunc = func() *openai.Client { return openai.New("k", "m", srv.URL+"/v1") }
	env.LiveNoteViewsFunc = func() *appmodel.NoteViews { return &appmodel.NoteViews{PathMap: pathMap} }
	env.LiveNoteChunksFunc = func() []appmodel.NoteChunk { return chunks }

	mcpOrder := func() []string {
		resp := callMCP(t, env, searchToolCall(t))
		payload := searchPayload(t, resp)
		var order []string
		for _, r := range payload.Results {
			order = append(order, r.Href)
		}
		return order
	}()

	siteOrder := func() []string {
		conn, err := sitesearch.Resolve(
			appreq.NewContext(context.Background(), &appreq.Request{}),
			siteSearchEnv{env},
			graphmodel.SearchInput{Query: "x"},
		)
		require.NoError(t, err)
		var order []string
		for _, n := range conn.Nodes {
			order = append(order, n.URL)
		}
		return order
	}()

	require.Equal(t, siteOrder, mcpOrder,
		"MCP and site search must produce the same ranking for the same query and corpus")
}

// siteSearchEnv adapts the MCP EnvMock to sitesearch.Env for parity testing:
// same retrieval dependencies, plus anonymous-visitor token and site config.
type siteSearchEnv struct {
	*EnvMock
}

func (siteSearchEnv) CurrentUserToken(context.Context) (*usertoken.Data, error) {
	return &usertoken.Data{}, nil
}

func (siteSearchEnv) SiteConfig(context.Context) appmodel.SiteConfig {
	return appmodel.SiteConfig{}
}
