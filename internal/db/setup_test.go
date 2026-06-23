package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/logger"
)

func TestSetup(t *testing.T) {
	// Create temporary database file
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "test.db")

	// Test setup
	config := SetupConfig{
		SkipDump:     true,
		DatabaseFile: dbFile,
		Logger:       &logger.TestLogger{Prefix: "[test]"},
	}

	conn, err := Setup(config)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer conn.Close()

	// Verify connection works
	if pingErr := conn.Ping(); pingErr != nil {
		t.Fatalf("Database ping failed: %v", pingErr)
	}

	// Verify foreign keys are enabled
	var fkEnabled int
	err = conn.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("Failed to check foreign keys pragma: %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("Expected foreign keys to be enabled (1), got %d", fkEnabled)
	}

	// Verify WAL mode is enabled
	var journalMode string
	err = conn.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("Failed to check journal mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("Expected journal mode to be 'wal', got %s", journalMode)
	}
}

// TestStrictReadOnlyRejectsWrites verifies that a connection opened with
// StrictReadOnly serves reads but fails writes with an error (the replica
// guardrail), and that SkipMigrations leaves the existing schema intact.
func TestStrictReadOnlyRejectsWrites(t *testing.T) {
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "test.db")

	// First, set up the DB normally (runs migrations) and seed a row.
	rw, err := Setup(SetupConfig{SkipDump: true, DatabaseFile: dbFile})
	require.NoError(t, err)
	_, err = rw.Exec("CREATE TABLE rr_probe (id INTEGER PRIMARY KEY, v TEXT)")
	require.NoError(t, err)
	_, err = rw.Exec("INSERT INTO rr_probe (id, v) VALUES (1, 'hello')")
	require.NoError(t, err)
	require.NoError(t, rw.Close())

	// Now open the same file strict read-only, skipping migrations.
	ro, err := Setup(SetupConfig{
		SkipDump:       true,
		DatabaseFile:   dbFile,
		ReadOnly:       true,
		StrictReadOnly: true,
		SkipMigrations: true,
	})
	require.NoError(t, err)
	defer ro.Close()

	// Reads work.
	var v string
	require.NoError(t, ro.QueryRow("SELECT v FROM rr_probe WHERE id = 1").Scan(&v))
	require.Equal(t, "hello", v)

	// Writes fail (do not silently mutate the file).
	_, err = ro.Exec("INSERT INTO rr_probe (id, v) VALUES (2, 'nope')")
	require.Error(t, err, "write on strict read-only connection must fail")
}

// TestPragmasAppliedToAllConnections verifies that busy_timeout and
// foreign_keys are set on every pooled connection, not just the one that
// happened to run a `PRAGMA` exec during setup. modernc.org/sqlite only
// applies DSN `_pragma=` parameters per new connection.
func TestPragmasAppliedToAllConnections(t *testing.T) {
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "test.db")

	pool, err := Setup(SetupConfig{
		SkipDump:     true,
		DatabaseFile: dbFile,
		ReadOnly:     true, // 25-connection pool
	})
	require.NoError(t, err)
	defer pool.Close()

	ctx := context.Background()

	// Pin several distinct connections from the pool at the same time.
	conns := make([]*sql.Conn, 0, 5)
	defer func() {
		for _, c := range conns {
			_ = c.Close()
		}
	}()

	for range conns {
		c, connErr := pool.Conn(ctx)
		require.NoError(t, connErr)
		conns = append(conns, c)
	}

	for i, c := range conns {
		var busyTimeout int
		require.NoError(t, c.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout))
		require.Equal(t, 20000, busyTimeout, "connection %d: busy_timeout not applied", i)

		var foreignKeys int
		require.NoError(t, c.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys))
		require.Equal(t, 1, foreignKeys, "connection %d: foreign_keys not applied", i)
	}
}

// TestWriteTxUpgradeNoBusySnapshot reproduces the e2e "database is locked
// (517)" failure: a deferred transaction takes a read snapshot, a second
// writer pool (like queueConn) commits, and the transaction's first write
// fails instantly with SQLITE_BUSY_SNAPSHOT — busy_timeout cannot help.
// Write transactions must begin immediate so writers queue instead.
func TestWriteTxUpgradeNoBusySnapshot(t *testing.T) {
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "test.db")

	config := SetupConfig{SkipDump: true, DatabaseFile: dbFile}

	connA, err := Setup(config) // like writeConn
	require.NoError(t, err)
	defer connA.Close()

	connB, err := Setup(config) // like queueConn
	require.NoError(t, err)
	defer connB.Close()

	ctx := context.Background()

	_, err = connA.ExecContext(ctx, "create table busy_snapshot_test (id integer primary key, v text)")
	require.NoError(t, err)

	tx, err := connA.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	// Read inside the transaction (with deferred BEGIN this takes a snapshot).
	var n int
	require.NoError(t, tx.QueryRowContext(ctx, "select count(*) from busy_snapshot_test").Scan(&n))

	// Concurrent write from the second pool. With deferred BEGIN it commits
	// immediately and invalidates the snapshot; with immediate BEGIN it
	// blocks on busy_timeout until the transaction commits.
	bDone := make(chan error, 1)
	go func() {
		_, bErr := connB.ExecContext(ctx, "insert into busy_snapshot_test (v) values ('b')")
		bDone <- bErr
	}()

	select {
	case bErr := <-bDone:
		require.NoError(t, bErr)
	case <-time.After(500 * time.Millisecond):
		// blocked waiting for the write lock — expected with immediate BEGIN
	}

	_, err = tx.ExecContext(ctx, "insert into busy_snapshot_test (v) values ('a')")
	require.NoError(t, err, "write inside transaction must not fail with SQLITE_BUSY_SNAPSHOT (517)")
	require.NoError(t, tx.Commit())

	select {
	case bErr := <-bDone:
		require.NoError(t, bErr)
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent insert did not complete after commit")
	}
}

func TestSetupWithNonexistentDatabase(t *testing.T) {
	// Use invalid path
	config := SetupConfig{
		SkipDump:     true,
		DatabaseFile: "/nonexistent/path/test.db",
		Logger:       nil,
	}

	_, err := Setup(config)
	if err == nil {
		t.Fatal("Expected Setup to fail with nonexistent path")
	}
}

func TestForeignKeyError(t *testing.T) {
	err := &ForeignKeyError{
		Count:      2,
		Violations: []string{"violation 1", "violation 2"},
	}

	expected := "found 2 foreign key violations: [violation 1 violation 2]"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}

	// Test single violation
	err = &ForeignKeyError{
		Count:      1,
		Violations: []string{"single violation"},
	}

	expected = "found 1 foreign key violation: single violation"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}
