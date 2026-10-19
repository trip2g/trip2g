package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestNoteQueryMetrics(t *testing.T) {
	var nilMetrics *NoteQueryMetrics
	nilMetrics.ObserveNoteQuery("q.md", "ok", 0.1) // must not panic

	reg := prometheus.NewRegistry()
	m := NewNoteQueryMetrics(reg)
	m.ObserveNoteQuery("queries/by_tag.md", "ok", 0.012)
	m.ObserveNoteQuery("queries/by_tag.md", "ok", 0.020)
	m.ObserveNoteQuery("unknown", "not_found", 0.001)

	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	require.Equal(t, "trip2g_note_query_duration_seconds", families[0].GetName())

	got := map[string]uint64{}
	for _, metric := range families[0].GetMetric() {
		labels := map[string]string{}
		for _, l := range metric.GetLabel() {
			labels[l.GetName()] = l.GetValue()
		}
		got[labels["query"]+" "+labels["result"]] = metric.GetHistogram().GetSampleCount()
	}
	require.Equal(t, map[string]uint64{"queries/by_tag.md ok": 2, "unknown not_found": 1}, got)
}
