package metrics

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
)

func TestUpdater_QueueDepth(t *testing.T) {
	reg := prometheus.NewRegistry()

	env := &EnvMock{
		CountAllNotePathsFunc:     func(ctx context.Context) (int64, error) { return 0, nil },
		CountVisibleNotePathsFunc: func(ctx context.Context) (int64, error) { return 0, nil },
		CountNoteVersionsFunc:     func(ctx context.Context) (int64, error) { return 0, nil },
		SumNoteAssetsSizesFunc:    func(ctx context.Context) (int64, error) { return 0, nil },
		CountNoteAssetsFunc:       func(ctx context.Context) (int64, error) { return 0, nil },
		ListGoqiteAllQueueStatsFunc: func(ctx context.Context) ([]db.ListGoqiteAllQueueStatsRow, error) {
			return []db.ListGoqiteAllQueueStatsRow{
				{Queue: "global_jobs", TotalJobs: 12, PendingCount: 5, RetryCount: 2},
			}, nil
		},
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
	env := &EnvMock{
		CountAllNotePathsFunc:     func(ctx context.Context) (int64, error) { return 0, nil },
		CountVisibleNotePathsFunc: func(ctx context.Context) (int64, error) { return 0, nil },
		CountNoteVersionsFunc:     func(ctx context.Context) (int64, error) { return 0, nil },
		SumNoteAssetsSizesFunc:    func(ctx context.Context) (int64, error) { return 0, nil },
		CountNoteAssetsFunc:       func(ctx context.Context) (int64, error) { return 0, nil },
		ListGoqiteAllQueueStatsFunc: func(ctx context.Context) ([]db.ListGoqiteAllQueueStatsRow, error) {
			calls++
			if calls == 1 {
				return []db.ListGoqiteAllQueueStatsRow{{Queue: "global_jobs", TotalJobs: 3, PendingCount: 3}}, nil
			}
			// Queue fully drained: absent from the grouped query results.
			return nil, nil
		},
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
