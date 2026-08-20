package fleet

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/cmd/fleet/internal/fleetmetrics"
)

// scrapeFleet renders the fleet's registry the way Prometheus would read it.
func scrapeFleet(t *testing.T, m *fleetmetrics.Metrics) string {
	t.Helper()
	rec := httptest.NewRecorder()
	m.Handler(nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	return rec.Body.String()
}

// newMeteredFleet builds the same one-role fleet the handler tests use, with a
// live metrics sink and a scoped-KB server behind it.
func newMeteredFleet(t *testing.T) (*Fleet, *fleetmetrics.Metrics) {
	t.Helper()
	srv, hc := newScopedKBServer(t, nil)
	cfg := Config{
		FleetID: "f1", FleetSecret: "seed", DefaultModel: "gpt-4o-mini",
		Trip2gBaseURL: srv.URL, TokenCeiling: 100000, StepCeiling: 25,
	}
	role := Role{
		NotePath: "roles/triage.md", Body: "Triage.", Mode: "change",
		ReadPatterns: []string{"boards/**"}, WritePatterns: []string{"boards/**"},
		MaxTokens: 4000, MaxSteps: 6, Concurrency: "skip", MaxDepth: 1,
	}
	m := fleetmetrics.New()
	f := NewFleet(cfg, hc, &stubLLM{}, nil)
	f.SetMetrics(m)
	f.SetRoles([]Role{role})
	return f, m
}

// TestMetrics_DeliveryIsRecorded asserts a served delivery lands in the
// delivery, fan-out, run and token series — the per-role spend view.
func TestMetrics_DeliveryIsRecorded(t *testing.T) {
	f, m := newMeteredFleet(t)

	rec := post(t, f, urlKey("roles/triage.md"), deliveryBody(t), true)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	out := scrapeFleet(t, m)
	require.Contains(t, out, `fleet_deliveries_total{kind="change",role="roles/triage.md",status="completed"} 1`)
	require.Contains(t, out, `fleet_runs_total{role="roles/triage.md",status="completed"} 1`)
	require.Contains(t, out, `fleet_tool_calls_total{outcome="ok",tool="patch_note"} 1`)
	require.Contains(t, out, `fleet_llm_tokens_total{kind="prompt",model="gpt-4o-mini",role="roles/triage.md"} 20`)
	require.Contains(t, out, `fleet_llm_tokens_total{kind="completion",model="gpt-4o-mini",role="roles/triage.md"} 10`)
	require.Contains(t, out, `fleet_runs_in_flight 0`)
}

// TestMetrics_RejectedDeliveriesAreCounted asserts requests turned away before
// a run — a wrong HMAC, an unknown key — are visible. Today they are silent.
func TestMetrics_RejectedDeliveriesAreCounted(t *testing.T) {
	f, m := newMeteredFleet(t)

	require.Equal(t, http.StatusUnauthorized, post(t, f, urlKey("roles/triage.md"), deliveryBody(t), false).Code)
	require.Equal(t, http.StatusNotFound, post(t, f, "nope", deliveryBody(t), false).Code)

	out := scrapeFleet(t, m)
	require.Contains(t, out, `fleet_delivery_auth_failures_total{reason="bad_signature"} 1`)
	require.Contains(t, out, `fleet_delivery_auth_failures_total{reason="unknown_key"} 1`)
}

// TestMetrics_RolesGaugeFlagsTheDenyAllTrap asserts a role declaring write
// tools with no write_patterns is published as misconfigured, not just logged
// once at startup.
func TestMetrics_RolesGaugeFlagsTheDenyAllTrap(t *testing.T) {
	m := fleetmetrics.New()
	f := NewFleet(Config{FleetID: "f1", FleetSecret: "seed"}, nil, &stubLLM{}, nil)
	f.SetMetrics(m)
	f.SetRoles([]Role{
		{NotePath: "roles/ok.md", Tools: []string{"write_note"}, WritePatterns: []string{"boards/**"}},
		{NotePath: "roles/trap.md", Tools: []string{"write_note"}},
	})

	out := scrapeFleet(t, m)
	require.Contains(t, out, "fleet_roles_registered 2")
	require.Contains(t, out, "fleet_roles_write_scope_misconfigured 1")
}
