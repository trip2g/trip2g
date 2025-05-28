package db

import (
	"path/filepath"
	"testing"

	"trip2g/internal/logger"
)

func TestSetup(t *testing.T) {
	// Create temporary database file
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "test.db")

	// Test setup
	config := SetupConfig{
		DatabaseFile: dbFile,
		Logger:       &logger.TestLogger{Prefix: "[test]"},
	}

	conn, err := Setup(config)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer conn.Close()

	// Verify connection works
	if err := conn.Ping(); err != nil {
		t.Fatalf("Database ping failed: %v", err)
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

func TestSetupWithNonexistentDatabase(t *testing.T) {
	// Use invalid path
	config := SetupConfig{
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