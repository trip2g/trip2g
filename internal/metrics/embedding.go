package metrics

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

// EmbeddingMetrics collects Prometheus metrics for the embedding pipeline:
// embedding API requests (whole-note, chunk, query), embed job outcomes, and
// the regeneration cron sweep. All record methods are nil-safe so callers
// without wired metrics (tests, partial app setups) can call them
// unconditionally.
type EmbeddingMetrics struct {
	requests      *prometheus.CounterVec
	duration      *prometheus.HistogramVec
	jobs          *prometheus.CounterVec
	regenEnqueued prometheus.Counter
	regenUpToDate prometheus.Counter
	regenErrors   prometheus.Counter
}

// NewEmbeddingMetrics creates and registers all embedding Prometheus metrics on reg.
func NewEmbeddingMetrics(reg prometheus.Registerer) *EmbeddingMetrics {
	m := &EmbeddingMetrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "trip2g_embedding_requests_total",
			Help: "Total number of embedding API requests by result, kind and error reason",
		}, []string{"result", "kind", "reason"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "trip2g_embedding_request_duration_seconds",
			Help:    "Embedding API request duration in seconds, per kind",
			Buckets: prometheus.DefBuckets,
		}, []string{"kind"}),
		jobs: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "trip2g_embedding_jobs_total",
			Help: "Total number of generate-note-version-embedding job executions by result",
		}, []string{"result"}),
		regenEnqueued: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "trip2g_embedding_regen_enqueued_total",
			Help: "Total number of notes enqueued for re-embedding by the regeneration cron",
		}),
		regenUpToDate: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "trip2g_embedding_regen_up_to_date_total",
			Help: "Total number of notes found already up to date by the regeneration cron",
		}),
		regenErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "trip2g_embedding_regen_errors_total",
			Help: "Total number of enqueue errors reported by the regeneration cron",
		}),
	}

	reg.MustRegister(m.requests, m.duration, m.jobs, m.regenEnqueued, m.regenUpToDate, m.regenErrors)
	return m
}

// RecordEmbeddingRequest records one embedding API call: requests_total
// (labeled by kind, result and — on error — a bounded failure reason) and the
// duration histogram (labeled by kind). reason is ignored (empty) on success.
func (m *EmbeddingMetrics) RecordEmbeddingRequest(kind, result, reason string, seconds float64) {
	if m == nil {
		return
	}
	m.requests.WithLabelValues(result, kind, reason).Inc()
	m.duration.WithLabelValues(kind).Observe(seconds)
}

// RecordJob counts one generate-note-version-embedding job execution by result
// ("succeeded" or "failed").
func (m *EmbeddingMetrics) RecordJob(result string) {
	if m == nil {
		return
	}
	m.jobs.WithLabelValues(result).Inc()
}

// RecordRegen adds one regeneration cron sweep's counts to the running totals.
func (m *EmbeddingMetrics) RecordRegen(enqueued, upToDate, errs int) {
	if m == nil {
		return
	}
	m.regenEnqueued.Add(float64(enqueued))
	m.regenUpToDate.Add(float64(upToDate))
	m.regenErrors.Add(float64(errs))
}

// Bounded failure reasons for the requests_total "reason" label — the
// embedding client classifies API errors into this fixed set so the label
// never grows unbounded.
const (
	EmbeddingErrorBatchTooLarge = "batch_too_large"
	EmbeddingErrorOverloaded    = "overloaded"
	EmbeddingErrorOther         = "other"
)

type embeddingMetricsContextKey struct{}

// ContextWithEmbeddingMetrics stores m in ctx so background job/cron code
// (which has no HTTP request to carry metrics through) can record embedding
// pipeline observations. Wired once, at the root context queues and the cron
// scheduler are built from.
func ContextWithEmbeddingMetrics(ctx context.Context, m *EmbeddingMetrics) context.Context {
	return context.WithValue(ctx, embeddingMetricsContextKey{}, m)
}

// EmbeddingMetricsFromContext returns the embedding metrics stored in ctx, or
// nil (all record methods are nil-safe).
func EmbeddingMetricsFromContext(ctx context.Context) *EmbeddingMetrics {
	m, _ := ctx.Value(embeddingMetricsContextKey{}).(*EmbeddingMetrics)
	return m
}
