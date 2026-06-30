package metrics

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// dbFileCollector is a custom Prometheus Collector that reads SQLite file-level
// metrics via PRAGMAs and the filesystem. It uses read-only PRAGMAs only
// (page_count, page_size, freelist_count) so it is safe to run on read replicas
// opened with query_only=1.
type dbFileCollector struct {
	db     *sql.DB
	dbPath string

	descSize     *prometheus.Desc
	descFreelist *prometheus.Desc
	descWAL      *prometheus.Desc
}

// NewDBFileCollector returns a Prometheus Collector that exposes three gauges:
//   - trip2g_db_size_bytes      — logical DB size (page_count * page_size)
//   - trip2g_db_freelist_pages  — freelist page count (fragmentation / VACUUM signal)
//   - trip2g_db_wal_bytes       — WAL file size from the filesystem
//
// Use the read pool so PRAGMAs never contend with the single writer.
func NewDBFileCollector(db *sql.DB, dbPath string) prometheus.Collector {
	return &dbFileCollector{
		db:     db,
		dbPath: dbPath,
		descSize: prometheus.NewDesc(
			"trip2g_db_size_bytes",
			"SQLite database logical size in bytes (page_count * page_size)",
			nil, nil,
		),
		descFreelist: prometheus.NewDesc(
			"trip2g_db_freelist_pages",
			"SQLite freelist page count (fragmentation / VACUUM-need signal)",
			nil, nil,
		),
		descWAL: prometheus.NewDesc(
			"trip2g_db_wal_bytes",
			"Size of the -wal file in bytes (WAL growth vs checkpointing)",
			nil, nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (c *dbFileCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.descSize
	ch <- c.descFreelist
	ch <- c.descWAL
}

// Collect implements prometheus.Collector. It runs read-only PRAGMAs with a
// short timeout and reads the WAL file size from the filesystem. If a PRAGMA
// query fails, that individual metric is skipped — Collect never panics.
func (c *dbFileCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var pageCount, pageSize, freelistCount int64

	if err := c.db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount); err != nil {
		// DB closed or otherwise unavailable — skip size+freelist metrics.
		goto walMetric
	}
	if err := c.db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
		goto walMetric
	}

	ch <- prometheus.MustNewConstMetric(c.descSize, prometheus.GaugeValue, float64(pageCount*pageSize))

	if err := c.db.QueryRowContext(ctx, "PRAGMA freelist_count").Scan(&freelistCount); err == nil {
		ch <- prometheus.MustNewConstMetric(c.descFreelist, prometheus.GaugeValue, float64(freelistCount))
	}

walMetric:
	// WAL size: filesystem stat only — never issue PRAGMA wal_checkpoint which
	// has side effects and may fail on query_only connections.
	var walBytes float64
	if info, err := os.Stat(c.dbPath + "-wal"); err == nil {
		walBytes = float64(info.Size())
	} else if !errors.Is(err, os.ErrNotExist) {
		// Stat error other than "not found" — emit 0 but still emit the metric.
		walBytes = 0
	}
	ch <- prometheus.MustNewConstMetric(c.descWAL, prometheus.GaugeValue, walBytes)
}

// RegisterDBCollectors registers go_sql_* pool stats collectors for the three
// database pools (read, write, queue) and one DBFileCollector for the shared
// SQLite file. Registration is idempotent: AlreadyRegisteredError is silently
// ignored.
func RegisterDBCollectors(read, write, queue *sql.DB, dbPath string) {
	mustRegisterOnce(collectors.NewDBStatsCollector(read, "read"))
	mustRegisterOnce(collectors.NewDBStatsCollector(write, "write"))
	mustRegisterOnce(collectors.NewDBStatsCollector(queue, "queue"))
	mustRegisterOnce(NewDBFileCollector(read, dbPath))
}

// mustRegisterOnce registers c on the default registry and panics only on
// errors other than AlreadyRegisteredError.
func mustRegisterOnce(c prometheus.Collector) {
	err := prometheus.Register(c)
	if err == nil {
		return
	}
	var already prometheus.AlreadyRegisteredError
	if errors.As(err, &already) {
		return
	}
	panic(err)
}
