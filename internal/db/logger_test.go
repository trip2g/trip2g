package db

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type captureLogger struct {
	mu    sync.Mutex
	warns []string
}

func (l *captureLogger) Info(string, ...interface{})  {}
func (l *captureLogger) Error(string, ...interface{}) {}
func (l *captureLogger) Debug(string, ...interface{}) {}

func (l *captureLogger) Warn(msg string, _ ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warns = append(l.warns, msg)
}

func (l *captureLogger) warnCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.warns)
}

type fakeDBTX struct {
	delay time.Duration
}

func (f *fakeDBTX) ExecContext(context.Context, string, ...interface{}) (sql.Result, error) {
	time.Sleep(f.delay)
	return nil, nil
}

func (f *fakeDBTX) PrepareContext(context.Context, string) (*sql.Stmt, error) { return nil, nil }

func (f *fakeDBTX) QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (f *fakeDBTX) QueryRowContext(context.Context, string, ...interface{}) *sql.Row { return nil }

func fullPoolStats() sql.DBStats {
	return sql.DBStats{MaxOpenConnections: 1, InUse: 1, WaitCount: 42}
}

func freePoolStats() sql.DBStats {
	return sql.DBStats{MaxOpenConnections: 1, InUse: 0}
}

// A momentarily busy pool with a fast write is the normal single-writer
// steady state (e.g. two cron jobs writing on the same tick) — no warning.
func TestDBLogger_FastWriteOnBusyPool_NoWarn(t *testing.T) {
	log := &captureLogger{}
	d := WithLogger(&fakeDBTX{}, log).
		WithPoolStats(fullPoolStats).
		WithSlowWriteThreshold(50 * time.Millisecond)

	_, err := d.ExecContext(context.Background(), "update t set x = 1")
	require.NoError(t, err)
	require.Equal(t, 0, log.warnCount())
}

// A write that stalls while the pool is exhausted is the real signal: warn
// with the measured duration.
func TestDBLogger_SlowWriteOnBusyPool_Warns(t *testing.T) {
	log := &captureLogger{}
	d := WithLogger(&fakeDBTX{delay: 20 * time.Millisecond}, log).
		WithPoolStats(fullPoolStats).
		WithSlowWriteThreshold(5 * time.Millisecond)

	_, err := d.ExecContext(context.Background(), "update t set x = 1")
	require.NoError(t, err)
	require.Equal(t, 1, log.warnCount())
}

// A slow write on a free pool is not contention — no warning.
func TestDBLogger_SlowWriteOnFreePool_NoWarn(t *testing.T) {
	log := &captureLogger{}
	d := WithLogger(&fakeDBTX{delay: 20 * time.Millisecond}, log).
		WithPoolStats(freePoolStats).
		WithSlowWriteThreshold(5 * time.Millisecond)

	_, err := d.ExecContext(context.Background(), "update t set x = 1")
	require.NoError(t, err)
	require.Equal(t, 0, log.warnCount())
}

// Without pool stats attached nothing changes.
func TestDBLogger_NoPoolStats_NoWarn(t *testing.T) {
	log := &captureLogger{}
	d := WithLogger(&fakeDBTX{delay: 20 * time.Millisecond}, log)

	_, err := d.ExecContext(context.Background(), "update t set x = 1")
	require.NoError(t, err)
	require.Equal(t, 0, log.warnCount())
}

// The write DBLogger reports in-flight statements to the WriteHolder so
// /debug/write-holder can name a long-running statement (e.g. VACUUM).
func TestDBLogger_WriteHolderTracksExec(t *testing.T) {
	holder := NewWriteHolder(true)
	started := make(chan struct{})
	release := make(chan struct{})
	d := WithLogger(&blockingDBTX{started: started, release: release}, &captureLogger{}).
		WithWriteHolder(holder)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = d.ExecContext(context.Background(), "VACUUM")
	}()

	<-started
	infos := holder.Snapshot()
	require.Len(t, infos, 1)
	require.Equal(t, "VACUUM", infos[0].Label)

	close(release)
	<-done
	require.Empty(t, holder.Snapshot())
}

type blockingDBTX struct {
	fakeDBTX
	started chan struct{}
	release chan struct{}
}

func (b *blockingDBTX) ExecContext(context.Context, string, ...interface{}) (sql.Result, error) {
	close(b.started)
	<-b.release
	return nil, nil
}
