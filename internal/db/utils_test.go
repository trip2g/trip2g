package db_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"trip2g/internal/db"
	"trip2g/internal/logger"

	"github.com/stretchr/testify/require"
)

// setupTestDB creates a temporary test database with migrations applied.
func setupTestDB(t *testing.T) (*sql.DB, *db.Queries, func()) {
	t.Helper()

	// Create temporary directory
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "test.db")

	// Setup test database
	config := db.SetupConfig{
		SkipDump:     true,
		DatabaseFile: dbFile,
		Logger:       &logger.TestLogger{Prefix: "[test]"},
	}

	conn, err := db.Setup(config)
	require.NoError(t, err, "Failed to setup test database")

	// Create queries instance
	queries := db.New(conn)

	// Return cleanup function
	cleanup := func() {
		conn.Close()
	}

	return conn, queries, cleanup
}

// mustExec executes a SQL statement and fails the test on error.
func mustExec(t *testing.T, db *sql.DB, query string, args ...interface{}) {
	t.Helper()

	_, err := db.Exec(query, args...)
	require.NoError(t, err, "Failed to execute query: %s", query)
}

// insertTestNotePath creates a test note path and returns the ID.
func insertTestNotePath(t *testing.T, db *sql.DB, path string) int64 {
	t.Helper()

	var pathID int64
	err := db.QueryRow(`
		insert into note_paths (value, value_hash, latest_content_hash) 
		values (?, ?, ?)
		returning id
	`, path, "hash-"+path, "content-hash-"+path).Scan(&pathID)
	require.NoError(t, err, "Failed to insert test note path")

	return pathID
}

// insertTestNoteVersion creates a test note version.
func insertTestNoteVersion(t *testing.T, db *sql.DB, pathID int64, content string) int64 {
	t.Helper()

	var versionID int64
	err := db.QueryRow(`
		insert into note_versions (path_id, version, content) 
		values (?, 1, ?)
		returning id
	`, pathID, content).Scan(&versionID)
	require.NoError(t, err, "Failed to insert test note version")

	return versionID
}
