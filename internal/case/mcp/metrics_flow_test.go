package mcp_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"trip2g/internal/appreq"
	"trip2g/internal/case/mcp"
	"trip2g/internal/db"
	"trip2g/internal/metrics"
	appmodel "trip2g/internal/model"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

// findRegMetric gathers the registry and returns the metric with the given
// name and exact label set, or nil if absent.
func findRegMetric(t *testing.T, g prometheus.Gatherer, name string, labels map[string]string) *dto.Metric {
	t.Helper()
	families, err := g.Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			got := map[string]string{}
			for _, lp := range m.GetLabel() {
				got[lp.GetName()] = lp.GetValue()
			}
			if len(got) != len(labels) {
				continue
			}
			match := true
			for k, v := range labels {
				if got[k] != v {
					match = false
					break
				}
			}
			if match {
				return m
			}
		}
	}
	return nil
}

func requireCounter(t *testing.T, g prometheus.Gatherer, name string, labels map[string]string, want float64) {
	t.Helper()
	m := findRegMetric(t, g, name, labels)
	require.NotNil(t, m, "metric %s with labels %v not found", name, labels)
	require.InDelta(t, want, m.GetCounter().GetValue(), 1e-9, "counter %s %v", name, labels)
}

// requireHistogram asserts the histogram holds exactly one sample of wantSum.
func requireHistogram(t *testing.T, g prometheus.Gatherer, name string, labels map[string]string, wantSum float64) {
	t.Helper()
	m := findRegMetric(t, g, name, labels)
	require.NotNil(t, m, "metric %s with labels %v not found", name, labels)
	require.Equal(t, uint64(1), m.GetHistogram().GetSampleCount(), "histogram count %s %v", name, labels)
	require.InDelta(t, wantSum, m.GetHistogram().GetSampleSum(), 1e-9, "histogram sum %s %v", name, labels)
}

// withSearchEnv wires the env funcs needed by the search tool: one readable
// note returned by text search, vector search disabled.
func withSearchEnv(env *EnvMock) {
	note := &appmodel.NoteView{Title: "Alpha", Path: "alpha.md", PathID: 1}
	textSearch := func(_ string) ([]appmodel.SearchResult, error) {
		return []appmodel.SearchResult{{NoteView: note, URL: "/alpha", HighlightedContent: []string{"snippet"}}}, nil
	}
	env.SearchLiveNotesFunc = textSearch
	env.SearchLatestNotesFunc = textSearch
	env.LiveNoteChunksFunc = func() []appmodel.NoteChunk { return nil }
	env.LatestNoteChunksFunc = func() []appmodel.NoteChunk { return nil }
	env.LiveNoteViewsFunc = appmodel.NewNoteViews
	if env.LatestNoteViewsFunc == nil {
		env.LatestNoteViewsFunc = appmodel.NewNoteViews
	}
	env.NoteURLFunc = func(n *appmodel.NoteView) string { return "https://x/" + n.Path }
}

// withFederationEnv wires two federation KB notes whose clients answer search.
func withFederationEnv(env *EnvMock) {
	kbA := &appmodel.NoteView{PathID: 11, MCPFederationKBURL: "https://a.example/_system/mcp", MCPFederationKBID: "alice"}
	kbB := &appmodel.NoteView{PathID: 12, MCPFederationKBURL: "https://b.example/_system/mcp", MCPFederationKBID: "bob"}
	nvs := appmodel.NewNoteViews()
	nvs.MCPFederationNotes = []*appmodel.MCPFederationNote{
		appmodel.NewMCPFederationNote(kbA),
		appmodel.NewMCPFederationNote(kbB),
	}
	env.LatestNoteViewsFunc = func() *appmodel.NoteViews { return nvs }
	env.FederationClientFunc = func(_ context.Context, _ string) (appmodel.Federation, error) {
		return &federationMock{
			searchFunc: func(_ context.Context, _ appmodel.MCPSearchParams) (appmodel.FederationResult, error) {
				return appmodel.FederationResult{Content: []appmodel.FederationContent{{Type: "text", Text: "remote"}}}, nil
			},
		}, nil
	}
}

