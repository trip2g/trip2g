// Package fleetmetrics is the fleet's Prometheus surface: the collectors plus
// the internal listener's handler (/metrics, pprof, liveness/readiness).
// Catalog and alerting recipes live in docs/dev/fleet_codellm_metrics.md.
//
// It owns its OWN registry rather than the global default one, so the fleet
// stays independent of the monolith's internal/metrics (which registers
// globally) and a test can gather a clean snapshot. Every record method is
// nil-safe, so call sites stay unconditional when metrics are disabled (tests,
// --once, --metrics-addr="").
package fleetmetrics

import (
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Delivery kinds and the LLM lanes, used as label values.
const (
	KindChange = "change"
	KindCron   = "cron"

	// LaneLLM is the role's own model endpoint; LaneExec is the exec tool's
	// endpoint (codellm). Splitting them keeps a degrading code executor
	// distinguishable from a degrading language model.
	LaneLLM  = "llm"
	LaneExec = "exec"

	// StatusOK / StatusError are the coarse outcomes shared by several lanes.
	// StatusPartial is a sync that refreshed the registry but could not use every
	// role note it found.
	StatusOK      = "ok"
	StatusError   = "error"
	StatusPartial = "partial"

	roleLabel   = "role"
	statusLabel = "status"
	modelLabel  = "model"
	laneLabel   = "lane"
)

// Metrics holds the fleet's collectors and the registry they live in.
type Metrics struct {
	reg *prometheus.Registry

	deliveries       *prometheus.CounterVec
	deliveryDuration *prometheus.HistogramVec
	deliveryAuthFail *prometheus.CounterVec
	runsInFlight     prometheus.Gauge
	fanoutItems      *prometheus.HistogramVec
	fanoutItemErrors *prometheus.CounterVec

	runs          *prometheus.CounterVec
	runSteps      *prometheus.HistogramVec
	runDuration   *prometheus.HistogramVec
	tokens        *prometheus.CounterVec
	toolCalls     *prometheus.CounterVec
	denials       *prometheus.CounterVec
	applyFailures *prometheus.CounterVec

	llmRequests *prometheus.CounterVec
	llmDuration *prometheus.HistogramVec
	llmRetries  *prometheus.CounterVec

	syncs              *prometheus.CounterVec
	syncDuration       prometheus.Histogram
	lastSync           prometheus.Gauge
	rolesRegistered    prometheus.Gauge
	rolesSkipped       prometheus.Counter
	rolesMisconfigured prometheus.Gauge
	webhookActions     *prometheus.CounterVec
	webhooksOwned      *prometheus.GaugeVec

	configInfo *prometheus.GaugeVec
}

// New builds the collectors on a private registry, together with the standard
// Go runtime and process collectors.
func New() *Metrics {
	m := &Metrics{reg: prometheus.NewRegistry()}
	m.initDelivery()
	m.initRun()
	m.initControlPlane()

	m.reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.deliveries, m.deliveryDuration, m.deliveryAuthFail, m.runsInFlight,
		m.fanoutItems, m.fanoutItemErrors,
		m.runs, m.runSteps, m.runDuration, m.tokens, m.toolCalls, m.denials, m.applyFailures,
		m.llmRequests, m.llmDuration, m.llmRetries,
		m.syncs, m.syncDuration, m.lastSync, m.rolesRegistered, m.rolesSkipped,
		m.rolesMisconfigured, m.webhookActions, m.webhooksOwned, m.configInfo,
	)
	return m
}

func (m *Metrics) initDelivery() {
	m.deliveries = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fleet_deliveries_total",
		Help: "Webhook deliveries received, by role, kind (change|cron) and outcome",
	}, []string{roleLabel, "kind", statusLabel})
	m.deliveryDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "fleet_delivery_duration_seconds",
		Help:    "End-to-end delivery handling time in seconds, including the agent run",
		Buckets: []float64{0.1, 0.5, 1, 5, 15, 30, 60, 120, 300, 600},
	}, []string{roleLabel, "kind"})
	m.deliveryAuthFail = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fleet_delivery_auth_failures_total",
		Help: "Rejected deliveries by reason (unknown_key|bad_signature|bad_payload|read_body)",
	}, []string{"reason"})
	m.runsInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "fleet_runs_in_flight",
		Help: "Agent runs currently executing; deliveries are handled synchronously, so this is also the drain backlog",
	})
	m.fanoutItems = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "fleet_fanout_items",
		Help:    "Items one delivery fanned out into (for_each)",
		Buckets: []float64{0, 1, 2, 5, 10, 25, 50, 100},
	}, []string{roleLabel})
	m.fanoutItemErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fleet_fanout_item_errors_total",
		Help: "Fan-out items that failed; trip2g records a partial batch as success, so this is the only signal",
	}, []string{roleLabel})
}

