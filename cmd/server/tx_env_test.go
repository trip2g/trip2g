package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/logger"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// newTxTestApp builds a minimal app wired to a real SQLite file, the same way
// TestWithTransaction does. The single connection serves as both read and
// write pool (db.Setup caps a non-read-only pool at 1 open connection, like
// production), so transaction-scoping bugs surface as blocked BeginTx calls.
func newTxTestApp(t *testing.T) *app {
	t.Helper()

	dbFile := filepath.Join(t.TempDir(), "test.db")

	conn, err := db.Setup(db.SetupConfig{
		SkipDump:     true,
		DatabaseFile: dbFile,
		Logger:       &logger.TestLogger{Prefix: "[test]"},
	})
	require.NoError(t, err, "failed to setup test database")
	t.Cleanup(func() { conn.Close() })

	log := &logger.TestLogger{}
	queries := db.New(db.WithLogger(conn, logger.WithPrefix(log, "read: no tx:")))
	writeQueries := db.NewWriteQueries(db.WithLogger(conn, logger.WithPrefix(log, "write: no tx:")))

	return &app{
		appState: &appState{
			log:       log,
			conn:      conn,
			writeConn: conn,
		},
		queries:      queries,
		Queries:      queries,
		WriteQueries: writeQueries,
	}
}

// newTestRequest stores a fresh appreq bound to a in a fasthttp context, the
// way the HTTP server does before dispatching (server.go: req.Env = a).
func newTestRequest(t *testing.T, a *app) *fasthttp.RequestCtx {
	t.Helper()

	fctx := &fasthttp.RequestCtx{}
	fctx.Init(&fasthttp.Request{}, nil, nil)

	req := appreq.Acquire()
	req.Env = a
	req.Req = fctx
	req.StoreInContext()
	t.Cleanup(func() { appreq.Release(req) })

	return fctx
}

// Nested WithTransaction must fail loudly. The write pool has a single
// connection, so silently beginning a second transaction inside an open one
// blocks forever waiting for the connection the outer scope holds.
func TestWithTransactionNestedFails(t *testing.T) {
	a := newTxTestApp(t)

	err := a.WithTransaction(context.Background(), func(txCtx context.Context, env *app) (bool, error) {
		done := make(chan error, 1)
		go func() {
			done <- env.WithTransaction(txCtx, func(context.Context, *app) (bool, error) {
				return true, nil
			})
		}()

		select {
		case nestedErr := <-done:
			require.Error(t, nestedErr, "nested WithTransaction must not silently open a second write transaction")
			require.Contains(t, nestedErr.Error(), "nested")
		case <-time.After(2 * time.Second):
			t.Error("nested WithTransaction deadlocked waiting for the single write connection")
		}

		return false, nil
	})
	require.NoError(t, err)
}

// A nested call on the base env (not the tx copy) must also be detected: the
// tx env travels in ctx under txEnvKey, and the write connection is still held.
func TestWithTransactionNestedViaCtxFails(t *testing.T) {
	a := newTxTestApp(t)

	err := a.WithTransaction(context.Background(), func(txCtx context.Context, _ *app) (bool, error) {
		done := make(chan error, 1)
		go func() {
			done <- a.WithTransaction(txCtx, func(context.Context, *app) (bool, error) {
				return true, nil
			})
		}()

		select {
		case nestedErr := <-done:
			require.Error(t, nestedErr)
			require.Contains(t, nestedErr.Error(), "nested")
		case <-time.After(2 * time.Second):
			t.Error("nested WithTransaction via ctx deadlocked waiting for the single write connection")
		}

		return false, nil
	})
	require.NoError(t, err)
}

// Acquiring a second tx env on a request that already has one must fail
// loudly instead of blocking on the held write connection.
func TestAcquireTxEnvInRequestDoubleAcquireFails(t *testing.T) {
	a := newTxTestApp(t)
	fctx := newTestRequest(t, a)

	require.NoError(t, a.AcquireTxEnvInRequest(fctx, "first"))
	defer func() { _ = a.ReleaseTxEnvInRequest(fctx, false) }()

	done := make(chan error, 1)
	go func() {
		done <- a.AcquireTxEnvInRequest(fctx, "second")
	}()

	select {
	case err := <-done:
		require.Error(t, err, "second acquire on the same request must fail")
	case <-time.After(2 * time.Second):
		t.Fatal("double AcquireTxEnvInRequest deadlocked waiting for the single write connection")
	}
}

// Release must restore the base env on the request so nothing keeps using the
// dead transaction scope after commit/rollback.
func TestReleaseTxEnvInRequestRestoresBaseEnv(t *testing.T) {
	a := newTxTestApp(t)
	fctx := newTestRequest(t, a)

	require.NoError(t, a.AcquireTxEnvInRequest(fctx, "tx"))

	req, err := appreq.FromCtx(fctx)
	require.NoError(t, err)
	require.NotSame(t, a, req.Env, "acquire must swap in a tx-scoped env")

	require.NoError(t, a.ReleaseTxEnvInRequest(fctx, true))
	require.Same(t, a, req.Env, "release must restore the base env on the request")
}

// Releasing without an open transaction must return an error, not panic or
// silently succeed.
func TestReleaseTxEnvInRequestWithoutAcquireFails(t *testing.T) {
	a := newTxTestApp(t)
	fctx := newTestRequest(t, a)

	require.Error(t, a.ReleaseTxEnvInRequest(fctx, true))
}
