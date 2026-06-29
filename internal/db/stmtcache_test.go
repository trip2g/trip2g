package db

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/logger"
)

// openStmtCacheTestDB opens a temp-file modernc SQLite DB seeded with a small
// table, and returns a StmtCache wrapping it (the read DBTX is the bare *sql.DB
// for these unit tests — wiring with WithLogger is exercised separately).
func openStmtCacheTestDB(t *testing.T) (*sql.DB, *StmtCache) {
	t.Helper()

	dir := t.TempDir()
	dbh, err := sql.Open("sqlite", filepath.Join(dir, "stmtcache.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = dbh.Close() })

	ctx := context.Background()
	_, err = dbh.ExecContext(ctx, "CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)")
	require.NoError(t, err)
	_, err = dbh.ExecContext(ctx, "INSERT INTO items (id, name) VALUES (1, 'alpha'), (2, 'beta')")
	require.NoError(t, err)

	return dbh, NewStmtCache(dbh, dbh)
}

// TestStmtCacheReusesPreparedStatement proves a repeated identical SQL string
// reuses the SAME cached *sql.Stmt (cache hit — no re-prepare).
func TestStmtCacheReusesPreparedStatement(t *testing.T) {
	dbh, cache := openStmtCacheTestDB(t)
	_ = dbh
	ctx := context.Background()

	const q = "SELECT name FROM items WHERE id = ?"

	rows1, err := cache.QueryContext(ctx, q, 1)
	require.NoError(t, err)
	require.NoError(t, rows1.Close())

	cache.mu.RLock()
	stmt1, ok := cache.stmts[q]
	size1 := len(cache.stmts)
	cache.mu.RUnlock()
	require.True(t, ok, "statement must be cached after first use")
	require.Equal(t, 1, size1)

	// Second call with the same SQL must reuse the same *sql.Stmt pointer.
	rows2, err := cache.QueryContext(ctx, q, 2)
	require.NoError(t, err)
	require.NoError(t, rows2.Close())

	cache.mu.RLock()
	stmt2 := cache.stmts[q]
	size2 := len(cache.stmts)
	cache.mu.RUnlock()
	require.Same(t, stmt1, stmt2, "identical SQL must reuse the cached statement")
	require.Equal(t, 1, size2, "cache must not grow for identical SQL")

	// A different SQL string gets its own cache entry.
	row := cache.QueryRowContext(ctx, "SELECT count(*) FROM items")
	var n int
	require.NoError(t, row.Scan(&n))
	require.Equal(t, 2, n)

	cache.mu.RLock()
	size3 := len(cache.stmts)
	cache.mu.RUnlock()
	require.Equal(t, 2, size3, "distinct SQL must add a cache entry")
}

// TestStmtCacheFunctionalParity proves queries through the wrapper return the
// same results as the unwrapped *sql.DB.
func TestStmtCacheFunctionalParity(t *testing.T) {
	dbh, cache := openStmtCacheTestDB(t)
	ctx := context.Background()

	const q = "SELECT name FROM items WHERE id = ?"

	// QueryRowContext parity.
	var direct, wrapped string
	require.NoError(t, dbh.QueryRowContext(ctx, q, 1).Scan(&direct))
	require.NoError(t, cache.QueryRowContext(ctx, q, 1).Scan(&wrapped))
	require.Equal(t, direct, wrapped)
	require.Equal(t, "alpha", wrapped)

	// QueryContext parity: collect all names ordered.
	rows, err := cache.QueryContext(ctx, "SELECT name FROM items ORDER BY id")
	require.NoError(t, err)
	defer rows.Close()
	var got []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		got = append(got, name)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"alpha", "beta"}, got)

	// ExecContext parity through the wrapper (cached stmt path).
	res, err := cache.ExecContext(ctx, "UPDATE items SET name = ? WHERE id = ?", "ALPHA", 1)
	require.NoError(t, err)
	affected, err := res.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)

	require.NoError(t, cache.QueryRowContext(ctx, q, 1).Scan(&wrapped))
	require.Equal(t, "ALPHA", wrapped)
}

