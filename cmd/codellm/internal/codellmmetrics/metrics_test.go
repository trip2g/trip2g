package codellmmetrics

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"trip2g/cmd/codellm/internal/coderun"
)

// TestInstrument_CountsStatusAndInFlight asserts the request lane records the
// real response status and leaves the in-flight gauge back at zero.
func TestInstrument_CountsStatusAndInFlight(t *testing.T) {
	m := New()
	var inFlightDuring float64
	h := m.Instrument("chat_completions", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		inFlightDuring = testutil.ToFloat64(m.inFlight)
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil))

	require.InDelta(t, 1.0, inFlightDuring, 0.001)
	require.InDelta(t, 0.0, testutil.ToFloat64(m.inFlight), 0.001)
	require.InDelta(t, 1.0, testutil.ToFloat64(m.requests.WithLabelValues("chat_completions", "422")), 0.001)
	require.Equal(t, 1, testutil.CollectAndCount(m.duration))
}

// TestRecordBlock_ExitCodeAndTruncation asserts a failed block lands in the
// outcome, exit-code and truncation series.
func TestRecordBlock_ExitCodeAndTruncation(t *testing.T) {
	m := New()
	m.RecordBlock(coderun.BlockStats{
		Program:         "python",
		Outcome:         coderun.BlockNonZeroExit,
		ExitCode:        3,
		DurationMs:      120,
		MaxRSSBytes:     4 << 20,
		StdoutBytes:     1024,
		StdoutTruncated: true,
		SandboxFallback: "unsupported OS",
	})

	require.InDelta(t, 1.0, testutil.ToFloat64(m.blocks.WithLabelValues("python", coderun.BlockNonZeroExit)), 0.001)
	require.InDelta(t, 1.0, testutil.ToFloat64(m.blockExitCodes.WithLabelValues("python", "3")), 0.001)
	require.InDelta(t, 1.0, testutil.ToFloat64(m.blockTruncated.WithLabelValues("python")), 0.001)
	require.InDelta(t, 1.0, testutil.ToFloat64(m.sandboxFallback.WithLabelValues("unsupported OS")), 0.001)
}

// TestRecordExecError_UsesCoderunKind asserts failures are counted by coderun's
// classification rather than by message text.
func TestRecordExecError_UsesCoderunKind(t *testing.T) {
	m := New()
	m.RecordExecError(nil)
	m.RecordExecError(&coderun.ExecError{Kind: coderun.KindSandboxRefused, Err: errors.New("refused")})
	m.RecordExecError(errors.New("plain"))

	require.InDelta(t, 1.0, testutil.ToFloat64(m.execError.WithLabelValues(coderun.KindSandboxRefused)), 0.001)
	require.InDelta(t, 1.0, testutil.ToFloat64(m.execError.WithLabelValues(coderun.KindUnclassified)), 0.001)
}

// TestHandler_ServesScrapeAndProbes asserts the internal listener exposes the
// scrape endpoint and the k8s-convention probes, and that readiness can fail.
func TestHandler_ServesScrapeAndProbes(t *testing.T) {
	m := New()
	m.SetConfigInfo("native", "false", "python,bash")
	m.RecordAuth(LaneCookie, Denied)

	ready := false
	h := m.Handler(func() bool { return ready })

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "codellm_config_info")
	require.Contains(t, rec.Body.String(), `codellm_auth_total{lane="cookie",result="denied"} 1`)
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
// metrics are disabled (--metrics-addr="").
func TestNilMetrics_IsSafe(t *testing.T) {
	var m *Metrics
	require.NotPanics(t, func() {
		m.RecordAuth(LaneAPIKey, Allowed)
		m.RecordBlock(coderun.BlockStats{})
		m.RecordExecError(errors.New("boom"))
		m.RecordReplay()
		m.ObserveRequestBlocks(2)
		m.RecordChange("write")
		m.SetConfigInfo("native", "false", "python")
		require.Nil(t, m.Registry())

		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
		rec := httptest.NewRecorder()
		m.Instrument("chat_completions", next).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil))
		require.Equal(t, http.StatusTeapot, rec.Code)

		rec = httptest.NewRecorder()
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
