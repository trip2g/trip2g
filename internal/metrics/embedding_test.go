package metrics

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestNewEmbeddingMetrics_Records(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	u := func(v uint64) *uint64 { return &v }

	cases := []struct {
		name   string
		record func(m *EmbeddingMetrics)
		metric string
		labels map[string]string

		wantCounter   *float64
		wantHistCount *uint64
		wantHistSum   *float64
	}{
		{
			name:        "requests_total counts ok by kind",
			record:      func(m *EmbeddingMetrics) { m.RecordEmbeddingRequest("whole_note", "ok", "", 0.1) },
			metric:      "trip2g_embedding_requests_total",
			labels:      map[string]string{"result": "ok", "kind": "whole_note", "reason": ""},
			wantCounter: f(1),
		},
		{
			name: "requests_total counts errors with bounded reason",
			record: func(m *EmbeddingMetrics) {
				m.RecordEmbeddingRequest("chunk", "error", EmbeddingErrorBatchTooLarge, 0.2)
			},
			metric:      "trip2g_embedding_requests_total",
			labels:      map[string]string{"result": "error", "kind": "chunk", "reason": EmbeddingErrorBatchTooLarge},
			wantCounter: f(1),
		},
		{
			name:          "request_duration observes per kind",
			record:        func(m *EmbeddingMetrics) { m.RecordEmbeddingRequest("query", "ok", "", 0.5) },
			metric:        "trip2g_embedding_request_duration_seconds",
			labels:        map[string]string{"kind": "query"},
			wantHistCount: u(1),
			wantHistSum:   f(0.5),
		},
		{
			name:        "jobs_total counts by result",
			record:      func(m *EmbeddingMetrics) { m.RecordJob("succeeded") },
			metric:      "trip2g_embedding_jobs_total",
			labels:      map[string]string{"result": "succeeded"},
			wantCounter: f(1),
		},
		{
			name:        "regen_enqueued_total accumulates",
			record:      func(m *EmbeddingMetrics) { m.RecordRegen(3, 5, 1) },
			metric:      "trip2g_embedding_regen_enqueued_total",
			labels:      map[string]string{},
			wantCounter: f(3),
		},
		{
			name:        "regen_up_to_date_total accumulates",
			record:      func(m *EmbeddingMetrics) { m.RecordRegen(3, 5, 1) },
			metric:      "trip2g_embedding_regen_up_to_date_total",
			labels:      map[string]string{},
			wantCounter: f(5),
		},
		{
			name:        "regen_errors_total accumulates",
			record:      func(m *EmbeddingMetrics) { m.RecordRegen(3, 5, 1) },
			metric:      "trip2g_embedding_regen_errors_total",
			labels:      map[string]string{},
			wantCounter: f(1),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg := prometheus.NewRegistry()
			m := NewEmbeddingMetrics(reg)
			tc.record(m)

			metric := findMetric(t, reg, tc.metric, tc.labels)
			if tc.wantCounter != nil {
				require.InDelta(t, *tc.wantCounter, metric.GetCounter().GetValue(), 1e-9)
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

func TestEmbeddingMetrics_NilSafe(t *testing.T) {
	var m *EmbeddingMetrics
	require.NotPanics(t, func() {
		m.RecordEmbeddingRequest("whole_note", "ok", "", 0.1)
		m.RecordJob("succeeded")
		m.RecordRegen(1, 2, 3)
	})
}

func TestEmbeddingMetricsContext_RoundTrips(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewEmbeddingMetrics(reg)

	ctx := ContextWithEmbeddingMetrics(context.Background(), m)
	require.Same(t, m, EmbeddingMetricsFromContext(ctx))
}

func TestEmbeddingMetricsFromContext_MissingReturnsNil(t *testing.T) {
	require.Nil(t, EmbeddingMetricsFromContext(context.Background()))
}