func (m *Metrics) initRun() {
	m.runs = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fleet_runs_total",
		Help: "Agent runs by role and terminal status (completed|capped|max_steps|error)",
	}, []string{roleLabel, statusLabel})
	m.runSteps = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "fleet_run_steps",
		Help:    "Tool-loop iterations one run consumed; drift toward the step ceiling precedes hitting it",
		Buckets: []float64{1, 2, 3, 5, 8, 13, 21, 34},
	}, []string{roleLabel})
	m.runDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "fleet_run_duration_seconds",
		Help:    "Agent run duration in seconds",
		Buckets: []float64{0.5, 1, 5, 15, 30, 60, 120, 300, 600},
	}, []string{roleLabel})
	m.tokens = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fleet_llm_tokens_total",
		Help: "LLM tokens spent, by model, role and kind (prompt|completion). A codellm-backed fleet reports zero: it executes code, it does not infer",
	}, []string{modelLabel, roleLabel, "kind"})
	m.toolCalls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fleet_tool_calls_total",
		Help: "Tool calls by tool and outcome (ok|denied|invalid_args|error|apply_failed|not_permitted)",
	}, []string{"tool", "outcome"})
	m.denials = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fleet_denials_total",
		Help: "Scope denials by kind (read|write|not_permitted); a steady rate usually means misconfigured read/write_patterns",
	}, []string{"kind"})
	m.applyFailures = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fleet_apply_failures_total",
		Help: "Write/patch applies that failed, by role and tool. Under HardFailApply each one kills its run",
	}, []string{roleLabel, "tool"})

	m.llmRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fleet_llm_requests_total",
		Help: "Chat-completion requests by lane (llm|exec), model and outcome",
	}, []string{laneLabel, modelLabel, statusLabel})
	m.llmDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "fleet_llm_request_duration_seconds",
		Help:    "Chat-completion request duration in seconds, retries included",
		Buckets: []float64{0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120},
	}, []string{laneLabel, modelLabel})
	m.llmRetries = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fleet_llm_retries_total",
		Help: "Retried chat-completion attempts by lane and reason (429|5xx|network); the earliest upstream-instability signal",
	}, []string{laneLabel, "reason"})
}

func (m *Metrics) initControlPlane() {
	m.syncs = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fleet_syncs_total",
		Help: "Discovery/reconcile poll cycles by outcome (ok|partial|error)",
	}, []string{statusLabel})
	m.syncDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "fleet_sync_duration_seconds",
		Help:    "Duration of one discovery+reconcile cycle in seconds",
		Buckets: prometheus.DefBuckets,
	})
	m.lastSync = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "fleet_last_successful_sync_timestamp_seconds",
		Help: "Unix time of the last poll cycle that refreshed the registry (ok or partial); 0 until the first one. Staleness here means the fleet is serving stale roles",
	})
	m.rolesRegistered = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "fleet_roles_registered",
		Help: "Roles currently in the live registry",
	})
	m.rolesSkipped = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "fleet_roles_skipped_total",
		Help: "Role notes skipped during discovery (parse or validation failure)",
	})
	m.rolesMisconfigured = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "fleet_roles_write_scope_misconfigured",
		Help: "Roles declaring write tools with no write_patterns — every write they attempt is denied",
	})
	m.webhookActions = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fleet_webhook_actions_total",
		Help: "Webhook reconcile actions by action (create|update|delete) and outcome; a nonzero steady rate means two fleets are fighting over the same id",
	}, []string{"action", statusLabel})
	m.webhooksOwned = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fleet_webhooks_owned",
		Help: "Webhooks this fleet owns in trip2g, by kind",
	}, []string{"kind"})
	m.configInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fleet_config_info",
		Help: "Always 1; labels carry the identity and routing this process runs with",
	}, []string{"fleet_id", "default_model", "exec_enabled"})
}

// Registry exposes the private registry for tests and for callers that want to
// register their own collectors alongside the fleet's.
func (m *Metrics) Registry() *prometheus.Registry {
	if m == nil {
		return nil
	}
	return m.reg
}

// Handler builds the internal listener's mux: Prometheus scrape, pprof, and the
// k8s-convention probes, mirroring the monolith's internal listener. Nothing
// here is authenticated, which is why the port stays loopback-bound. ready may
// be nil (always ready).
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

// RecordDelivery counts one handled delivery and its wall time.
func (m *Metrics) RecordDelivery(role, kind, status string, seconds float64) {
	if m == nil {
		return
	}
	m.deliveries.WithLabelValues(role, kind, status).Inc()
	m.deliveryDuration.WithLabelValues(role, kind).Observe(seconds)
}