func TestMCPEndpointMetrics(t *testing.T) {
	cases := []struct {
		name     string
		body     []byte
		headers  map[string]string
		resolver appreq.PersonalTokenResolver
		setup    func(t *testing.T, env *EnvMock)
		assert   func(t *testing.T, reg *prometheus.Registry)
	}{
		{
			name: "anonymous search records request, auth and result count",
			body: mcpToolsCallBody(t, "search", map[string]any{"query": "alpha"}),
			setup: func(_ *testing.T, env *EnvMock) {
				withSearchEnv(env)
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireCounter(t, reg, "trip2g_mcp_requests_total",
					map[string]string{"method": "tools/call", "tool": "search", "auth": "anonymous", "status": "ok"}, 1)
				requireCounter(t, reg, "trip2g_mcp_auth_total", map[string]string{"auth": "anonymous"}, 1)
				m := findRegMetric(t, reg, "trip2g_mcp_request_duration_seconds", map[string]string{"tool": "search"})
				require.NotNil(t, m, "duration histogram for tool=search not found")
				require.Equal(t, uint64(1), m.GetHistogram().GetSampleCount())
				requireHistogram(t, reg, "trip2g_mcp_search_results_returned", map[string]string{"tool": "search"}, 1)
			},
		},
		{
			name:    "api key tools call records auth=api_key",
			body:    mcpToolsCallBody(t, "search", map[string]any{"query": "alpha"}),
			headers: map[string]string{"X-API-Key": "test-api-key"},
			setup: func(_ *testing.T, env *EnvMock) {
				withSearchEnv(env)
				env.ResolveAPIKeyFunc = func(_ context.Context, _, _ string) (*db.ApiKey, error) {
					return &db.ApiKey{ID: 1}, nil
				}
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireCounter(t, reg, "trip2g_mcp_requests_total",
					map[string]string{"method": "tools/call", "tool": "search", "auth": "api_key", "status": "ok"}, 1)
				requireCounter(t, reg, "trip2g_mcp_auth_total", map[string]string{"auth": "api_key"}, 1)
			},
		},
		{
			name: "federated fan-out observes touched bases and outbound requests",
			body: mcpToolsCallBody(t, "federated_search", map[string]any{"query": "alpha"}),
			setup: func(_ *testing.T, env *EnvMock) {
				withFederationEnv(env)
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireHistogram(t, reg, "trip2g_mcp_fanout_bases", map[string]string{}, 2)
				requireCounter(t, reg, "trip2g_mcp_federated_requests_total", map[string]string{"peer": "alice", "status": "ok"}, 1)
				requireCounter(t, reg, "trip2g_mcp_federated_requests_total", map[string]string{"peer": "bob", "status": "ok"}, 1)
				requireCounter(t, reg, "trip2g_mcp_requests_total",
					map[string]string{"method": "tools/call", "tool": "federated_search", "auth": "anonymous", "status": "ok"}, 1)
			},
		},
		{
			name: "tool error records status=error and tool_errors_total",
			body: mcpToolsCallBody(t, "search", map[string]any{}),
			setup: func(_ *testing.T, env *EnvMock) {
				withSearchEnv(env)
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireCounter(t, reg, "trip2g_mcp_requests_total",
					map[string]string{"method": "tools/call", "tool": "search", "auth": "anonymous", "status": "error"}, 1)
				requireCounter(t, reg, "trip2g_mcp_tool_errors_total",
					map[string]string{"tool": "search", "reason": "invalid_params"}, 1)
			},
		},
		{
			name: "unknown tool is labeled other",
			body: mcpToolsCallBody(t, "totally_unknown_tool", map[string]any{}),
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireCounter(t, reg, "trip2g_mcp_requests_total",
					map[string]string{"method": "tools/call", "tool": "other", "auth": "anonymous", "status": "error"}, 1)
				requireCounter(t, reg, "trip2g_mcp_tool_errors_total",
					map[string]string{"tool": "other", "reason": "not_found"}, 1)
			},
		},
		{
			name: "tools list records discovery counter",
			body: mcpToolsListBody(t),
			setup: func(_ *testing.T, env *EnvMock) {
				dyn := &appmodel.NoteView{Path: "tools/my.md", MCPMethod: "my_tool", Title: "My tool"}
				env.LatestNoteViewsFunc = func() *appmodel.NoteViews {
					return &appmodel.NoteViews{
						List:    []*appmodel.NoteView{dyn},
						PathMap: map[string]*appmodel.NoteView{dyn.Path: dyn},
					}
				}
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireCounter(t, reg, "trip2g_mcp_tools_list_total", map[string]string{"auth": "anonymous"}, 1)
			},
		},
		{
			name:    "federation depth header is observed",
			body:    mcpInitBody,
			headers: map[string]string{"X-MCP-Federation-Depth": "2"},
			setup: func(_ *testing.T, env *EnvMock) {
				env.FederationMaxDepthFunc = func() int { return 5 }
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireHistogram(t, reg, "trip2g_mcp_federation_depth", map[string]string{}, 2)
			},
		},
		{
			name: "direct request without depth header observes depth 0",
			body: mcpInitBody,
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireHistogram(t, reg, "trip2g_mcp_federation_depth", map[string]string{}, 0)
			},
		},
		{
			name:    "malformed depth header observes depth 0",
			body:    mcpInitBody,
			headers: map[string]string{"X-MCP-Federation-Depth": "abc"},
			setup: func(_ *testing.T, env *EnvMock) {
				env.FederationMaxDepthFunc = func() int { return 5 }
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireHistogram(t, reg, "trip2g_mcp_federation_depth", map[string]string{}, 0)
			},
		},
		{
			name:    "negative depth header observes depth 0",
			body:    mcpInitBody,
			headers: map[string]string{"X-MCP-Federation-Depth": "-3"},
			setup: func(_ *testing.T, env *EnvMock) {
				env.FederationMaxDepthFunc = func() int { return 5 }
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireHistogram(t, reg, "trip2g_mcp_federation_depth", map[string]string{}, 0)
			},
		},
		{
			name: "dynamic tool call is labeled dynamic, never by frontmatter name",
			body: mcpToolsCallBody(t, "my_dynamic_tool", map[string]any{}),
			setup: func(_ *testing.T, env *EnvMock) {
				dyn := &appmodel.NoteView{Path: "tools/my.md", MCPMethod: "my_dynamic_tool", Content: []byte("body")}
				env.LatestNoteViewsFunc = func() *appmodel.NoteViews {
					return &appmodel.NoteViews{
						List:    []*appmodel.NoteView{dyn},
						PathMap: map[string]*appmodel.NoteView{dyn.Path: dyn},
					}
				}
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireCounter(t, reg, "trip2g_mcp_requests_total",
					map[string]string{"method": "tools/call", "tool": "dynamic", "auth": "anonymous", "status": "ok"}, 1)
				require.Nil(t, findRegMetric(t, reg, "trip2g_mcp_requests_total",
					map[string]string{"method": "tools/call", "tool": "my_dynamic_tool", "auth": "anonymous", "status": "ok"}),
					"frontmatter-derived tool name must not become a label value")
			},
		},
		{
			name: "single-kb federation client failure counts an outbound error",
			body: mcpToolsCallBody(t, "federated_search", map[string]any{"query": "alpha", "kb_id": "alice"}),
			setup: func(_ *testing.T, env *EnvMock) {
				withFederationEnv(env)
				env.FederationClientFunc = func(_ context.Context, _ string) (appmodel.Federation, error) {
					return nil, errors.New("no active federation secret")
				}
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireCounter(t, reg, "trip2g_mcp_federated_requests_total", map[string]string{"peer": "alice", "status": "error"}, 1)
			},
		},
		{
			name:     "invalid personal token records rejected request",
			body:     mcpInitBody,
			headers:  map[string]string{"Authorization": "Bearer t2g_revoked"},
			resolver: &testPersonalTokenResolver{err: errors.New("token revoked")},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireCounter(t, reg, "trip2g_mcp_requests_total",
					map[string]string{"method": "initialize", "tool": "", "auth": "token", "status": "error"}, 1)
				requireCounter(t, reg, "trip2g_mcp_auth_total", map[string]string{"auth": "token"}, 1)
			},
		},
		{
			name:    "exceeded federation depth records rejected request",
			body:    mcpInitBody,
			headers: map[string]string{"X-MCP-Federation-Depth": "6"},
			setup: func(_ *testing.T, env *EnvMock) {
				env.FederationMaxDepthFunc = func() int { return 5 }
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireCounter(t, reg, "trip2g_mcp_requests_total",
					map[string]string{"method": "initialize", "tool": "", "auth": "anonymous", "status": "error"}, 1)
				requireCounter(t, reg, "trip2g_mcp_auth_total", map[string]string{"auth": "anonymous"}, 1)
			},
		},
		{
			name:    "invalid api key records rejected request as auth=api_key",
			body:    mcpInitBody,
			headers: map[string]string{"X-API-Key": "bad-key"},
			setup: func(_ *testing.T, env *EnvMock) {
				env.ResolveAPIKeyFunc = func(_ context.Context, _, _ string) (*db.ApiKey, error) {
					return nil, errors.New("invalid API key")
				}
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireCounter(t, reg, "trip2g_mcp_requests_total",
					map[string]string{"method": "initialize", "tool": "", "auth": "api_key", "status": "error"}, 1)
				requireCounter(t, reg, "trip2g_mcp_auth_total", map[string]string{"auth": "api_key"}, 1)
			},
		},
		{
			name:    "failed federation bearer records rejected request as auth=federation",
			body:    mcpInitBody,
			headers: map[string]string{"Authorization": "Basic not-a-bearer"},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireCounter(t, reg, "trip2g_mcp_requests_total",
					map[string]string{"method": "initialize", "tool": "", "auth": "federation", "status": "error"}, 1)
				requireCounter(t, reg, "trip2g_mcp_auth_total", map[string]string{"auth": "federation"}, 1)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg := prometheus.NewRegistry()
			m := metrics.NewMCPMetrics(reg)

			env := buildDispatchEnv(t, false)
			env.MCPMetricsFunc = func() *metrics.MCPMetrics { return m }
			if tc.setup != nil {
				tc.setup(t, env)
			}

			fasthttpCtx := buildMCPFasthttpCtx(tc.body, "")
			for k, v := range tc.headers {
				fasthttpCtx.Request.Header.Set(k, v)
			}
			req := wiredRequest(fasthttpCtx, env, tc.resolver)
			defer appreq.Release(req)

			_, err := (&mcp.Endpoint{}).Handle(req)
			require.NoError(t, err)

			var resp mcp.Response
			require.NoError(t, json.Unmarshal(fasthttpCtx.Response.Body(), &resp))

			tc.assert(t, reg)
		})
	}
}

