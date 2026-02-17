package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

//go:generate moq -out test.go -pkg metrics . Env

// Env defines the environment interface for metrics updater.
type Env interface {
	CountAllNotePaths(ctx context.Context) (int64, error)
	CountVisibleNotePaths(ctx context.Context) (int64, error)
	CountNoteVersions(ctx context.Context) (int64, error)
	SumNoteAssetsSizes(ctx context.Context) (int64, error)
	CountNoteAssets(ctx context.Context) (int64, error)
}

// Updater periodically updates Prometheus metrics.
type Updater struct {
	env              Env
	allNotePaths     prometheus.Gauge
	visibleNotePaths prometheus.Gauge
	noteVersions     prometheus.Gauge
	noteAssetsSize   prometheus.Gauge
	noteAssetsCount  prometheus.Gauge
	interval         time.Duration
}

// NewUpdater creates a metrics updater with Prometheus gauges.
func NewUpdater(env Env, interval time.Duration) *Updater {
	allNotePaths := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "trip2g_note_paths_all",
		Help: "Total number of note paths (including hidden)",
	})
	prometheus.MustRegister(allNotePaths)

	visibleNotePaths := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "trip2g_note_paths_visible",
		Help: "Number of visible note paths (excluding hidden)",
	})
	prometheus.MustRegister(visibleNotePaths)

	noteVersions := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "trip2g_note_versions",
		Help: "Total number of note versions",
	})
	prometheus.MustRegister(noteVersions)

	noteAssetsSize := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "trip2g_note_assets_bytes",
		Help: "Total size of note assets in bytes",
	})
	prometheus.MustRegister(noteAssetsSize)

	noteAssetsCount := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "trip2g_note_assets",
		Help: "Number of note assets",
	})
	prometheus.MustRegister(noteAssetsCount)

	return &Updater{
		env:              env,
		allNotePaths:     allNotePaths,
		visibleNotePaths: visibleNotePaths,
		noteVersions:     noteVersions,
		noteAssetsSize:   noteAssetsSize,
		noteAssetsCount:  noteAssetsCount,
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
}
