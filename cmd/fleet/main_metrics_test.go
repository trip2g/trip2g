package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	fleetappconfig "trip2g/cmd/fleet/appconfig"
	"trip2g/cmd/fleet/internal/fleet"
	"trip2g/cmd/fleet/internal/fleetmetrics"
	"trip2g/internal/zerologger"
)

// TestSyncStatus_Precedence pins how a poll cycle is graded: a reconcile
// failure means the cycle did not land at all, while dropped role notes leave
// the registry refreshed but incomplete.
func TestSyncStatus_Precedence(t *testing.T) {
	tests := []struct {
		name         string
		reconcileErr error
		skipped      int
		want         string
	}{
		{name: "clean cycle", want: fleetmetrics.StatusOK},
		{name: "dropped role notes", skipped: 2, want: fleetmetrics.StatusPartial},
		{name: "reconcile failed", reconcileErr: errors.New("trip2g down"), want: fleetmetrics.StatusError},
		{
			name:         "reconcile failure outranks dropped notes",
			reconcileErr: errors.New("trip2g down"),
			skipped:      2,
			want:         fleetmetrics.StatusError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, syncStatus(tt.reconcileErr, tt.skipped))
		})
	}
}

// TestSyncStatus_PartialStillRefreshesFreshness is the alerting contract: one
// permanently unparseable role note must not freeze the staleness gauge and
// turn the fleet's top alert into a standing complaint about a typo.
func TestSyncStatus_PartialStillRefreshesFreshness(t *testing.T) {
	m := fleetmetrics.New()
	m.RecordSync(syncStatus(nil, 1), 0.1)

	out := scrapeMetrics(t, m)
	require.Contains(t, out, `fleet_syncs_total{status="partial"} 1`)
	require.NotContains(t, out, "fleet_last_successful_sync_timestamp_seconds 0\n")
}

// TestSyncStatus_ErrorLeavesFreshnessUnset asserts a cycle that never reached
// trip2g does not advance the staleness gauge.
func TestSyncStatus_ErrorLeavesFreshnessUnset(t *testing.T) {
	m := fleetmetrics.New()
	m.RecordSync(syncStatus(errors.New("down"), 0), 0.1)

	out := scrapeMetrics(t, m)
	require.Contains(t, out, `fleet_syncs_total{status="error"} 1`)
	require.Contains(t, out, "fleet_last_successful_sync_timestamp_seconds 0\n")
}

// TestStartMetricsServer_DisabledByEmptyAddr asserts an empty --metrics-addr
// starts nothing and yields the nil sink every call site tolerates.
func TestStartMetricsServer_DisabledByEmptyAddr(t *testing.T) {
	cli := cliFlags{appCfg: fleetappconfig.DefaultConfig()}
	cli.appCfg.MetricsAddr = ""

	require.Nil(t, startMetricsServer(zerologger.New("error", false), cli, nil))
}

// TestStartMetricsServer_PublishesIdentity asserts an enabled listener comes up
// with this process's identity already published, so two fleets are
// distinguishable from a scrape alone.
func TestStartMetricsServer_PublishesIdentity(t *testing.T) {
	cli := cliFlags{appCfg: fleetappconfig.DefaultConfig()}
	cli.appCfg.MetricsAddr = freeAddr(t)
	cli.cfg = fleet.Config{FleetID: "f-test", DefaultModel: "gpt-4o-mini", ExecBaseURL: "http://codellm"}

	m := startMetricsServer(zerologger.New("error", false), cli, nil)
	require.NotNil(t, m)
	require.Contains(t, scrapeMetrics(t, m),
		`fleet_config_info{default_model="gpt-4o-mini",exec_enabled="true",fleet_id="f-test"} 1`)
}

// TestIsLoopbackAddr covers the check behind the non-loopback warning: the
// internal listener serves unauthenticated pprof, so an off-box bind must be
// recognised rather than assumed away.
func TestIsLoopbackAddr(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{addr: "127.0.0.1:18090", want: true},
		{addr: "localhost:18090", want: true},
		{addr: "[::1]:18090", want: true},
		{addr: "0.0.0.0:18090", want: false},
		{addr: ":18090", want: false},
		{addr: "10.0.0.5:18090", want: false},
		{addr: "garbage", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			require.Equal(t, tt.want, isLoopbackAddr(tt.addr))
		})
	}
}

// scrapeMetrics renders a registry the way Prometheus would read it.
func scrapeMetrics(t *testing.T, m *fleetmetrics.Metrics) string {
	t.Helper()
	rec := httptest.NewRecorder()
	m.Handler(nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	return rec.Body.String()
}

// freeAddr returns a loopback address that was free a moment ago. The listener
// under test binds it itself, so it is released first.
func freeAddr(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.NotFoundHandler())
	addr := srv.Listener.Addr().String()
	srv.Close()
	return addr
}