// TestDynamicToolLabelBounded pins the cardinality bound: calls to different
// mcp_method tools share one "dynamic" series instead of minting one series
// per author-controlled frontmatter name.
func TestDynamicToolLabelBounded(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewMCPMetrics(reg)

	dynA := &appmodel.NoteView{Path: "a.md", MCPMethod: "tool_a", Content: []byte("a")}
	dynB := &appmodel.NoteView{Path: "b.md", MCPMethod: "tool_b", Content: []byte("b")}
	env := buildDispatchEnv(t, false)
	env.MCPMetricsFunc = func() *metrics.MCPMetrics { return m }
	env.LatestNoteViewsFunc = func() *appmodel.NoteViews {
		return &appmodel.NoteViews{
			List:    []*appmodel.NoteView{dynA, dynB},
			PathMap: map[string]*appmodel.NoteView{dynA.Path: dynA, dynB.Path: dynB},
		}
	}

	for _, tool := range []string{"tool_a", "tool_b"} {
		fasthttpCtx := buildMCPFasthttpCtx(mcpToolsCallBody(t, tool, map[string]any{}), "")
		req := wiredRequest(fasthttpCtx, env, nil)
		_, err := (&mcp.Endpoint{}).Handle(req)
		appreq.Release(req)
		require.NoError(t, err)
	}

	requireCounter(t, reg, "trip2g_mcp_requests_total",
		map[string]string{"method": "tools/call", "tool": "dynamic", "auth": "anonymous", "status": "ok"}, 2)
	for _, name := range []string{"tool_a", "tool_b"} {
		require.Nil(t, findRegMetric(t, reg, "trip2g_mcp_requests_total",
			map[string]string{"method": "tools/call", "tool": name, "auth": "anonymous", "status": "ok"}),
			"per-name series %q must not exist", name)
	}
}