// TestStmtCacheRecoversAfterSchemaChange proves a cached statement survives a
// schema change: a subsequent call through the wrapper recovers (re-prepares)
// and returns correct results rather than erroring.
//
// modernc.org/sqlite uses sqlite3_prepare_v2, which transparently re-prepares a
// statement after a non-breaking schema change, so SQLITE_SCHEMA rarely surfaces
// to the caller; the explicit fallback in StmtCache is defense-in-depth for the
// case where it does. This test guards the required end-to-end behavior.
func TestStmtCacheRecoversAfterSchemaChange(t *testing.T) {
	dbh, cache := openStmtCacheTestDB(t)
	ctx := context.Background()

	const q = "SELECT name FROM items WHERE id = ?"

	var name string
	require.NoError(t, cache.QueryRowContext(ctx, q, 1).Scan(&name))
	require.Equal(t, "alpha", name)

	// Cache the statement via QueryContext too (the path with the explicit fallback).
	rows, err := cache.QueryContext(ctx, q, 1)
	require.NoError(t, err)
	require.NoError(t, rows.Close())

	// Schema change that bumps the schema cookie but keeps the query valid.
	_, err = dbh.ExecContext(ctx, "ALTER TABLE items ADD COLUMN extra TEXT")
	require.NoError(t, err)

	// Subsequent call on the cached SQL must still work.
	rows, err = cache.QueryContext(ctx, q, 1)
	require.NoError(t, err, "cached query must recover after a schema change")
	defer rows.Close()
	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&name))
	require.Equal(t, "alpha", name)
}

// TestIsSchemaChangedErr covers the SQLITE_SCHEMA detection predicate used by
// the fallback. modernc surfaces SQLITE_SCHEMA via sqlite3_errstr(17), whose
// canonical text is "database schema has changed".
func TestIsSchemaChangedErr(t *testing.T) {
	require.False(t, isSchemaChangedErr(nil))
	require.False(t, isSchemaChangedErr(errors.New("no such table: items")))
	require.True(t, isSchemaChangedErr(errors.New("database schema has changed (17)")))

	// A real modernc error for an unrelated failure must not be treated as a
	// schema change.
	dir := t.TempDir()
	dbh, err := sql.Open("sqlite", filepath.Join(dir, "x.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = dbh.Close() })
	_, realErr := dbh.ExecContext(context.Background(), "SELECT * FROM no_such_table")
	require.Error(t, realErr)
	require.False(t, isSchemaChangedErr(realErr))
}

// TestStmtCacheWiringServesGeneratedReadQuery exercises the exact read-pool
// wiring from cmd/server/main.go — Setup (migrated 25-conn read pool) ->
// WithLogger -> NewStmtCache -> db.New — and runs a real generated sqlc read
// query through it twice, proving the integrated read path works and caches.
func TestStmtCacheWiringServesGeneratedReadQuery(t *testing.T) {
	dir := t.TempDir()
	conn, err := Setup(SetupConfig{
		SkipDump:     true,
		DatabaseFile: filepath.Join(dir, "wire.db"),
		ReadOnly:     true, // 25-connection read pool, as in main.go
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	cache := NewStmtCache(conn, WithLogger(conn, &logger.TestLogger{Prefix: "[read]"}))
	queries := New(cache)

	ctx := context.Background()
	n1, err := queries.CountNoteVersions(ctx)
	require.NoError(t, err)
	n2, err := queries.CountNoteVersions(ctx)
	require.NoError(t, err)
	require.Equal(t, n1, n2)

	cache.mu.RLock()
	size := len(cache.stmts)
	cache.mu.RUnlock()
	require.Equal(t, 1, size, "generated read query must be cached once through the real wiring")
}

// TestStmtCacheConcurrentReuse proves the cache is safe under concurrent reads
// and prepares exactly one statement per distinct SQL.
func TestStmtCacheConcurrentReuse(t *testing.T) {
	_, cache := openStmtCacheTestDB(t)
	ctx := context.Background()

	const q = "SELECT name FROM items WHERE id = ?"

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			row := cache.QueryRowContext(ctx, q, 1)
			var name string
			_ = row.Scan(&name)
		}()
	}
	wg.Wait()

	cache.mu.RLock()
	defer cache.mu.RUnlock()
	require.Equal(t, 1, len(cache.stmts), "concurrent identical SQL must prepare exactly one statement")
}
