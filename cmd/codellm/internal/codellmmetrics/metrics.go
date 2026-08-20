// Package codellmmetrics is codellm's Prometheus surface: the collectors plus
// the internal listener's handler (/metrics, pprof, liveness/readiness).
//
// It owns its OWN registry rather than the global default one, so codellm stays
// independent of the monolith's internal/metrics (which registers globally) and
// a test can gather a clean snapshot. Every record method is nil-safe, so call
// sites stay unconditional when metrics are disabled (tests, --metrics-addr="").
package codellmmetrics

import (
	"net/http"
	"net/http/pprof"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"trip2g/cmd/codellm/internal/coderun"
)

// Auth lanes and results reported by RecordAuth.
const (
	LaneAPIKey  = "apikey"
	LaneCookie  = "cookie"
	Allowed     = "allowed"
	Denied      = "denied"
	statusLabel = "status"
)

// Metrics holds codellm's collectors and the registry they live in.
type Metrics struct {
	reg *prometheus.Registry

	requests  *prometheus.CounterVec
	duration  *prometheus.HistogramVec
	inFlight  prometheus.Gauge
	auth      *prometheus.CounterVec
	execError *prometheus.CounterVec

	blocks          *prometheus.CounterVec
	blockExitCodes  *prometheus.CounterVec
	blockDuration   *prometheus.HistogramVec
	blockMaxRSS     *prometheus.HistogramVec
	blockStdout     *prometheus.HistogramVec
	blockTruncated  *prometheus.CounterVec
	sandboxFallback *prometheus.CounterVec
	requestBlocks   prometheus.Histogram

	changes    *prometheus.CounterVec
	configInfo *prometheus.GaugeVec
}

// New builds the collectors on a private registry, together with the standard
// Go runtime and process collectors (goroutines, GC, open fds, RSS).
func New() *Metrics {
	m := &Metrics{
		reg: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "codellm_requests_total",
			Help: "Total HTTP requests served by codellm, by endpoint and status code",
		}, []string{"endpoint", statusLabel}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "codellm_request_duration_seconds",
			Help:    "codellm HTTP request duration in seconds, by endpoint",
			Buckets: prometheus.DefBuckets,
		}, []string{"endpoint"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "codellm_requests_in_flight",
			Help: "Requests currently being served; each one may fork an interpreter child",
		}),
		auth: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "codellm_auth_total",
			Help: "Auth decisions on the browser-facing endpoints, by lane (apikey|cookie) and result; denials on a code-executing endpoint are a probing signal",
		}, []string{"lane", "result"}),
		execError: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "codellm_exec_errors_total",
			Help: "Code-execution failures by kind (unknown_fence, disallowed_program, sandbox_refused, timeout, parse_error, ...)",
		}, []string{"kind"}),
		blocks: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "codellm_blocks_total",
			Help: "Executed code blocks by interpreter and outcome (ok|nonzero_exit|timeout|start_failed)",
		}, []string{"program", "outcome"}),
		blockExitCodes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "codellm_block_exit_codes_total",
			Help: "Executed code blocks by interpreter and process exit code (-1 = never produced one)",
		}, []string{"program", "exit_code"}),
		blockDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "codellm_block_duration_seconds",
			Help:    "Wall time of one executed code block in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5, 15, 30, 60, 300},
		}, []string{"program"}),
		blockMaxRSS: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "codellm_block_max_rss_bytes",
			Help:    "Peak resident memory of one executed code block in bytes",
			Buckets: prometheus.ExponentialBuckets(1<<20, 4, 8),
		}, []string{"program"}),
		blockStdout: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "codellm_block_stdout_bytes",
			Help:    "Captured stdout size of one executed code block in bytes",
			Buckets: prometheus.ExponentialBuckets(256, 4, 8),
		}, []string{"program"}),
		blockTruncated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "codellm_block_stdout_truncated_total",
			Help: "Code blocks whose stdout hit the cap and was cut (the overflow is dropped, which later reads as a parse error)",
		}, []string{"program"}),
		sandboxFallback: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "codellm_sandbox_fallbacks_total",
			Help: "Blocks a besteffort policy degraded to unsandboxed execution, by reason",
		}, []string{"reason"}),
		requestBlocks: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "codellm_request_blocks",
			Help:    "Number of fenced code blocks executed per request",
			Buckets: []float64{1, 2, 3, 5, 8, 13, 21},
		}),
		changes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "codellm_changes_total",
			Help: "Note changes returned as tool_calls, by kind (write|patch)",
		}, []string{"kind"}),
		configInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "codellm_config_info",
			Help: "Always 1; labels carry the execution posture this process runs with",
		}, []string{"sandbox_mode", "sandbox_network", "allowed_programs"}),
	}

	m.reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.requests, m.duration, m.inFlight, m.auth, m.execError,
		m.blocks, m.blockExitCodes, m.blockDuration, m.blockMaxRSS,
		m.blockStdout, m.blockTruncated, m.sandboxFallback, m.requestBlocks,
		m.changes, m.configInfo,
	)
	return m
}

