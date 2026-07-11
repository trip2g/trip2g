package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

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
			searchFunc: func(_ context.Context, _ appmodel.FederationSearchParams) (appmodel.FederationResult, error) {
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
				requireCounter(t, reg, "trip2g_mcp_federated_requests_total", map[string]string{"status": "ok"}, 2)
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
				requireCounter(t, reg, "trip2g_mcp_federated_requests_total", map[string]string{"status": "error"}, 1)
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
