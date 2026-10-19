package metrics

import (
	"context"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"trip2g/internal/case/system/federationtopology"
	"trip2g/internal/db"
	"trip2g/internal/logger"
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
	Logger() logger.Logger
	federationtopology.Env
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
	fedSelf          *prometheus.GaugeVec
	fedKeys          *prometheus.GaugeVec
	fedKeySubgraphs  *prometheus.GaugeVec
	fedPeers         *prometheus.GaugeVec
	fedSelfLabels    []string
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

	fedSecrets := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "trip2g_federation_secrets",
		Help: "Number of live (not revoked) federation keys by direction (inbound, outbound)",
	}, []string{"direction"})
	reg.MustRegister(fedSecrets)

	fedSubgraphs := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "trip2g_federation_secret_subgraphs",
		Help: "Number of live federation keys scoped to each subgraph, by direction",
	}, []string{"direction", "subgraph"})
	reg.MustRegister(fedSubgraphs)

	fedSelf := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "trip2g_federation_self_info",
		Help: "This base as peers address it: its kb_id and MCP URL. Always 1",
	}, []string{"kb_id", "mcp_url"})
	reg.MustRegister(fedSelf)

	fedKeys := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "trip2g_federation_key_info",
		Help: "One series per live federation key, by direction, kid and the peer's kb_url (outbound only). Always 1",
	}, []string{"direction", "kid", "kb_url"})
	reg.MustRegister(fedKeys)

	fedKeySubgraphs := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "trip2g_federation_key_subgraph_info",
		Help: "One series per subgraph a live federation key is scoped to. Always 1",
	}, []string{"direction", "kid", "subgraph"})
	reg.MustRegister(fedKeySubgraphs)

	fedPeers := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "trip2g_federation_peer_info",
		Help: "One series per KB-note: the peer outbound calls are labelled with, and the kb_url it points to. Always 1",
	}, []string{"peer", "kb_url"})
	reg.MustRegister(fedPeers)

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
		fedSelf:          fedSelf,
		fedKeys:          fedKeys,
		fedKeySubgraphs:  fedKeySubgraphs,
		fedPeers:         fedPeers,
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

	u.updateFederationMetrics(ctx)
}

// Federation key directions: an inbound key lets a peer read this base, an
// outbound key lets this base read a peer (it carries the peer's kb_url).
const (
	directionInbound  = "inbound"
	directionOutbound = "outbound"
)

func (u *Updater) updateFederationMetrics(ctx context.Context) {
	topology, err := federationtopology.Resolve(ctx, u.env)
	if err != nil {
		u.env.Logger().Error("federation metrics: resolve topology, keeping last values", "error", err)
		return
	}

	// Replaced, not reset, so an always-present series never blinks out
	// between a reset and a scrape.
	self := []string{topology.Self.KBID, topology.Self.MCPURL}
	if u.fedSelfLabels != nil && !slices.Equal(u.fedSelfLabels, self) {
		u.fedSelf.DeleteLabelValues(u.fedSelfLabels...)
	}
	u.fedSelf.WithLabelValues(self...).Set(1)
	u.fedSelfLabels = self

	// Reset so a revoked or deleted key, and a subgraph whose last key was
	// unscoped, drops out.
	u.fedKeys.Reset()
	u.fedKeySubgraphs.Reset()
	u.fedSubgraphs.Reset()
	u.fedPeers.Reset()

	keys := map[string]int{directionInbound: 0, directionOutbound: 0}
	perSubgraph := map[[2]string]int{}
	record := func(direction string, secrets []federationtopology.Secret) {
		for _, secret := range secrets {
			if secret.RevokedAt != nil {
				continue
			}
			keys[direction]++
			kbURL := ""
			if secret.KBURL != nil {
				kbURL = labelURL(*secret.KBURL)
			}
			u.fedKeys.WithLabelValues(direction, secret.KID, kbURL).Set(1)
			for _, subgraph := range secret.Subgraphs {
				perSubgraph[[2]string{direction, subgraph.Name}]++
				u.fedKeySubgraphs.WithLabelValues(direction, secret.KID, subgraph.Name).Set(1)
			}
		}
	}
	record(directionInbound, topology.Inbound)
	record(directionOutbound, topology.Outbound)

	for direction, n := range keys {
		u.fedSecrets.WithLabelValues(direction).Set(float64(n))
	}
	for labels, n := range perSubgraph {
		u.fedSubgraphs.WithLabelValues(labels[0], labels[1]).Set(float64(n))
	}
	for _, note := range topology.KBNotes {
		u.fedPeers.WithLabelValues(note.KBID, labelURL(note.KBURL)).Set(1)
	}
}

// labelURL keeps scheme, host and path. A URL typed into a key or a KB-note
// may carry credentials in its userinfo or query, and /metrics has no auth.
func labelURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host + parsed.EscapedPath()
}