// federationBearer signs an HS256 federation JWT for kid the way a peer does.
func federationBearer(t *testing.T, secret []byte, kid string) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT", "kid": kid})
	require.NoError(t, err)
	now := time.Now().Unix()
	claims, err := json.Marshal(map[string]any{"iss": "https://peer.example", "iat": now, "exp": now + 30, "rid": "r"})
	require.NoError(t, err)
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(unsigned))
	return "Bearer " + unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// withInboundKey makes the base accept kid, scoped to subgraphs.
func withInboundKey(env *EnvMock, secret []byte, kid string, subgraphs []string) {
	env.FederationSecretByKIDFunc = func(_ context.Context, got string) (db.FederationSecret, bool, error) {
		if got != kid {
			return db.FederationSecret{}, false, nil
		}
		return db.FederationSecret{ID: 1, Kid: kid, SecretCrypt: secret}, true, nil
	}
	env.DecryptDataFunc = func(b []byte) ([]byte, error) { return b, nil }
	env.ListFederationSecretSubgraphsByKIDFunc = func(_ context.Context, _ string) ([]string, error) {
		return subgraphs, nil
	}
}

func scopedNote(pathID int64, path string, subgraphs ...string) *appmodel.NoteView {
	return &appmodel.NoteView{
		Path:          path,
		PathID:        pathID,
		Title:         path,
		Permalink:     "/" + path,
		HTML:          "<h1>Title</h1><h2>Part</h2><p>body</p>",
		SubgraphNames: subgraphs,
	}
}

