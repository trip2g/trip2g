package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
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

func TestUpdater_QueueDepth_DeletesOnlyVanishedQueues(t *testing.T) {
	reg := prometheus.NewRegistry()

	calls := 0
	env := emptyEnv()
	env.ListGoqiteAllQueueStatsFunc = func(ctx context.Context) ([]db.ListGoqiteAllQueueStatsRow, error) {
		calls++
		if calls == 1 {
			return []db.ListGoqiteAllQueueStatsRow{
				{Queue: "global_jobs", PendingCount: 3},
				{Queue: "mail", PendingCount: 1, RetryCount: 1},
			}, nil
		}
		return []db.ListGoqiteAllQueueStatsRow{{Queue: "global_jobs", PendingCount: 4}}, nil
	}

	u := NewUpdater(env, 0, reg)
	u.updateMetrics(context.Background())
	u.updateMetrics(context.Background())

	pending := findMetric(t, reg, "trip2g_job_queue_depth", map[string]string{"queue": "global_jobs", "state": "pending"})
	require.InDelta(t, 4, pending.GetGauge().GetValue(), 1e-9)

	families, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if mf.GetName() != "trip2g_job_queue_depth" {
			continue
		}
		require.Len(t, mf.GetMetric(), 2, "only the drained queue's series are deleted")
		for _, m := range mf.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "queue" {
					require.Equal(t, "global_jobs", l.GetValue())
				}
			}
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
		PublicURLFunc: func() string { return "https://kb.example/" },
		LoadSiteConfigFunc: func(ctx context.Context) (model.SiteConfig, error) {
			return model.SiteConfig{}, nil
		},
		ListAllSubgraphsFunc: func(ctx context.Context) ([]db.Subgraph, error) {
			return nil, nil
		},
		LatestNoteViewsFunc: func() *model.NoteViews { return &model.NoteViews{} },
		LoggerFunc:          func() logger.Logger { return &logger.TestLogger{} },
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
		require.Len(t, mf.GetMetric(), 2, "neither a revoked key's nor an outbound key's scope is reported")
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

func TestUpdater_FederationSecrets_OutboundKeySharingAKID(t *testing.T) {
	reg := prometheus.NewRegistry()

	peerURL := "https://peer.example/_system/mcp"
	env := emptyEnv()
	env.ListFederationSecretsFunc = func(ctx context.Context) ([]db.ListFederationSecretsRow, error) {
		return []db.ListFederationSecretsRow{
			{ID: 1, Kid: "shared"},
			{ID: 2, Kid: "shared", KbUrl: &peerURL},
		}, nil
	}
	env.ListAllFederationSecretScopesFunc = func(ctx context.Context) ([]db.ListAllFederationSecretScopesRow, error) {
		return []db.ListAllFederationSecretScopesRow{
			{Kid: "shared", SubgraphID: 1, SubgraphName: "docs"},
		}, nil
	}

	u := NewUpdater(env, 0, reg)
	u.updateMetrics(context.Background())

	inbound := findMetric(t, reg, "trip2g_federation_secret_subgraphs", map[string]string{"direction": "inbound", "subgraph": "docs"})
	require.InDelta(t, 1, inbound.GetGauge().GetValue(), 1e-9)

	families, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if mf.GetName() == "trip2g_federation_secret_subgraphs" {
			require.Len(t, mf.GetMetric(), 1, "the outbound key must not pick up the inbound key's scope")
		}
	}
	outbound := findMetric(t, reg, "trip2g_federation_secrets", map[string]string{"direction": "outbound"})
	require.InDelta(t, 1, outbound.GetGauge().GetValue(), 1e-9)
}

func TestUpdater_FederationSecrets_DeletesOnlyVanishedSubgraphs(t *testing.T) {
	reg := prometheus.NewRegistry()

	calls := 0
	env := emptyEnv()
	env.ListFederationSecretsFunc = func(ctx context.Context) ([]db.ListFederationSecretsRow, error) {
		return []db.ListFederationSecretsRow{{ID: 1, Kid: "partner"}}, nil
	}
	env.ListAllFederationSecretScopesFunc = func(ctx context.Context) ([]db.ListAllFederationSecretScopesRow, error) {
		calls++
		if calls == 1 {
			return []db.ListAllFederationSecretScopesRow{
				{Kid: "partner", SubgraphID: 1, SubgraphName: "docs"},
				{Kid: "partner", SubgraphID: 2, SubgraphName: "sales"},
			}, nil
		}
		return []db.ListAllFederationSecretScopesRow{{Kid: "partner", SubgraphID: 1, SubgraphName: "docs"}}, nil
	}

	u := NewUpdater(env, 0, reg)
	u.updateMetrics(context.Background())
	u.updateMetrics(context.Background())

	docs := findMetric(t, reg, "trip2g_federation_secret_subgraphs", map[string]string{"direction": "inbound", "subgraph": "docs"})
	require.InDelta(t, 1, docs.GetGauge().GetValue(), 1e-9)

	families, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if mf.GetName() == "trip2g_federation_secret_subgraphs" {
			require.Len(t, mf.GetMetric(), 1, "only the subgraph removed from the key drops out")
		}
	}
}

