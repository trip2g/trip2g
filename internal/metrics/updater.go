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
	ListFederationSecrets(ctx context.Context) ([]db.ListFederationSecretsRow, error)
	ListAllFederationSecretScopes(ctx context.Context) ([]db.ListAllFederationSecretScopesRow, error)
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
	fedSecrets       *prometheus.GaugeVec
	fedSubgraphs     *prometheus.GaugeVec
	interval         time.Duration

	lastQueueDepth   labelPairs
	lastFedSubgraphs labelPairs
}

// labelPairs holds the values of a two-label gauge vector by label pair.
type labelPairs map[[2]string]float64

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

	fedSecrets := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "trip2g_federation_secrets",
		Help: "Number of live (not revoked) federation keys by direction (inbound, outbound)",
	}, []string{"direction"})
	reg.MustRegister(fedSecrets)

	fedSubgraphs := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "trip2g_federation_secret_subgraphs",
		Help: "Number of live inbound federation keys scoped to each subgraph (scope only applies to inbound keys)",
	}, []string{"direction", "subgraph"})
	reg.MustRegister(fedSubgraphs)

	return &Updater{
		env:              env,
		allNotePaths:     allNotePaths,
		visibleNotePaths: visibleNotePaths,
		noteVersions:     noteVersions,
		noteAssetsSize:   noteAssetsSize,
		noteAssetsCount:  noteAssetsCount,
		queueDepth:       queueDepth,
		fedSecrets:       fedSecrets,
		fedSubgraphs:     fedSubgraphs,
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

	// Job queue depth, per queue. A queue that drains to zero drops out of the
	// grouped query, so its series is deleted rather than left stale.
	stats, err := u.env.ListGoqiteAllQueueStats(ctx)
	if err == nil {
		depth := labelPairs{}
		for _, s := range stats {
			depth[[2]string{s.Queue, "pending"}] = float64(s.PendingCount)
			depth[[2]string{s.Queue, "retrying"}] = float64(s.RetryCount)
		}
		u.lastQueueDepth = replaceGauges(u.queueDepth, u.lastQueueDepth, depth)
	}

	u.updateFederationMetrics(ctx)
}

// Federation key directions: an inbound key lets a peer read this base, an
// outbound key lets this base read a peer (it carries the peer's kb_url).
const (
	directionInbound  = "inbound"
	directionOutbound = "outbound"
)

func (u *Updater) updateFederationMetrics(ctx context.Context) {
	secrets, err := u.env.ListFederationSecrets(ctx)
	if err != nil {
		return
	}
	scopes, err := u.env.ListAllFederationSecretScopes(ctx)
	if err != nil {
		return
	}

	subgraphsByKID := map[string][]string{}
	for _, scope := range scopes {
		subgraphsByKID[scope.Kid] = append(subgraphsByKID[scope.Kid], scope.SubgraphName)
	}

	keys := map[string]int{directionInbound: 0, directionOutbound: 0}
	perSubgraph := labelPairs{}
	for _, secret := range secrets {
		if secret.RevokedAt != nil {
			continue
		}
		direction := directionInbound
		if secret.KbUrl != nil {
			direction = directionOutbound
		}
		keys[direction]++
		// Scope rows are keyed by kid alone and only gate inbound keys, so an
		// outbound key sharing a kid would pick up an inbound key's scope.
		if direction != directionInbound {
			continue
		}
		for _, subgraph := range subgraphsByKID[secret.Kid] {
			perSubgraph[[2]string{direction, subgraph}]++
		}
	}

	for direction, n := range keys {
		u.fedSecrets.WithLabelValues(direction).Set(float64(n))
	}
	u.lastFedSubgraphs = replaceGauges(u.fedSubgraphs, u.lastFedSubgraphs, perSubgraph)
}

// replaceGauges sets every pair in next, then deletes the pairs set last round
// that are absent now. Unlike Reset-then-refill, a scrape never sees the vector
// empty between the two steps.
func replaceGauges(vec *prometheus.GaugeVec, last, next labelPairs) labelPairs {
	for labels, v := range next {
		vec.WithLabelValues(labels[0], labels[1]).Set(v)
	}
	for labels := range last {
		if _, ok := next[labels]; !ok {
			vec.DeleteLabelValues(labels[0], labels[1])
		}
	}
	return next
}