// Registry exposes the private registry for tests and for callers that want to
// register their own collectors alongside codellm's.
func (m *Metrics) Registry() *prometheus.Registry {
	if m == nil {
		return nil
	}
	return m.reg
}

// Handler builds the internal listener's mux: Prometheus scrape, pprof, and
// the k8s-convention probes. It mirrors the monolith's internal listener
// (cmd/server: /metrics + /debug/pprof/* + /healthz + /livez + /readyz), which
// is why the whole port must stay loopback-bound: none of it is authenticated.
// ready may be nil (always ready).
func (m *Metrics) Handler(ready func() bool) http.Handler {
	mux := http.NewServeMux()
	if m != nil {
		mux.Handle("GET /metrics", promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{Registry: m.reg}))
	}

	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("alive"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		if ready != nil && !ready() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	return mux
}

// Instrument wraps next with the request lane: in-flight gauge, duration, and
// the status-code counter under the given endpoint label.
func (m *Metrics) Instrument(endpoint string, next http.Handler) http.Handler {
	if m == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.inFlight.Inc()
		defer m.inFlight.Dec()

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		timer := prometheus.NewTimer(m.duration.WithLabelValues(endpoint))
		next.ServeHTTP(rec, r)
		timer.ObserveDuration()
		m.requests.WithLabelValues(endpoint, strconv.Itoa(rec.status)).Inc()
	})
}

// RecordAuth counts one auth decision: lane is LaneAPIKey or LaneCookie,
// result is Allowed or Denied.
func (m *Metrics) RecordAuth(lane, result string) {
	if m == nil {
		return
	}
	m.auth.WithLabelValues(lane, result).Inc()
}

// RecordBlock records one executed code block. It is the coderun.CodeInput
// Observe seam, so it takes what coderun measured verbatim.
func (m *Metrics) RecordBlock(s coderun.BlockStats) {
	if m == nil {
		return
	}
	program := s.Program
	if program == "" {
		program = "unknown"
	}
	m.blocks.WithLabelValues(program, s.Outcome).Inc()
	m.blockExitCodes.WithLabelValues(program, strconv.Itoa(s.ExitCode)).Inc()
	m.blockDuration.WithLabelValues(program).Observe(float64(s.DurationMs) / 1000)
	if s.MaxRSSBytes > 0 {
		m.blockMaxRSS.WithLabelValues(program).Observe(float64(s.MaxRSSBytes))
	}
	m.blockStdout.WithLabelValues(program).Observe(float64(s.StdoutBytes))
	if s.StdoutTruncated {
		m.blockTruncated.WithLabelValues(program).Inc()
	}
	if s.SandboxFallback != "" {
		m.sandboxFallback.WithLabelValues(s.SandboxFallback).Inc()
	}
}

// RecordExecError counts one failed run by coderun's failure kind. A no-op for
// a nil error so the call site stays a single unconditional line.
func (m *Metrics) RecordExecError(err error) {
	if m == nil || err == nil {
		return
	}
	m.execError.WithLabelValues(coderun.ErrorKind(err)).Inc()
}

// ObserveRequestBlocks records how many blocks one request executed.
func (m *Metrics) ObserveRequestBlocks(n int) {
	if m == nil {
		return
	}
	m.requestBlocks.Observe(float64(n))
}

// RecordChange counts one returned note change.
func (m *Metrics) RecordChange(kind string) {
	if m == nil {
		return
	}
	m.changes.WithLabelValues(kind).Inc()
}

// SetConfigInfo publishes the execution posture this process runs with, so an
// operator can see sandbox mode, network access and the interpreter allowlist
// from a scrape instead of from the box.
func (m *Metrics) SetConfigInfo(sandboxMode, sandboxNetwork, allowedPrograms string) {
	if m == nil {
		return
	}
	m.configInfo.WithLabelValues(sandboxMode, sandboxNetwork, allowedPrograms).Set(1)
}

// statusRecorder captures the response status for the request counter.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Unwrap keeps http.ResponseController able to reach the real writer, so
// wrapping does not quietly disable flushing or hijacking for a future
// streaming endpoint.
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
