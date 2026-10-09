package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
)

func TestUpdater_QueueDepth(t *testing.T) {
	reg := prometheus.NewRegistry()

	env := emptyEnv()
	env.ListGoqiteAllQueueStatsFunc = func(ctx context.Context) ([]db.ListGoqiteAllQueueStatsRow, error) {
		return []db.ListGoqiteAllQueueStatsRow{
			{Queue: "global_jobs", TotalJobs: 12, PendingCount: 5, RetryCount: 2},
		}, nil
	}

	u := NewUpdater(env, 0, reg)
	u.updateMetrics(context.Background())

	pending := findMetric(t, reg, "trip2g_job_queue_depth", map[string]string{"queue": "global_jobs", "state": "pending"})
	require.InDelta(t, 5, pending.GetGauge().GetValue(), 1e-9)

	retrying := findMetric(t, reg, "trip2g_job_queue_depth", map[string]string{"queue": "global_jobs", "state": "retrying"})
	require.InDelta(t, 2, retrying.GetGauge().GetValue(), 1e-9)
}

func TestUpdater_QueueDepth_ResetsDrainedQueues(t *testing.T) {
	reg := prometheus.NewRegistry()

	calls := 0
	env := emptyEnv()
	env.ListGoqiteAllQueueStatsFunc = func(ctx context.Context) ([]db.ListGoqiteAllQueueStatsRow, error) {
		calls++
		if calls == 1 {
			return []db.ListGoqiteAllQueueStatsRow{{Queue: "global_jobs", TotalJobs: 3, PendingCount: 3}}, nil
		}
		// Queue fully drained: absent from the grouped query results.
		return nil, nil
	}

	u := NewUpdater(env, 0, reg)
	u.updateMetrics(context.Background())
	u.updateMetrics(context.Background())

	families, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if mf.GetName() == "trip2g_job_queue_depth" {
			require.Empty(t, mf.GetMetric(), "stale queue depth series must be reset after the queue drains")
		}
	}
}

func emptyEnv() *EnvMock {
	return &EnvMock{
		CountAllNotePathsFunc:     func(ctx context.Context) (int64, error) { return 0, nil },
		CountVisibleNotePathsFunc: func(ctx context.Context) (int64, error) { return 0, nil },
		CountNoteVersionsFunc:     func(ctx context.Context) (int64, error) { return 0, nil },
		SumNoteAssetsSizesFunc:    func(ctx context.Context) (int64, error) { return 0, nil },
		CountNoteAssetsFunc:       func(ctx context.Context) (int64, error) { return 0, nil },
		ListGoqiteAllQueueStatsFunc: func(ctx context.Context) ([]db.ListGoqiteAllQueueStatsRow, error) {
			return nil, nil
		},
		ListFederationSecretsFunc: func(ctx context.Context) ([]db.ListFederationSecretsRow, error) {
			return nil, nil
		},
		ListAllFederationSecretScopesFunc: func(ctx context.Context) ([]db.ListAllFederationSecretScopesRow, error) {
			return nil, nil
		},
	}
}

func TestUpdater_FederationSecrets(t *testing.T) {
	reg := prometheus.NewRegistry()

	peerURL := "https://peer.example/_system/mcp"
	revokedAt := time.Now()
	env := emptyEnv()
	env.ListFederationSecretsFunc = func(ctx context.Context) ([]db.ListFederationSecretsRow, error) {
		return []db.ListFederationSecretsRow{
			{ID: 1, Kid: "partner"},
			{ID: 2, Kid: "reader"},
			{ID: 3, Kid: "gone", RevokedAt: &revokedAt},
			{ID: 4, Kid: "peer", KbUrl: &peerURL},
		}, nil
	}
	env.ListAllFederationSecretScopesFunc = func(ctx context.Context) ([]db.ListAllFederationSecretScopesRow, error) {
		return []db.ListAllFederationSecretScopesRow{
			{Kid: "partner", SubgraphID: 1, SubgraphName: "docs"},
			{Kid: "partner", SubgraphID: 2, SubgraphName: "sales"},
			{Kid: "reader", SubgraphID: 1, SubgraphName: "docs"},
			{Kid: "gone", SubgraphID: 3, SubgraphName: "archive"},
			{Kid: "peer", SubgraphID: 1, SubgraphName: "docs"},
		}, nil
	}

	u := NewUpdater(env, 0, reg)
	u.updateMetrics(context.Background())

	gauges := []struct {
		metric string
		labels map[string]string
		want   float64
	}{
		{"trip2g_federation_secrets", map[string]string{"direction": "inbound"}, 2},
		{"trip2g_federation_secrets", map[string]string{"direction": "outbound"}, 1},
		{"trip2g_federation_secret_subgraphs", map[string]string{"direction": "inbound", "subgraph": "docs"}, 2},
		{"trip2g_federation_secret_subgraphs", map[string]string{"direction": "inbound", "subgraph": "sales"}, 1},
		{"trip2g_federation_secret_subgraphs", map[string]string{"direction": "outbound", "subgraph": "docs"}, 1},
	}
	for _, g := range gauges {
		m := findMetric(t, reg, g.metric, g.labels)
		require.InDelta(t, g.want, m.GetGauge().GetValue(), 1e-9, "%s %v", g.metric, g.labels)
	}

	families, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if mf.GetName() != "trip2g_federation_secret_subgraphs" {
			continue
		}
		require.Len(t, mf.GetMetric(), 3, "a subgraph only a revoked key was scoped to must not be reported")
	}
}

func TestUpdater_FederationSecrets_DropsUnscopedSubgraphs(t *testing.T) {
	reg := prometheus.NewRegistry()

	calls := 0
	env := emptyEnv()
	env.ListFederationSecretsFunc = func(ctx context.Context) ([]db.ListFederationSecretsRow, error) {
		return []db.ListFederationSecretsRow{{ID: 1, Kid: "partner"}}, nil
	}
	env.ListAllFederationSecretScopesFunc = func(ctx context.Context) ([]db.ListAllFederationSecretScopesRow, error) {
		calls++
		if calls == 1 {
			return []db.ListAllFederationSecretScopesRow{{Kid: "partner", SubgraphID: 1, SubgraphName: "docs"}}, nil
		}
		return nil, nil
	}

	u := NewUpdater(env, 0, reg)
	u.updateMetrics(context.Background())
	u.updateMetrics(context.Background())

	families, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if mf.GetName() == "trip2g_federation_secret_subgraphs" {
			require.Empty(t, mf.GetMetric(), "a subgraph removed from its last key must drop out")
		}
	}
	m := findMetric(t, reg, "trip2g_federation_secrets", map[string]string{"direction": "inbound"})
	require.InDelta(t, 1, m.GetGauge().GetValue(), 1e-9)
}