func TestUpdater_FederationTopology(t *testing.T) {
	reg := prometheus.NewRegistry()

	peerURL := "https://user:pass@peer.example/_system/mcp?token=x#frag"
	revokedAt := time.Now()
	env := emptyEnv()
	env.LoadSiteConfigFunc = func(ctx context.Context) (model.SiteConfig, error) {
		return model.SiteConfig{KBID: "home"}, nil
	}
	env.ListFederationSecretsFunc = func(ctx context.Context) ([]db.ListFederationSecretsRow, error) {
		return []db.ListFederationSecretsRow{
			{ID: 1, Kid: "partner"},
			{ID: 2, Kid: "gone", RevokedAt: &revokedAt},
			{ID: 3, Kid: "peer", KbUrl: &peerURL},
		}, nil
	}
	env.ListAllFederationSecretScopesFunc = func(ctx context.Context) ([]db.ListAllFederationSecretScopesRow, error) {
		return []db.ListAllFederationSecretScopesRow{
			{Kid: "partner", SubgraphID: 1, SubgraphName: "docs"},
			{Kid: "partner", SubgraphID: 2, SubgraphName: "sales"},
			{Kid: "gone", SubgraphID: 3, SubgraphName: "archive"},
		}, nil
	}
	env.LatestNoteViewsFunc = func() *model.NoteViews {
		return &model.NoteViews{MCPFederationNotes: []*model.MCPFederationNote{
			{ID: "peer", URL: "https://peer.example/_system/mcp?token=x"},
		}}
	}

	u := NewUpdater(env, 0, reg)
	u.updateMetrics(context.Background())

	series := []struct {
		metric string
		labels map[string]string
	}{
		{"trip2g_federation_self_info", map[string]string{"kb_id": "home", "mcp_url": "https://kb.example/_system/mcp"}},
		{"trip2g_federation_key_info", map[string]string{"direction": "inbound", "kid": "partner", "kb_url": ""}},
		{"trip2g_federation_key_info", map[string]string{"direction": "outbound", "kid": "peer", "kb_url": "https://peer.example/_system/mcp"}},
		{"trip2g_federation_key_subgraph_info", map[string]string{"kid": "partner", "subgraph": "docs"}},
		{"trip2g_federation_key_subgraph_info", map[string]string{"kid": "partner", "subgraph": "sales"}},
		{"trip2g_federation_peer_info", map[string]string{"peer": "peer", "kb_url": "https://peer.example/_system/mcp"}},
	}
	for _, s := range series {
		m := findMetric(t, reg, s.metric, s.labels)
		require.InDelta(t, 1, m.GetGauge().GetValue(), 1e-9, "%s %v", s.metric, s.labels)
	}

	want := map[string]int{
		"trip2g_federation_self_info":         1,
		"trip2g_federation_key_info":          2,
		"trip2g_federation_key_subgraph_info": 2,
		"trip2g_federation_peer_info":         1,
	}
	families, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if n, ok := want[mf.GetName()]; ok {
			require.Len(t, mf.GetMetric(), n, "%s: a revoked key must not be reported", mf.GetName())
		}
	}
}

func TestUpdater_FederationTopology_DropsRemovedKeys(t *testing.T) {
	reg := prometheus.NewRegistry()

	calls := 0
	env := emptyEnv()
	env.ListFederationSecretsFunc = func(ctx context.Context) ([]db.ListFederationSecretsRow, error) {
		calls++
		if calls == 1 {
			return []db.ListFederationSecretsRow{{ID: 1, Kid: "partner"}}, nil
		}
		return nil, nil
	}
	env.ListAllFederationSecretScopesFunc = func(ctx context.Context) ([]db.ListAllFederationSecretScopesRow, error) {
		return []db.ListAllFederationSecretScopesRow{{Kid: "partner", SubgraphID: 1, SubgraphName: "docs"}}, nil
	}
	env.LatestNoteViewsFunc = func() *model.NoteViews {
		if calls == 1 {
			return &model.NoteViews{MCPFederationNotes: []*model.MCPFederationNote{
				{ID: "peer", URL: "https://peer.example/_system/mcp"},
			}}
		}
		return &model.NoteViews{}
	}

	u := NewUpdater(env, 0, reg)
	u.updateMetrics(context.Background())

	findMetric(t, reg, "trip2g_federation_key_info", map[string]string{"direction": "inbound", "kid": "partner", "kb_url": ""})
	findMetric(t, reg, "trip2g_federation_key_subgraph_info", map[string]string{"kid": "partner", "subgraph": "docs"})
	findMetric(t, reg, "trip2g_federation_peer_info", map[string]string{"peer": "peer", "kb_url": "https://peer.example/_system/mcp"})

	u.updateMetrics(context.Background())

	families, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range families {
		switch mf.GetName() {
		case "trip2g_federation_key_info", "trip2g_federation_key_subgraph_info", "trip2g_federation_peer_info":
			require.Empty(t, mf.GetMetric(), "%s: a deleted key or KB-note must drop out", mf.GetName())
		}
	}
}

