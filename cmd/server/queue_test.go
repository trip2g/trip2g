package main

import (
	"context"
	"database/sql"
	_ "embed"
	"testing"
	"time"
	"trip2g/internal/logger"

	"maragu.dev/goqite"
	"maragu.dev/goqite/jobs"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/goqite_schema.sql
var goqiteSchema string

func newTestQueue(t *testing.T, ctx context.Context) *appQueue {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.ExecContext(ctx, goqiteSchema)
	require.NoError(t, err)

	q := goqite.New(goqite.NewOpts{
		DB:   db,
		Name: "test_queue",
	})

	log := &logger.TestLogger{Prefix: "[test_queue]"}

	runner := jobs.NewRunner(jobs.NewRunnerOpts{
		Queue:        q,
		Limit:        1,
		PollInterval: 50 * time.Millisecond,
		Log:          log,
	})

	// Register a no-op job so the runner doesn't panic
	runner.Register("noop", func(ctx context.Context, m []byte) error {
		return nil
	})

	return &appQueue{
		name:    "test_queue",
		q:       q,
		rootCtx: ctx,
		runner:  runner,
		logger:  log,
	}
}

func TestAppQueue_DoubleStartCausesDeadlock(t *testing.T) {
	// This test reproduces the production bug: calling start() twice leaks a
	// goroutine and makes stopAndWait() hang forever because the old
	// goroutine's cancel was overwritten.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	aq := newTestQueue(t, ctx)

	aq.start()
	aq.start() // double start — overwrites cancel, leaks first goroutine

	// stopAndWait should NOT hang. Give it a generous timeout.
	done := make(chan struct{})
	go func() {
		aq.stopAndWait()
		close(done)
	}()

	select {
	case <-done:
		// success — stopAndWait returned
	case <-time.After(3 * time.Second):
		t.Fatal("stopAndWait deadlocked after double start()")
	}
}

func TestAppQueue_StartOnCancelledRootCtx(t *testing.T) {
	// If rootCtx is already cancelled, start() should not silently create
	// a zombie queue that looks running but processes nothing.

	// Build queue with a live context first, then cancel it before start().
	ctx, cancel := context.WithCancel(context.Background())
	aq := newTestQueue(t, ctx)
	cancel() // cancel before start

	aq.start()

	// Queue should report as stopped since rootCtx is dead
	require.True(t, aq.isStopped(), "queue should be stopped when rootCtx is cancelled")
}

func TestAppQueue_StopStartCycleWorks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	aq := newTestQueue(t, ctx)

	// Normal lifecycle: start → stop → start → stop
	aq.start()
	require.False(t, aq.isStopped())

	aq.stopAndWait()
	require.True(t, aq.isStopped())

	aq.start()
	require.False(t, aq.isStopped())

	done := make(chan struct{})
	go func() {
		aq.stopAndWait()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(3 * time.Second):
		t.Fatal("stopAndWait deadlocked on normal stop-start cycle")
	}

	require.True(t, aq.isStopped())
}

func TestAppQueue_StartIdempotent(t *testing.T) {
	// Calling start() on an already running queue should be safe (no leak).
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	aq := newTestQueue(t, ctx)

	aq.start()
	aq.start()
	aq.start()

	done := make(chan struct{})
	go func() {
		aq.stopAndWait()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(3 * time.Second):
		t.Fatal("stopAndWait deadlocked after multiple start() calls")
	}
}
