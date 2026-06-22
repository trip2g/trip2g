package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestAcquireWriterSlotSucceedsWhenFree verifies the probe returns nil quickly
// when nothing holds the write lock.
func TestAcquireWriterSlotSucceedsWhenFree(t *testing.T) {
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "test.db")

	// Migrate/create the DB first so the file exists with WAL set up.
	conn, err := Setup(SetupConfig{SkipDump: true, DatabaseFile: dbFile})
	require.NoError(t, err)
	defer conn.Close()

	ctx := context.Background()

	err = AcquireWriterSlot(ctx, dbFile, 2*time.Second)
	require.NoError(t, err, "probe should succeed when the write lock is free")
}

// TestAcquireWriterSlotBlocksWhileHeldThenSucceeds verifies the probe fails
// within a short timeout while another connection holds BEGIN IMMEDIATE, and
// succeeds once that transaction is released. Mirrors the busy-snapshot test
// style in setup_test.go.
func TestAcquireWriterSlotBlocksWhileHeldThenSucceeds(t *testing.T) {
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "test.db")

	// holder is a writeConn-like pool (single conn, _txlock=immediate).
	holder, err := Setup(SetupConfig{SkipDump: true, DatabaseFile: dbFile})
	require.NoError(t, err)
	defer holder.Close()

	ctx := context.Background()

	// A schema write so the file is initialised and the holder owns the lock.
	_, err = holder.ExecContext(ctx, "create table writer_slot_test (id integer primary key)")
	require.NoError(t, err)

	// Hold BEGIN IMMEDIATE open: this grabs the write lock and does not release.
	tx, err := holder.BeginTx(ctx, nil)
	require.NoError(t, err)
	// Force the write lock to be taken (immediate txlock takes it at BEGIN, but
	// issue a write to be certain the lock is held).
	_, err = tx.ExecContext(ctx, "insert into writer_slot_test default values")
	require.NoError(t, err)

	// While held, the probe must fail within its short timeout. The probe's
	// busy_timeout is 20s, so a timeout shorter than that bounds the failure.
	err = AcquireWriterSlot(ctx, dbFile, 300*time.Millisecond)
	require.Error(t, err, "probe must fail while another connection holds the write lock")

	// Release the lock.
	require.NoError(t, tx.Commit())

	// Now the probe must succeed.
	err = AcquireWriterSlot(ctx, dbFile, 5*time.Second)
	require.NoError(t, err, "probe should succeed after the write lock is released")
}