func TestUpdater_FederationTopology_ScopeOnlyOnInboundKeys(t *testing.T) {
	reg := prometheus.NewRegistry()

	peerURL := "https://peer.example/_system/mcp"
	env := emptyEnv()
	env.ListFederationSecretsFunc = func(ctx context.Context) ([]db.ListFederationSecretsRow, error) {
		return []db.ListFederationSecretsRow{
			{ID: 1, Kid: "shared"},
			{ID: 2, Kid: "shared", KbUrl: &peerURL},
			{ID: 3, Kid: "solo", KbUrl: &peerURL},
		}, nil
	}
	env.ListAllFederationSecretScopesFunc = func(ctx context.Context) ([]db.ListAllFederationSecretScopesRow, error) {
		return []db.ListAllFederationSecretScopesRow{
			{Kid: "shared", SubgraphID: 1, SubgraphName: "docs"},
			{Kid: "solo", SubgraphID: 1, SubgraphName: "docs"},
		}, nil
	}

	u := NewUpdater(env, 0, reg)
	u.updateMetrics(context.Background())

	findMetric(t, reg, "trip2g_federation_key_info", map[string]string{"direction": "outbound", "kid": "shared", "kb_url": peerURL})
	findMetric(t, reg, "trip2g_federation_key_info", map[string]string{"direction": "outbound", "kid": "solo", "kb_url": peerURL})
	findMetric(t, reg, "trip2g_federation_key_subgraph_info", map[string]string{"kid": "shared", "subgraph": "docs"})
	families, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if mf.GetName() == "trip2g_federation_key_subgraph_info" {
			require.Len(t, mf.GetMetric(), 1, "the scope belongs to the inbound key alone")
		}
	}
}

func TestUpdater_FederationTopology_KeepsLastValuesOnError(t *testing.T) {
	reg := prometheus.NewRegistry()

	calls := 0
	env := emptyEnv()
	env.ListFederationSecretsFunc = func(ctx context.Context) ([]db.ListFederationSecretsRow, error) {
		return []db.ListFederationSecretsRow{{ID: 1, Kid: "partner"}}, nil
	}
	env.ListAllSubgraphsFunc = func(ctx context.Context) ([]db.Subgraph, error) {
		calls++
		if calls == 1 {
			return nil, nil
		}
		return nil, errors.New("database is locked")
	}

	u := NewUpdater(env, 0, reg)
	u.updateMetrics(context.Background())
	require.Empty(t, env.LoggerCalls(), "a successful update must not log")

	u.updateMetrics(context.Background())

	findMetric(t, reg, "trip2g_federation_key_info", map[string]string{"direction": "inbound", "kid": "partner", "kb_url": ""})
	require.Len(t, env.LoggerCalls(), 1, "a failed topology read must be logged")
}

func TestUpdater_FederationTopology_SelfFollowsPublicURL(t *testing.T) {
	reg := prometheus.NewRegistry()

	publicURL := "https://old.example"
	env := emptyEnv()
	env.PublicURLFunc = func() string { return publicURL }

	u := NewUpdater(env, 0, reg)
	u.updateMetrics(context.Background())
	publicURL = "https://new.example"
	u.updateMetrics(context.Background())

	findMetric(t, reg, "trip2g_federation_self_info", map[string]string{"kb_id": "new.example", "mcp_url": "https://new.example/_system/mcp"})
	families, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if mf.GetName() == "trip2g_federation_self_info" {
			require.Len(t, mf.GetMetric(), 1, "the previous self must be replaced, not kept")
		}
	}
}

func TestLabelURL(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"https://user:pass@peer.example/_system/mcp?token=x#frag", "https://peer.example/_system/mcp"},
		{"  https://peer.example:8443/_system/mcp  ", "https://peer.example:8443/_system/mcp"},
		{"http://[::1]:8080/mcp", "http://[::1]:8080/mcp"},
		{"https://peer.example/a%2Fb", "https://peer.example/a%2Fb"},
		{"peer.example/_system/mcp", ""},
		{"//peer.example/_system/mcp", ""},
		{"::not a url", ""},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, labelURL(tt.raw), tt.raw)
	}
}
