package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

// findMetric gathers the registry and returns the metric with the given name
// and exact label set, or fails the test.
func findMetric(t *testing.T, g prometheus.Gatherer, name string, labels map[string]string) *dto.Metric {
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
	t.Fatalf("metric %s with labels %v not found", name, labels)
	return nil
}

func TestNewMCPMetrics_Records(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	u := func(v uint64) *uint64 { return &v }

	cases := []struct {
		name   string
		record func(m *MCPMetrics)
		metric string
		labels map[string]string

		wantCounter   *float64
		wantGauge     *float64
		wantHistCount *uint64
		wantHistSum   *float64
	}{
		{
			name:        "requests_total counts by method tool auth status",
			record:      func(m *MCPMetrics) { m.RecordMCPRequest("tools/call", "search", "anonymous", "ok", 0.01) },
			metric:      "trip2g_mcp_requests_total",
			labels:      map[string]string{"method": "tools/call", "tool": "search", "auth": "anonymous", "status": "ok"},
			wantCounter: f(1),
		},
		{
			name:          "request_duration observes per tool",
			record:        func(m *MCPMetrics) { m.RecordMCPRequest("tools/call", "search", "anonymous", "ok", 0.25) },
			metric:        "trip2g_mcp_request_duration_seconds",
			labels:        map[string]string{"tool": "search"},
			wantHistCount: u(1),
			wantHistSum:   f(0.25),
		},
		{
			name:          "request_duration falls back to method when tool empty",
			record:        func(m *MCPMetrics) { m.RecordMCPRequest("initialize", "", "token", "ok", 0.5) },
			metric:        "trip2g_mcp_request_duration_seconds",
			labels:        map[string]string{"tool": "initialize"},
			wantHistCount: u(1),
			wantHistSum:   f(0.5),
		},
		{
			name:        "auth_total counts by auth kind",
			record:      func(m *MCPMetrics) { m.RecordMCPRequest("tools/list", "", "anonymous", "ok", 0.01) },
			metric:      "trip2g_mcp_auth_total",
			labels:      map[string]string{"auth": "anonymous"},
			wantCounter: f(1),
		},
		{
			name:        "tools_list_total counts by auth kind",
			record:      func(m *MCPMetrics) { m.RecordToolsList("api_key") },
			metric:      "trip2g_mcp_tools_list_total",
			labels:      map[string]string{"auth": "api_key"},
			wantCounter: f(1),
		},
		{
			name:        "tool_errors_total counts by tool and reason",
			record:      func(m *MCPMetrics) { m.RecordToolError("search", "invalid_params") },
			metric:      "trip2g_mcp_tool_errors_total",
			labels:      map[string]string{"tool": "search", "reason": "invalid_params"},
			wantCounter: f(1),
		},
		{
			name:          "federation_depth observes incoming depth",
			record:        func(m *MCPMetrics) { m.ObserveFederationDepth(3) },
			metric:        "trip2g_mcp_federation_depth",
			labels:        map[string]string{},
			wantHistCount: u(1),
			wantHistSum:   f(3),
		},
		{
			name:          "fanout_bases observes touched bases",
			record:        func(m *MCPMetrics) { m.ObserveFanoutBases(4) },
			metric:        "trip2g_mcp_fanout_bases",
			labels:        map[string]string{},
			wantHistCount: u(1),
			wantHistSum:   f(4),
		},
		{
			name:        "federated_requests_total counts by status",
			record:      func(m *MCPMetrics) { m.RecordFederatedRequest("timeout") },
			metric:      "trip2g_mcp_federated_requests_total",
			labels:      map[string]string{"status": "timeout"},
			wantCounter: f(1),
		},
		{
			name:          "search_results_returned observes per tool",
			record:        func(m *MCPMetrics) { m.ObserveSearchResults("search", 7) },
			metric:        "trip2g_mcp_search_results_returned",
			labels:        map[string]string{"tool": "search"},
			wantHistCount: u(1),
			wantHistSum:   f(7),
		},
		{
			name:      "dynamic_tools_registered gauge is set",
			record:    func(m *MCPMetrics) { m.SetDynamicToolsRegistered(5) },
			metric:    "trip2g_mcp_dynamic_tools_registered",
			labels:    map[string]string{},
			wantGauge: f(5),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg := prometheus.NewRegistry()
			m := NewMCPMetrics(reg)
			tc.record(m)

			metric := findMetric(t, reg, tc.metric, tc.labels)
			if tc.wantCounter != nil {
				require.InDelta(t, *tc.wantCounter, metric.GetCounter().GetValue(), 1e-9)
			}
			if tc.wantGauge != nil {
				require.InDelta(t, *tc.wantGauge, metric.GetGauge().GetValue(), 1e-9)
			}
			if tc.wantHistCount != nil {
				require.Equal(t, *tc.wantHistCount, metric.GetHistogram().GetSampleCount())
			}
			if tc.wantHistSum != nil {
				require.InDelta(t, *tc.wantHistSum, metric.GetHistogram().GetSampleSum(), 1e-9)
			}
		})
	}
}

func TestMCPMetrics_NilSafe(t *testing.T) {
	var m *MCPMetrics
	require.NotPanics(t, func() {
		m.RecordMCPRequest("tools/call", "search", "anonymous", "ok", 0.01)
		m.RecordToolsList("anonymous")
		m.RecordToolError("search", "internal")
		m.ObserveFederationDepth(1)
		m.ObserveFanoutBases(2)
		m.RecordFederatedRequest("ok")
		m.ObserveSearchResults("similar", 3)
		m.SetDynamicToolsRegistered(4)
	})
}
