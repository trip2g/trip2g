package db

import (
	"database/sql"
	"fmt"
	"net/url"

	"trip2g/internal/logger"

	mdb "trip2g/db"

	_ "trip2g/internal/dbmate/sqlite"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "modernc.org/sqlite"
)

// SetupConfig contains configuration for database setup.
type SetupConfig struct {
	SkipDump     bool
	DatabaseFile string
	Logger       logger.Logger
}

// Setup initializes the database with migrations, pragmas, and validation.
// It returns a configured database connection ready for use.
func Setup(config SetupConfig) (*sql.DB, error) {
	// Run migrations
	err := runMigrations(config.DatabaseFile, config.SkipDump)
	if err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Open database connection
	conn, err := openConnection(config.DatabaseFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// TODO: do something with that... it breaks sign in
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(25)
	conn.SetConnMaxLifetime(0)
	conn.SetConnMaxIdleTime(0)

	// Enable SQLite pragmas
	err = enablePragmas(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to enable pragmas: %w", err)
	}

	// Check foreign key constraints
	err = checkForeignKeys(conn)
	if err != nil {
		return nil, fmt.Errorf("foreign key check failed: %w", err)
	}

	// Show SQLite version (optional, for debugging)
	if config.Logger != nil {
		version, versionErr := getSQLiteVersion(conn)
		if versionErr == nil {
			config.Logger.Info("SQLite database initialized", "version", version, "file", config.DatabaseFile)
		}
	}

	return conn, nil
}

// runMigrations executes database migrations using dbmate.
func runMigrations(databaseFile string, skipDump bool) error {
	u, err := url.Parse("sqlite:" + databaseFile)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	dbm := dbmate.New(u)
	dbm.MigrationsDir = []string{"migrations"}
	dbm.FS = mdb.FS
	dbm.AutoDumpSchema = !skipDump

	err = dbm.CreateAndMigrate()
	if err != nil {
		return fmt.Errorf("dbmate migration failed: %w", err)
	}

	return nil
}

// openConnection opens a SQLite database connection with optimized settings.
func openConnection(databaseFile string) (*sql.DB, error) {
	// build url with params
	url := &url.URL{Path: databaseFile}
	q := url.Query()
	q.Set("_journal", "WAL")
	q.Set("_timeout", "20000")
	q.Set("_busy_timeout", "20000")
	url.RawQuery = q.Encode()

	conn, err := sql.Open("sqlite", url.String())
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite connection: %w", err)
	}

	// Test the connection
	err = conn.Ping()
	if err != nil {
		_ = conn.Close() // Ignore close error since we're already handling ping error
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return conn, nil
}

// enablePragmas configures SQLite for optimal performance and safety.
func enablePragmas(db *sql.DB) error {
	pragmas := `
		PRAGMA foreign_keys = ON;
		PRAGMA synchronous = NORMAL;
		PRAGMA strict = ON;
		PRAGMA temp_store = MEMORY;
		PRAGMA mmap_size = 268435456;
		PRAGMA cache_size = -64000;
		PRAGMA wal_autocheckpoint = 1000;
		PRAGMA wal_checkpoint(TRUNCATE);
		PRAGMA busy_timeout = 20000;
	`

	_, err := db.Exec(pragmas)
	if err != nil {
		return fmt.Errorf("failed to enable pragmas: %w", err)
	}

	return nil
}

// checkForeignKeys validates all foreign key constraints in the database.
func checkForeignKeys(db *sql.DB) error {
	rows, err := db.Query("PRAGMA foreign_key_check;")
	if err != nil {
		return fmt.Errorf("failed to check foreign keys: %w", err)
	}
	defer rows.Close()

	violationCount := 0
	violations := []string{}

	for rows.Next() {
		var table string
		var rowid int
		var parent string
		var fkid int

		err = rows.Scan(&table, &rowid, &parent, &fkid)
		if err != nil {
			return fmt.Errorf("failed to scan foreign key check result: %w", err)
		}

		violationCount++
		violation := fmt.Sprintf("table %s (rowid %d): parent %s, fkid %d", table, rowid, parent, fkid)
		violations = append(violations, violation)
	}

	err = rows.Err()
	if err != nil {
		return fmt.Errorf("error during foreign key check: %w", err)
	}

	if violationCount > 0 {
		return &ForeignKeyError{
			Count:      violationCount,
			Violations: violations,
		}
	}

	return nil
}

// getSQLiteVersion returns the SQLite version string.
func getSQLiteVersion(db *sql.DB) (string, error) {
	var version string
	err := db.QueryRow("SELECT sqlite_version();").Scan(&version)
	if err != nil {
		return "", fmt.Errorf("failed to get SQLite version: %w", err)
	}
	return version, nil
}

// ForeignKeyError represents foreign key constraint violations.
type ForeignKeyError struct {
	Count      int
	Violations []string
}

func (e *ForeignKeyError) Error() string {
	if e.Count == 1 {
		return fmt.Sprintf("found 1 foreign key violation: %s", e.Violations[0])
	}
	return fmt.Sprintf("found %d foreign key violations: %v", e.Count, e.Violations)
}
