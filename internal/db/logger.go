package db

import (
	"context"
	"database/sql"
	"strings"
	"trip2g/internal/logger"
)

type DBLogger struct {
	db        DBTX
	log       logger.Logger
	statsFunc func() sql.DBStats // if set, checked before each operation to warn on pool exhaustion
}

func WithLogger(db DBTX, log logger.Logger) *DBLogger {
	return &DBLogger{
		db:  db,
		log: log,
	}
}

// WithPoolStats attaches a stats func so the logger can warn when all connections
// are in use before a new operation attempts to acquire one. Use this only for the
// non-tx write connection where a blocked pool acquisition causes hangs.
func (d *DBLogger) WithPoolStats(stats func() sql.DBStats) *DBLogger {
	d.statsFunc = stats
	return d
}

func (d *DBLogger) checkPool(query string) {
	if d.statsFunc == nil {
		return
	}
	s := d.statsFunc()
	if s.MaxOpenConnections > 0 && s.InUse >= s.MaxOpenConnections {
		d.log.Warn("write pool exhausted — next operation will block",
			"in_use", s.InUse,
			"max", s.MaxOpenConnections,
			"wait_count", s.WaitCount,
			"query", formatSQL(query),
		)
	}
}

func formatSQL(s string) string {
	return strings.ReplaceAll(s, "\n", " ")
}

func (d *DBLogger) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	d.checkPool(query)
	d.log.Debug("ExecContext", "query", formatSQL(query), "args", args)

	res, err := d.db.ExecContext(ctx, query, args...)
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
