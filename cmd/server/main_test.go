package main

import (
	"database/sql"
	"path/filepath"
	"testing"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/logger"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestWithTransaction(t *testing.T) {
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

	defer conn.Close()

	log := &logger.TestLogger{}

	queries := db.New(db.WithLogger(conn, logger.WithPrefix(log, "read: no tx:")))
	writeQueries := db.NewWriteQueries(db.WithLogger(conn, logger.WithPrefix(log, "write: no tx:")))

	// set app
	a := &app{
		log:          log,
		conn:         conn,
		queries:      queries,
		Queries:      queries,
		WriteQueries: writeQueries,
		graphTxs: &graphTransactions{
			EnvMap: make(map[*app]*sql.Tx),
		},
	}

	// Create properly initialized fasthttp context
	fctx := &fasthttp.RequestCtx{}
	fctx.Init(&fasthttp.Request{}, nil, nil)

	req := appreq.Acquire()
	req.Env = a
	req.Req = fctx
	req.StoreInContext() // appreq.FromCtx(ctx)
	defer appreq.Release(req)

	// test rollback
	err = a.AcquireTxEnvInRequest(fctx, "tx")
	require.NoError(t, err)

	ctxReq, err := appreq.FromCtx(fctx)
	require.NoError(t, err)

	env, ok := ctxReq.Env.(*app)
	require.True(t, ok)

	user, err := env.InsertUserWithEmail(fctx, db.InsertUserWithEmailParams{Email: "test@test.com", CreatedVia: "test"})
	require.NoError(t, err)

	selectedUser, err := env.UserByID(fctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, "test@test.com", selectedUser.Email.String)

	err = a.ReleaseTxEnvInRequest(fctx, false)
	require.NoError(t, err)

	_, err = a.UserByID(fctx, user.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// test commit
	err = a.AcquireTxEnvInRequest(fctx, "tx")
	require.NoError(t, err)

	ctxReq, err = appreq.FromCtx(fctx)
	require.NoError(t, err)

	env, ok = ctxReq.Env.(*app)
	require.True(t, ok)

	user, err = env.InsertUserWithEmail(fctx, db.InsertUserWithEmailParams{Email: "test@test.com", CreatedVia: "test"})
	require.NoError(t, err)

	selectedUser, err = env.UserByID(fctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, "test@test.com", selectedUser.Email.String)

	err = a.ReleaseTxEnvInRequest(fctx, true)
	require.NoError(t, err)

	selectedUser, err = a.UserByID(fctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, "test@test.com", selectedUser.Email.String)
}
