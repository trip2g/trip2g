package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"
)

// AcquireWriterSlot probes whether the SQLite write lock is currently
// grabbable on the given database file. It opens a dedicated single-connection
// pool with `_txlock=immediate` and, in a loop bounded by timeout, runs
// `BEGIN IMMEDIATE; COMMIT` until it succeeds once. A successful probe proves
// the write lock could be taken at that moment.
//
// IMPORTANT: this is a PROBE, not a hold. The transaction is committed
// immediately — holding it open would block the app's own writeConn. It does
// NOT guarantee hard single-writer mutual exclusion across processes; that is
// deferred to a later phase (Consul). It is an honest-but-minimal signal that
// the previous instance has released the writer slot.
//
// The probe connection is closed before returning.
func AcquireWriterSlot(ctx context.Context, databaseFile string, timeout time.Duration) error {
	// build url with the immediate txlock so BEGIN takes the write lock.
	// A short busy_timeout makes each BEGIN IMMEDIATE attempt fail fast when the
	// lock is held; the outer loop (bounded by timeout) governs total wait, so
	// the WriterAcquireTimeout is honored instead of a single 20s SQLite block.
	u := &url.URL{Path: databaseFile}
	q := u.Query()
	q.Add("_pragma", "busy_timeout(200)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Set("_txlock", "immediate")
	u.RawQuery = q.Encode()

	conn, err := sql.Open("sqlite", u.String())
	if err != nil {
		return fmt.Errorf("failed to open writer-slot probe connection: %w", err)
	}
	defer conn.Close()

	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)

	deadline := time.Now().Add(timeout)

	var lastErr error
	for {
		probeErr := probeWriteLock(ctx, conn)
		if probeErr == nil {
			return nil
		}
		lastErr = probeErr

		if time.Now().After(deadline) {
			return fmt.Errorf("failed to acquire writer slot within %s: %w", timeout, lastErr)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("writer-slot acquire cancelled: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// probeWriteLock runs a single BEGIN IMMEDIATE; COMMIT cycle. It returns nil
// when the write lock was grabbable. The transaction is always committed (or
// rolled back on error) so the lock is released immediately.
func probeWriteLock(ctx context.Context, conn *sql.DB) error {
	// BeginTx on a connection opened with _txlock=immediate issues
	// BEGIN IMMEDIATE, which takes the write lock right away.
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin immediate probe transaction: %w", err)
	}

	if commitErr := tx.Commit(); commitErr != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to commit probe transaction: %w", commitErr)
	}

	return nil
}
