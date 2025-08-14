package db

import (
	"context"
	"database/sql"
	"strings"
	"trip2g/internal/logger"
)

type DBLogger struct {
	db  DBTX
	log logger.Logger
}

func WithLogger(db DBTX, log logger.Logger) *DBLogger {
	return &DBLogger{
		db:  db,
		log: log,
	}
}

func formatSQL(s string) string {
	return strings.ReplaceAll(s, "\n", " ")
}

func (d *DBLogger) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
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
