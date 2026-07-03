package db

import (
	"context"
	"database/sql"
	"strings"
	"time"
	"trip2g/internal/logger"
)

// defaultSlowWriteThreshold is how long a write may take on a contended pool
// before it is worth a warning. Sub-millisecond handoffs between concurrent
// writers are the normal single-writer steady state (e.g. several cron jobs
// writing on the same tick) and must stay silent.
const defaultSlowWriteThreshold = time.Second

type DBLogger struct {
	db                 DBTX
	log                logger.Logger
	statsFunc          func() sql.DBStats // if set, writes are timed to warn on real pool contention
	slowWriteThreshold time.Duration
	writeHolder        *WriteHolder // if set, in-flight writes are reported for /debug/write-holder
}

func WithLogger(db DBTX, log logger.Logger) *DBLogger {
	return &DBLogger{
		db:                 db,
		log:                log,
		slowWriteThreshold: defaultSlowWriteThreshold,
	}
}

// WithPoolStats attaches a stats func so the logger can warn when an operation
// stalls on an exhausted pool. Use this only for the non-tx write connection
// where a blocked pool acquisition causes hangs.
func (d *DBLogger) WithPoolStats(stats func() sql.DBStats) *DBLogger {
	d.statsFunc = stats
	return d
}

// WithSlowWriteThreshold overrides the contended-write warning threshold.
func (d *DBLogger) WithSlowWriteThreshold(threshold time.Duration) *DBLogger {
	d.slowWriteThreshold = threshold
	return d
}

// WithWriteHolder attaches a WriteHolder so in-flight statements on this
// connection show up in write-holder snapshots (dev diagnostic).
func (d *DBLogger) WithWriteHolder(h *WriteHolder) *DBLogger {
	d.writeHolder = h
	return d
}

// poolContended reports whether every pool connection is in use right now,
// meaning the next acquisition will have to wait.
func (d *DBLogger) poolContended() bool {
	if d.statsFunc == nil {
		return false
	}
	s := d.statsFunc()
	return s.MaxOpenConnections > 0 && s.InUse >= s.MaxOpenConnections
}

func formatSQL(s string) string {
	return strings.ReplaceAll(s, "\n", " ")
}

func (d *DBLogger) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	d.log.Debug("ExecContext", "query", formatSQL(query), "args", args)

	contended := d.poolContended()
	start := time.Now()

	releaseHolder := d.writeHolder.Acquire(formatSQL(query))

	res, err := d.db.ExecContext(ctx, query, args...)

	releaseHolder()

	if contended {
		if elapsed := time.Since(start); elapsed >= d.slowWriteThreshold {
			s := d.statsFunc()
			d.log.Warn("write stalled on exhausted write pool",
				"waited", elapsed,
				"in_use", s.InUse,
				"max", s.MaxOpenConnections,
				"wait_count", s.WaitCount,
				"query", formatSQL(query),
			)
		}
	}

	if err != nil {
		d.log.Error("ExecContext Error", "error", err, "query", query, "args", args)
	}

	return res, err
}

func (d *DBLogger) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	d.log.Debug("PrepareContext", "query", formatSQL(query))

	stmt, err := d.db.PrepareContext(ctx, query)
	if err != nil {
		d.log.Error("PrepareContext", "query", query, "err", err)
	}

	return stmt, err
}

func (d *DBLogger) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	d.log.Debug("QueryContext", "query", formatSQL(query), "args", args)

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		d.log.Error("QueryContext Error", "error", err)
	}

	return rows, err
}

func (d *DBLogger) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	d.log.Debug("QueryRowContext", "query", formatSQL(query), "args", args)

	return d.db.QueryRowContext(ctx, query, args...)
}