func withNotes(env *EnvMock, notes ...*appmodel.NoteView) {
	nvs := &appmodel.NoteViews{PathMap: map[string]*appmodel.NoteView{}}
	for _, note := range notes {
		nvs.List = append(nvs.List, note)
		nvs.PathMap[note.Path] = note
	}
	env.LatestNoteViewsFunc = func() *appmodel.NoteViews { return nvs }
	env.LatestNoteChunksFunc = func() []appmodel.NoteChunk { return nil }
	env.NoteURLFunc = func(n *appmodel.NoteView) string { return "https://x/" + n.Path }
}

func TestMCPFederatedInboundMetrics(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")

	cases := []struct {
		name   string
		body   []byte
		auth   bool
		setup  func(env *EnvMock)
		assert func(t *testing.T, reg *prometheus.Registry)
	}{
		{
			name: "search by a peer counts the request and each result under its subgraph",
			body: mcpToolsCallBody(t, "search", map[string]any{"query": "alpha"}),
			auth: true,
			setup: func(env *EnvMock) {
				docs := scopedNote(1, "docs.md", "docs")
				both := scopedNote(2, "both.md", "sales", "docs")
				hidden := scopedNote(3, "hidden.md", "private")
				textSearch := func(_ string) ([]appmodel.SearchResult, error) {
					return []appmodel.SearchResult{
						{NoteView: docs, HighlightedContent: []string{"a"}},
						{NoteView: both, HighlightedContent: []string{"a"}},
						{NoteView: hidden, HighlightedContent: []string{"a"}},
					}, nil
				}
				withSearchEnv(env)
				env.SearchLiveNotesFunc = textSearch
				env.SearchLatestNotesFunc = textSearch
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireCounter(t, reg, "trip2g_mcp_requests_total",
					map[string]string{"method": "tools/call", "tool": "search", "auth": "federation", "status": "ok"}, 1)
				requireCounter(t, reg, "trip2g_mcp_federated_inbound_requests_total",
					map[string]string{"peer": "partner", "tool": "search"}, 1)
				requireCounter(t, reg, "trip2g_mcp_federated_results_served_total",
					map[string]string{"peer": "partner", "subgraph": "docs"}, 2)
				require.Nil(t, findRegMetric(t, reg, "trip2g_mcp_federated_results_served_total",
					map[string]string{"peer": "partner", "subgraph": "private"}),
					"a result the key cannot read is never returned, so never counted")
			},
		},
		{
			name: "note_html by a peer counts the note under the subgraph the key matched",
			body: mcpToolsCallBody(t, "note_html", map[string]any{"path": "both.md"}),
			auth: true,
			setup: func(env *EnvMock) {
				withNotes(env, scopedNote(2, "both.md", "sales", "docs"))
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireCounter(t, reg, "trip2g_mcp_federated_inbound_requests_total",
					map[string]string{"peer": "partner", "tool": "note_html"}, 1)
				requireCounter(t, reg, "trip2g_mcp_federated_notes_served_total",
					map[string]string{"peer": "partner", "subgraph": "docs"}, 1)
			},
		},
		{
			name: "expand by a peer counts the note it opened",
			body: mcpToolsCallBody(t, "expand", map[string]any{"path": "docs.md"}),
			auth: true,
			setup: func(env *EnvMock) {
				withNotes(env, scopedNote(1, "docs.md", "docs"))
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireCounter(t, reg, "trip2g_mcp_federated_notes_served_total",
					map[string]string{"peer": "partner", "subgraph": "docs"}, 1)
			},
		},
		{
			name: "a note outside every subgraph is counted as none",
			body: mcpToolsCallBody(t, "note_html", map[string]any{"path": "open.md"}),
			auth: true,
			setup: func(env *EnvMock) {
				withNotes(env, scopedNote(4, "open.md"))
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireCounter(t, reg, "trip2g_mcp_federated_notes_served_total",
					map[string]string{"peer": "partner", "subgraph": "none"}, 1)
			},
		},
		{
			name: "a refused read is not counted as served",
			body: mcpToolsCallBody(t, "note_html", map[string]any{"path": "hidden.md"}),
			auth: true,
			setup: func(env *EnvMock) {
				withNotes(env, scopedNote(3, "hidden.md", "private"))
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireCounter(t, reg, "trip2g_mcp_federated_inbound_requests_total",
					map[string]string{"peer": "partner", "tool": "note_html"}, 1)
				require.Nil(t, findRegMetric(t, reg, "trip2g_mcp_federated_notes_served_total",
					map[string]string{"peer": "partner", "subgraph": "none"}))
				require.Nil(t, findRegMetric(t, reg, "trip2g_mcp_federated_notes_served_total",
					map[string]string{"peer": "partner", "subgraph": "private"}))
			},
		},
		{
			name: "a non-tool request by a peer is labeled by its method",
			body: mcpInitBody,
			auth: true,
			assert: func(t *testing.T, reg *prometheus.Registry) {
				requireCounter(t, reg, "trip2g_mcp_federated_inbound_requests_total",
					map[string]string{"peer": "partner", "tool": "initialize"}, 1)
			},
		},
		{
			name: "an anonymous read records no federated series",
			body: mcpToolsCallBody(t, "note_html", map[string]any{"path": "open.md"}),
			setup: func(env *EnvMock) {
				withNotes(env, scopedNote(4, "open.md"))
			},
			assert: func(t *testing.T, reg *prometheus.Registry) {
				families, err := reg.Gather()
				require.NoError(t, err)
				for _, mf := range families {
					switch mf.GetName() {
					case "trip2g_mcp_federated_inbound_requests_total",
						"trip2g_mcp_federated_notes_served_total",
						"trip2g_mcp_federated_results_served_total":
						require.Empty(t, mf.GetMetric(), mf.GetName())
					}
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg := prometheus.NewRegistry()
			m := metrics.NewMCPMetrics(reg)

			env := buildDispatchEnv(t, false)
			env.MCPMetricsFunc = func() *metrics.MCPMetrics { return m }
			withInboundKey(env, secret, "partner", []string{"docs"})
			if tc.setup != nil {
				tc.setup(env)
			}

			authHeader := ""
			if tc.auth {
				authHeader = federationBearer(t, secret, "partner")
			}
			fasthttpCtx := buildMCPFasthttpCtx(tc.body, authHeader)
			req := wiredRequest(fasthttpCtx, env, nil)
			defer appreq.Release(req)

			_, err := (&mcp.Endpoint{}).Handle(req)
			require.NoError(t, err)

			tc.assert(t, reg)
		})
	}
}
