package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"trip2g/internal/logger"

	mdb "trip2g/db"

	_ "trip2g/internal/dbmate/sqlite"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "modernc.org/sqlite"

	sqldblogger "github.com/simukti/sqldb-logger"
)

// writeOpPattern matches SQL write operations (insert, update, delete, replace).
var writeOpPattern = regexp.MustCompile(`(?im)^\s*(insert|update|delete|replace)\s`)

// SetupConfig contains configuration for database setup.
type SetupConfig struct {
	SkipDump     bool
	DatabaseFile string
	Logger       logger.Logger
	ReadOnly     bool
	LogQueries   bool
	CheckStatus  bool
	DevMode      bool
	// SkipMigrations skips running dbmate migrations on setup. Used by read-only
	// replicas whose DB file is owned by the replicator (LiteFS/Litestream) and
	// already migrated by the leader — running migrations would write or fail on
	// a read-only mount.
	SkipMigrations bool
	// StrictReadOnly opens the SQLite connection with mode=ro so any write
	// attempt fails with SQLITE_READONLY instead of mutating the file. Used by
	// replicas as a hard guardrail.
	StrictReadOnly bool
}

// Setup initializes the database with migrations, pragmas, and validation.
// It returns a configured database connection ready for use.
func Setup(config SetupConfig) (*sql.DB, error) {
	// Run migrations (skipped on read-only replicas: the file is owned by the
	// replicator and already migrated by the leader).
	if !config.SkipMigrations {
		err := runMigrations(config.DatabaseFile, config.SkipDump)
		if err != nil {
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}
	}

	var queryLogger logger.Logger

	if config.LogQueries && config.Logger != nil {
		prefix := "[WRITE DB]:"

		if config.ReadOnly {
			prefix = "[READ-ONLY DB]:"
		}

		queryLogger = logger.WithPrefix(config.Logger, prefix)
	}

	// Open database connection
	conn, err := openConnection(config, queryLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// TODO: do something with that... it breaks sign in
	if config.ReadOnly {
		conn.SetMaxOpenConns(25)
		conn.SetMaxIdleConns(25)
		// Release idle read connections after 30s so WAL snapshots don't accumulate
		// and prevent checkpoints, which causes SQLITE_BUSY_SNAPSHOT (517) for writers.
		conn.SetConnMaxIdleTime(30 * time.Second)
	} else {
		conn.SetMaxOpenConns(1)
		conn.SetMaxIdleConns(1)
		conn.SetConnMaxIdleTime(0)
	}

	conn.SetConnMaxLifetime(0)

	if config.CheckStatus {
		// Check foreign key constraints
		err = checkForeignKeys(conn)
		if err != nil {
			return nil, fmt.Errorf("foreign key check failed: %w", err)
		}

		// Show SQLite version
		if config.Logger != nil {
			version, versionErr := getSQLiteVersion(conn)
			if versionErr == nil {
				config.Logger.Info("SQLite database initialized", "version", version, "file", config.DatabaseFile)
			}
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

type queryInterceptor struct {
	logger       logger.Logger
	panicOnWrite bool
}

func (p *queryInterceptor) Log(ctx context.Context, level sqldblogger.Level, msg string, data map[string]interface{}) {
	if p.panicOnWrite {
		if query, ok := data["query"].(string); ok && writeOpPattern.MatchString(query) {
			panic("write operation on read-only connection: " + query)
		}
	}

	if p.logger != nil {
		p.logger.Debug(msg, "level", level, "data", data)
	}
}

// openConnection opens a SQLite database connection with optimized settings.
//
// All settings go through modernc.org/sqlite DSN parameters (`_pragma`,
// `_txlock`) because the driver applies them to every new connection in the
// pool. A one-off `db.Exec("PRAGMA ...")` only configures the single pooled
// connection it happens to run on, leaving the rest with defaults
// (busy_timeout=0, foreign_keys=OFF).
func openConnection(config SetupConfig, queryLog logger.Logger) (*sql.DB, error) {
	// build url with params
	url := &url.URL{Path: config.DatabaseFile}
	q := url.Query()
	// Strict read-only: set query_only so any write fails ("attempt to write a
	// readonly database") instead of mutating the file, while WAL reads keep
	// working normally. The replica's hard guardrail. Applied per pooled
	// connection via modernc's _pragma DSN mechanism (same as the pragmas below).
	if config.StrictReadOnly {
		q.Add("_pragma", "query_only(1)")
	}
	// The driver always applies busy_timeout before the other pragmas.
	q.Add("_pragma", "busy_timeout(20000)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "synchronous(NORMAL)")
	q.Add("_pragma", "temp_store(2)") // 2 = MEMORY
	q.Add("_pragma", "mmap_size(268435456)")
	q.Add("_pragma", "cache_size(-64000)")
	q.Add("_pragma", "wal_autocheckpoint(1000)")

	if !config.ReadOnly {
		// Writers must take the write lock at BEGIN. The default deferred
		// mode starts every transaction as a reader; upgrading to a write
		// after another connection has committed fails instantly with
		// SQLITE_BUSY_SNAPSHOT (517), and busy_timeout cannot help because
		// the snapshot is already stale. With immediate, concurrent writers
		// queue on busy_timeout instead.
		q.Set("_txlock", "immediate")
	}

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

	// panicOnWrite is a dev-only guardrail: crash loudly if a write hits the
	// read-only connection. (In production the replica relies on query_only.)
	panicOnWrite := config.DevMode && config.ReadOnly
	if queryLog != nil || panicOnWrite {
		conn = sqldblogger.OpenDriver(url.String(), conn.Driver(), &queryInterceptor{
			logger:       queryLog,
			panicOnWrite: panicOnWrite,
		})
	}

	return conn, nil
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