// RecordDeliveryAuthFailure counts a delivery rejected before it ever reached a
// run: unknown key, bad HMAC, unreadable body.
func (m *Metrics) RecordDeliveryAuthFailure(reason string) {
	if m == nil {
		return
	}
	m.deliveryAuthFail.WithLabelValues(reason).Inc()
}

// RunStarted marks a run as in flight and returns the function that clears it.
func (m *Metrics) RunStarted() func() {
	if m == nil {
		return func() {}
	}
	m.runsInFlight.Inc()
	return m.runsInFlight.Dec
}

// ObserveFanout records how many items a delivery expanded into and how many
// of them failed.
func (m *Metrics) ObserveFanout(role string, items, failed int) {
	if m == nil {
		return
	}
	m.fanoutItems.WithLabelValues(role).Observe(float64(items))
	if failed > 0 {
		m.fanoutItemErrors.WithLabelValues(role).Add(float64(failed))
	}
}

// RecordRun records one finished agent run.
func (m *Metrics) RecordRun(role, status string, steps int, seconds float64) {
	if m == nil {
		return
	}
	m.runs.WithLabelValues(role, status).Inc()
	m.runSteps.WithLabelValues(role).Observe(float64(steps))
	m.runDuration.WithLabelValues(role).Observe(seconds)
}

// RecordTokens attributes token spend to both the model that burned it and the
// role that asked for it, so cost-per-model and cost-per-role are two views of
// one series instead of two counters that can drift apart.
func (m *Metrics) RecordTokens(model, role, kind string, n int) {
	if m == nil || n <= 0 {
		return
	}
	m.tokens.WithLabelValues(model, role, kind).Add(float64(n))
}

// RecordToolCall counts one tool invocation.
func (m *Metrics) RecordToolCall(tool, outcome string) {
	if m == nil {
		return
	}
	m.toolCalls.WithLabelValues(tool, outcome).Inc()
}

// RecordDenial counts one scope denial.
func (m *Metrics) RecordDenial(kind string) {
	if m == nil {
		return
	}
	m.denials.WithLabelValues(kind).Inc()
}

// RecordApplyFailure counts a write/patch that could not be applied.
func (m *Metrics) RecordApplyFailure(role, tool string) {
	if m == nil {
		return
	}
	m.applyFailures.WithLabelValues(role, tool).Inc()
}

// RecordLLMRequest counts one chat-completion call (all retries included) and
// its duration.
func (m *Metrics) RecordLLMRequest(lane, model, status string, seconds float64) {
	if m == nil {
		return
	}
	m.llmRequests.WithLabelValues(lane, model, status).Inc()
	m.llmDuration.WithLabelValues(lane, model).Observe(seconds)
}

// RecordLLMRetry counts one retried attempt by reason.
func (m *Metrics) RecordLLMRetry(lane, reason string) {
	if m == nil {
		return
	}
	m.llmRetries.WithLabelValues(lane, reason).Inc()
}

// RecordSync records one poll cycle. Any cycle that refreshed the registry
// stamps the freshness gauge, StatusPartial included: a single unparseable role
// note must not freeze it forever. Roles that could not be used are counted
// separately by AddRolesSkipped.
func (m *Metrics) RecordSync(status string, seconds float64) {
	if m == nil {
		return
	}
	m.syncs.WithLabelValues(status).Inc()
	m.syncDuration.Observe(seconds)
	if status != StatusError {
		m.lastSync.Set(float64(time.Now().Unix()))
	}
}

// SetRoles publishes the live registry size and how many of those roles declare
// write tools they can never use.
func (m *Metrics) SetRoles(registered, misconfigured int) {
	if m == nil {
		return
	}
	m.rolesRegistered.Set(float64(registered))
	m.rolesMisconfigured.Set(float64(misconfigured))
}

// AddRolesSkipped counts role notes discovery could not use.
func (m *Metrics) AddRolesSkipped(n int) {
	if m == nil || n <= 0 {
		return
	}
	m.rolesSkipped.Add(float64(n))
}

// RecordWebhookAction counts one reconcile action against trip2g.
func (m *Metrics) RecordWebhookAction(action, status string) {
	if m == nil {
		return
	}
	m.webhookActions.WithLabelValues(action, status).Inc()
}

// SetWebhooksOwned publishes how many webhooks of a kind this fleet owns after
// the current reconcile.
func (m *Metrics) SetWebhooksOwned(kind string, n int) {
	if m == nil {
		return
	}
	m.webhooksOwned.WithLabelValues(kind).Set(float64(n))
}

// SetConfigInfo publishes this process's identity and routing, so an operator
// can tell two fleets apart from a scrape alone.
func (m *Metrics) SetConfigInfo(fleetID, defaultModel, execEnabled string) {
	if m == nil {
		return
	}
	m.configInfo.WithLabelValues(fleetID, defaultModel, execEnabled).Set(1)
}
