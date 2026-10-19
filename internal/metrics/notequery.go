package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// NoteQueryMetrics times named note queries (noteQuery) per query note. The
// query label is the query note's path, so cardinality is bounded by the
// vault's query notes; unknown names share one label. Nil-safe.
type NoteQueryMetrics struct {
	duration *prometheus.HistogramVec
}

// NewNoteQueryMetrics creates and registers the note query metrics on reg.
func NewNoteQueryMetrics(reg prometheus.Registerer) *NoteQueryMetrics {
	m := &NoteQueryMetrics{
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "trip2g_note_query_duration_seconds",
			Help: "Named note query (noteQuery) duration in seconds, by query note and result",
			// 1ms .. ~16s: a query is normally milliseconds; the tail shows runaways.
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
		}, []string{"query", "result"}),
	}
	reg.MustRegister(m.duration)
	return m
}

// ObserveNoteQuery records one call. result is ok, error, not_found or failed.
func (m *NoteQueryMetrics) ObserveNoteQuery(query, result string, seconds float64) {
	if m == nil {
		return
	}
	m.duration.WithLabelValues(query, result).Observe(seconds)
}
