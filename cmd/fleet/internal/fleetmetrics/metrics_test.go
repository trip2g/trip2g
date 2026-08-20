package fleetmetrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

// scrape renders the registry the way Prometheus would read it.
func scrape(t *testing.T, m *Metrics) string {
	t.Helper()
	rec := httptest.NewRecorder()
	m.Handler(nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	return rec.Body.String()
}

// TestRecordSync_StampsFreshnessOnlyOnSuccess is the alerting contract: the
// freshness gauge must not advance on a failed cycle, or a fleet serving stale
// roles looks healthy forever.
func TestRecordSync_StampsFreshnessOnlyOnSuccess(t *testing.T) {
	m := New()
	m.RecordSync(StatusError, 0.2)
	require.InDelta(t, 0.0, testutil.ToFloat64(m.lastSync), 0.001)

	m.RecordSync(StatusOK, 0.3)
	require.Positive(t, testutil.ToFloat64(m.lastSync))
	require.InDelta(t, 1.0, testutil.ToFloat64(m.syncs.WithLabelValues(StatusOK)), 0.001)
	require.InDelta(t, 1.0, testutil.ToFloat64(m.syncs.WithLabelValues(StatusError)), 0.001)
}

// TestRecordTokens_AttributesModelAndRole asserts one series answers both
// "what does this model cost" and "what does this role cost".
func TestRecordTokens_AttributesModelAndRole(t *testing.T) {
	m := New()
	m.RecordTokens("gpt-4o-mini", "roles/triage.md", "prompt", 120)
	m.RecordTokens("gpt-4o-mini", "roles/triage.md", "completion", 40)
	m.RecordTokens("gpt-4o-mini", "roles/triage.md", "completion", 0) // ignored

	out := scrape(t, m)
	require.Contains(t, out, `fleet_llm_tokens_total{kind="prompt",model="gpt-4o-mini",role="roles/triage.md"} 120`)
	require.Contains(t, out, `fleet_llm_tokens_total{kind="completion",model="gpt-4o-mini",role="roles/triage.md"} 40`)
}

// TestRunStarted_BalancesInFlight asserts the returned release function clears
// the gauge, so a leaked run shows up rather than being hidden.
func TestRunStarted_BalancesInFlight(t *testing.T) {
	m := New()
	done := m.RunStarted()
	require.InDelta(t, 1.0, testutil.ToFloat64(m.runsInFlight), 0.001)
	done()
	require.InDelta(t, 0.0, testutil.ToFloat64(m.runsInFlight), 0.001)
}

// TestObserveFanout_CountsFailuresOnly asserts a clean batch adds nothing to
// the error counter while a partial one does.
func TestObserveFanout_CountsFailuresOnly(t *testing.T) {
	m := New()
	m.ObserveFanout("roles/a.md", 5, 0)
	require.Equal(t, 0, testutil.CollectAndCount(m.fanoutItemErrors))

	m.ObserveFanout("roles/a.md", 5, 2)
	require.InDelta(t, 2.0, testutil.ToFloat64(m.fanoutItemErrors.WithLabelValues("roles/a.md")), 0.001)
}

// TestHandler_ServesScrapeAndProbes asserts the internal listener exposes the
// scrape endpoint, pprof and the k8s-convention probes, readiness included.
func TestHandler_ServesScrapeAndProbes(t *testing.T) {
	m := New()
	m.SetConfigInfo("f1", "gpt-4o-mini", "true")

	ready := false
	h := m.Handler(func() bool { return ready })

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `fleet_config_info{default_model="gpt-4o-mini",exec_enabled="true",fleet_id="f1"} 1`)
	require.Contains(t, rec.Body.String(), "go_goroutines")

	for _, path := range []string{"/healthz", "/livez"} {
		rec = httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusOK, rec.Code, path)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)

	ready = true
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	require.Equal(t, http.StatusOK, rec.Code)
}

// TestNilMetrics_IsSafe asserts every call site can stay unconditional when
// metrics are disabled (--once, --metrics-addr="").
func TestNilMetrics_IsSafe(t *testing.T) {
	var m *Metrics
	require.NotPanics(t, func() {
		m.RecordDelivery("r", KindChange, StatusOK, 1)
		m.RecordDeliveryAuthFailure("bad_signature")
		m.RunStarted()()
		m.ObserveFanout("r", 1, 1)
		m.RecordRun("r", StatusOK, 2, 1)
		m.RecordTokens("m", "r", "prompt", 5)
		m.RecordToolCall("write_note", "ok")
		m.RecordDenial("write")
		m.RecordApplyFailure("r", "patch_note")
		m.RecordLLMRequest(LaneLLM, "m", StatusOK, 1)
		m.RecordLLMRetry(LaneExec, "429")
		m.RecordSync(StatusOK, 1)
		m.SetRoles(3, 1)
		m.AddRolesSkipped(2)
		m.RecordWebhookAction("create", StatusOK)
		m.SetWebhooksOwned(KindCron, 2)
		m.SetConfigInfo("f1", "m", "false")
		require.Nil(t, m.Registry())

		rec := httptest.NewRecorder()
		m.Handler(nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		require.Equal(t, http.StatusOK, rec.Code)
	})
}

// TestMetricNames_FollowPrometheusConventions guards the naming the scrape
// contract depends on: counters end in _total, durations are in seconds.
func TestMetricNames_FollowPrometheusConventions(t *testing.T) {
	m := New()
	problems, err := testutil.GatherAndLint(m.reg)
	require.NoError(t, err)
	var msgs []string
	for _, p := range problems {
		// The Go runtime collector's own names are client_golang's business.
		if strings.HasPrefix(p.Metric, "go_") || strings.HasPrefix(p.Metric, "process_") {
			continue
		}
		msgs = append(msgs, p.Metric+": "+p.Text)
	}
	require.Empty(t, msgs, strings.Join(msgs, "\n"))
}
