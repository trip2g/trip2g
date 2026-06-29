package db

import (
	"context"
	"database/sql"
	"strings"
	"sync"
)

// StmtCache wraps the read-pool DBTX and caches a long-lived *sql.Stmt per
// distinct SQL string, so the SQLite driver reuses each compiled statement
// instead of re-parsing and re-planning the SQL on every read.
//
// modernc.org/sqlite (>= v1.48) keeps a compiled-statement cache per
// connection; reusing a *sql.Stmt lets it skip the prepare (parse+plan) that
// otherwise dominates CPU under anonymous read load. A *sql.Stmt obtained from a
// *sql.DB is pool-aware: database/sql re-prepares it lazily on each physical
// connection and caches it there, so across the read pool every query is parsed
// at most once per connection, then reused.
//
// Only the non-transactional read pool is wrapped. A *sql.Stmt bound to a
// *sql.Tx dies with the transaction, so transactional paths (WithTx) and the
// single-connection write pool must NOT use this wrapper.
type StmtCache struct {
	inner DBTX    // delegate for PrepareContext (DBTX interface completeness)
	db    *sql.DB // read pool; source of pool-aware cached *sql.Stmt

	// A plain map guarded by RWMutex (with double-checked locking in getStmt),
	// not sync.Map. This is a grow-only, write-once/read-many cache — a pattern
	// sync.Map is built for — but the cached value is expensive to build (a DB
	// prepare round-trip) and owns a resource that must be Closed. RWMutex +
	// double-check guarantees each distinct query is prepared EXACTLY once;
	// sync.Map.LoadOrStore would let two goroutines racing the same cold key both
	// prepare, then orphan (never Close) the loser's *sql.Stmt. After warmup,
	// reads take only the RLock, so there is no read contention to trade away.
	mu    sync.RWMutex
	stmts map[string]*sql.Stmt
}

var _ DBTX = (*StmtCache)(nil)

// NewStmtCache wraps inner with a prepared-statement cache. readDB must be the
// same non-transactional read pool that inner ultimately runs against; cached
// statements are prepared on readDB so they stay pool-aware.
func NewStmtCache(readDB *sql.DB, inner DBTX) *StmtCache {
	return &StmtCache{
		inner: inner,
		db:    readDB,
		stmts: make(map[string]*sql.Stmt),
	}
}

// getStmt returns the cached statement for query, preparing and caching it on
// first use. Concurrent callers preparing the same SQL collapse to one stmt.
func (c *StmtCache) getStmt(ctx context.Context, query string) (*sql.Stmt, error) {
	c.mu.RLock()
	stmt, ok := c.stmts[query]
	c.mu.RUnlock()
	if ok {
		return stmt, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	// Re-check: another goroutine may have prepared it while we waited.
	if stmt, ok := c.stmts[query]; ok {
		return stmt, nil
	}
	stmt, err := c.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	c.stmts[query] = stmt
	return stmt, nil
}

// reprepare drops any cached statement for query and prepares a fresh one. Used
// to recover from a SQLITE_SCHEMA error so a runtime DDL cannot permanently
// break a cached query.
func (c *StmtCache) reprepare(ctx context.Context, query string) (*sql.Stmt, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if old, ok := c.stmts[query]; ok {
		delete(c.stmts, query)
		_ = old.Close()
	}
	stmt, err := c.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	c.stmts[query] = stmt
	return stmt, nil
}

func (c *StmtCache) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := c.getStmt(ctx, query)
	if err != nil {
		return nil, err
	}
	rows, err := stmt.QueryContext(ctx, args...)
	if isSchemaChangedErr(err) {
		stmt, rerr := c.reprepare(ctx, query)
		if rerr != nil {
			return nil, rerr
		}
		return stmt.QueryContext(ctx, args...)
	}
	return rows, err
}

func (c *StmtCache) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	stmt, err := c.getStmt(ctx, query)
	if err != nil {
		// Could not prepare a cached statement; fall back to the inner DBTX so
		// the caller still gets a *sql.Row whose deferred error surfaces on
		// Scan (a *sql.Row carrying our prepare error cannot be constructed).
		return c.inner.QueryRowContext(ctx, query, args...)
	}
	// QueryRowContext defers its error to Scan, so the SQLITE_SCHEMA fallback
	// cannot be applied here; modernc's prepare_v2 transparently re-prepares
	// after a schema change, so this path recovers without the explicit retry.
	return stmt.QueryRowContext(ctx, args...)
}

func (c *StmtCache) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	stmt, err := c.getStmt(ctx, query)
	if err != nil {
		return nil, err
	}
	res, err := stmt.ExecContext(ctx, args...)
	if isSchemaChangedErr(err) {
		stmt, rerr := c.reprepare(ctx, query)
		if rerr != nil {
			return nil, rerr
		}
		return stmt.ExecContext(ctx, args...)
	}
	return res, err
}

// PrepareContext passes through to the wrapped DBTX. The generated read queries
// never call it, but it keeps StmtCache a faithful DBTX.
func (c *StmtCache) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return c.inner.PrepareContext(ctx, query)
}

// isSchemaChangedErr reports whether err is SQLITE_SCHEMA ("database schema has
// changed"). modernc surfaces it via sqlite3_errstr(17), whose canonical text
// always contains "schema has changed".
func isSchemaChangedErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "schema has changed")
}
