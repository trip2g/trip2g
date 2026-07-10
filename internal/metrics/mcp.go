package metrics

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

// MCPMetrics collects Prometheus metrics for the MCP endpoint (/_system/mcp).
// All record methods are nil-safe so callers without wired metrics (tests,
// partial app setups) can call them unconditionally.
type MCPMetrics struct {
	requests          *prometheus.CounterVec
	duration          *prometheus.HistogramVec
	fanoutBases       prometheus.Histogram
	federationDepth   prometheus.Histogram
	federatedRequests *prometheus.CounterVec
	searchResults     *prometheus.HistogramVec
	toolErrors        *prometheus.CounterVec
	auth              *prometheus.CounterVec
	dynamicTools      prometheus.GaugeFunc
	dynamicToolsFn    atomic.Pointer[func() int]
	toolsList         *prometheus.CounterVec
}

// NewMCPMetrics creates and registers all MCP Prometheus metrics on reg.
func NewMCPMetrics(reg prometheus.Registerer) *MCPMetrics {
	m := &MCPMetrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "trip2g_mcp_requests_total",
			Help: "Total number of MCP JSON-RPC requests",
		}, []string{"method", "tool", "auth", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "trip2g_mcp_request_duration_seconds",
			Help:    "MCP request duration in seconds, per tool (or JSON-RPC method for non-tool calls)",
			Buckets: prometheus.DefBuckets,
		}, []string{"tool"}),
		fanoutBases: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "trip2g_mcp_fanout_bases",
			Help:    "Number of knowledge bases successfully reached by a federated fan-out query",
			Buckets: []float64{0, 1, 2, 3, 5, 8, 13, 21},
		}),
		federationDepth: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "trip2g_mcp_federation_depth",
			Help:    "X-MCP-Federation-Depth of incoming MCP requests (0 = direct client)",
			Buckets: []float64{0, 1, 2, 3, 4, 5, 8},
		}),
		federatedRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "trip2g_mcp_federated_requests_total",
			Help: "Total number of outbound federated MCP requests to peer hubs",
		}, []string{"status"}),
		searchResults: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "trip2g_mcp_search_results_returned",
			Help:    "Number of results returned by MCP search tools",
			Buckets: []float64{0, 1, 3, 5, 10, 20, 50},
		}, []string{"tool"}),
		toolErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "trip2g_mcp_tool_errors_total",
			Help: "Total number of MCP tools/call errors by tool and reason",
		}, []string{"tool", "reason"}),
		auth: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "trip2g_mcp_auth_total",
			Help: "Total number of MCP requests by auth kind",
		}, []string{"auth"}),
		toolsList: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "trip2g_mcp_tools_list_total",
			Help: "Total number of MCP tools/list calls (discovery signal)",
		}, []string{"auth"}),
	}

	// Computed at scrape time from the source set via SetDynamicToolsSource, so
	// the value always reflects the currently published note snapshot: no
	// staleness after reloads and no ordering races between concurrent writers.
	m.dynamicTools = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "trip2g_mcp_dynamic_tools_registered",
		Help: "Number of dynamic per-note (mcp_method) tools currently exposed",
	}, func() float64 {
		if fn := m.dynamicToolsFn.Load(); fn != nil {
			return float64((*fn)())
		}
		return 0
	})

	reg.MustRegister(
		m.requests, m.duration, m.fanoutBases, m.federationDepth,
		m.federatedRequests, m.searchResults, m.toolErrors, m.auth,
		m.dynamicTools, m.toolsList,
	)
	return m
}

// SetDynamicToolsSource wires the function the dynamic-tools gauge reads on
// every scrape.
func (m *MCPMetrics) SetDynamicToolsSource(fn func() int) {
	if m == nil || fn == nil {
		return
	}
	m.dynamicToolsFn.Store(&fn)
}

// RecordMCPRequest records one MCP request: requests_total, auth_total and the
// duration histogram. The duration is labeled by tool, falling back to the
// JSON-RPC method for non-tool calls.
func (m *MCPMetrics) RecordMCPRequest(method, tool, auth, status string, seconds float64) {
	if m == nil {
		return
	}
	m.requests.WithLabelValues(method, tool, auth, status).Inc()
	m.auth.WithLabelValues(auth).Inc()
	durationTool := tool
	if durationTool == "" {
		durationTool = method
	}
	m.duration.WithLabelValues(durationTool).Observe(seconds)
}

// RecordToolsList counts a tools/list call by auth kind.
func (m *MCPMetrics) RecordToolsList(auth string) {
	if m == nil {
		return
	}
	m.toolsList.WithLabelValues(auth).Inc()
}

// RecordToolError counts a failed tools/call by tool and bounded reason.
func (m *MCPMetrics) RecordToolError(tool, reason string) {
	if m == nil {
		return
	}
	m.toolErrors.WithLabelValues(tool, reason).Inc()
}

// ObserveFederationDepth observes the incoming X-MCP-Federation-Depth.
func (m *MCPMetrics) ObserveFederationDepth(depth int) {
	if m == nil {
		return
	}
	m.federationDepth.Observe(float64(depth))
}

// ObserveFanoutBases observes how many bases a federated fan-out query reached.
func (m *MCPMetrics) ObserveFanoutBases(n int) {
	if m == nil {
		return
	}
	m.fanoutBases.Observe(float64(n))
}

// RecordFederatedRequest counts one outbound federated request (ok|error|timeout).
func (m *MCPMetrics) RecordFederatedRequest(status string) {
	if m == nil {
		return
	}
	m.federatedRequests.WithLabelValues(status).Inc()
}

// ObserveSearchResults observes the result count returned by a search tool.
func (m *MCPMetrics) ObserveSearchResults(tool string, n int) {
	if m == nil {
		return
	}
	m.searchResults.WithLabelValues(tool).Observe(float64(n))
}
