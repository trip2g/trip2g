package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"trip2g/internal/db"
)

//go:generate go tool github.com/matryer/moq -out test.go -pkg metrics . Env

// Env defines the environment interface for metrics updater.
type Env interface {
	CountAllNotePaths(ctx context.Context) (int64, error)
	CountVisibleNotePaths(ctx context.Context) (int64, error)
	CountNoteVersions(ctx context.Context) (int64, error)
	SumNoteAssetsSizes(ctx context.Context) (int64, error)
	CountNoteAssets(ctx context.Context) (int64, error)
	ListGoqiteAllQueueStats(ctx context.Context) ([]db.ListGoqiteAllQueueStatsRow, error)
}

// Updater periodically updates Prometheus metrics.
type Updater struct {
	env              Env
	allNotePaths     prometheus.Gauge
	visibleNotePaths prometheus.Gauge
	noteVersions     prometheus.Gauge
	noteAssetsSize   prometheus.Gauge
	noteAssetsCount  prometheus.Gauge
	queueDepth       *prometheus.GaugeVec
	interval         time.Duration
}

// NewUpdater creates a metrics updater with Prometheus gauges, registered on reg.
func NewUpdater(env Env, interval time.Duration, reg prometheus.Registerer) *Updater {
	allNotePaths := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "trip2g_note_paths_all",
		Help: "Total number of note paths (including hidden)",
	})
	reg.MustRegister(allNotePaths)

	visibleNotePaths := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "trip2g_note_paths_visible",
		Help: "Number of visible note paths (excluding hidden)",
	})
	reg.MustRegister(visibleNotePaths)

	noteVersions := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "trip2g_note_versions",
		Help: "Total number of note versions",
	})
	reg.MustRegister(noteVersions)

	noteAssetsSize := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "trip2g_note_assets_bytes",
		Help: "Total size of note assets in bytes",
	})
	reg.MustRegister(noteAssetsSize)

	noteAssetsCount := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "trip2g_note_assets",
		Help: "Number of note assets",
	})
	reg.MustRegister(noteAssetsCount)

	// state="pending" is never-received jobs; state="retrying" is jobs redelivered
	// at least once (received > 1). goqite's per-queue MaxReceive lives only in
	// process memory (appQueue), not the DB, so a job that finally exceeds it and
	// becomes undeliverable still shows up here as "retrying" rather than a
	// separate "dead" state — a sustained climb in either state is the signal
	// worth alerting on.
	queueDepth := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "trip2g_job_queue_depth",
		Help: "Number of goqite job-queue rows by queue and state (pending, retrying)",
	}, []string{"queue", "state"})
	reg.MustRegister(queueDepth)

	return &Updater{
		env:              env,
		allNotePaths:     allNotePaths,
		visibleNotePaths: visibleNotePaths,
		noteVersions:     noteVersions,
		noteAssetsSize:   noteAssetsSize,
		noteAssetsCount:  noteAssetsCount,
		queueDepth:       queueDepth,
		interval:         interval,
	}
}

// Run starts the periodic metrics update loop.
// It should be called in a goroutine: go updater.Run(ctx).
func (u *Updater) Run(ctx context.Context) error {
	ticker := time.NewTicker(u.interval)
	defer ticker.Stop()

	// Update metrics immediately on start.
	u.updateMetrics(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			u.updateMetrics(ctx)
		}
	}
}

func (u *Updater) updateMetrics(ctx context.Context) {
	var err error

	// All note paths.
	allCount, err := u.env.CountAllNotePaths(ctx)
	if err == nil {
		u.allNotePaths.Set(float64(allCount))
	}

	// Visible note paths.
	visibleCount, err := u.env.CountVisibleNotePaths(ctx)
	if err == nil {
		u.visibleNotePaths.Set(float64(visibleCount))
	}

	// Note versions.
	versionsCount, err := u.env.CountNoteVersions(ctx)
	if err == nil {
		u.noteVersions.Set(float64(versionsCount))
	}

	// Note assets size.
	assetsSize, err := u.env.SumNoteAssetsSizes(ctx)
	if err == nil {
		u.noteAssetsSize.Set(float64(assetsSize))
	}

	// Note assets count.
	assetsCount, err := u.env.CountNoteAssets(ctx)
	if err == nil {
		u.noteAssetsCount.Set(float64(assetsCount))
	}

	// Job queue depth, per queue. Reset first so a queue that drains to zero
	// (and drops out of the grouped query) doesn't leave a stale nonzero value.
	stats, err := u.env.ListGoqiteAllQueueStats(ctx)
	if err == nil {
		u.queueDepth.Reset()
		for _, s := range stats {
			u.queueDepth.WithLabelValues(s.Queue, "pending").Set(float64(s.PendingCount))
			u.queueDepth.WithLabelValues(s.Queue, "retrying").Set(float64(s.RetryCount))
		}
	}
}
