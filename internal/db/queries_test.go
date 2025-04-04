package db

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/sqlite"

	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func createDB(t *testing.T) *sql.DB {
	f, err := os.CreateTemp("", "trip2g_test_db_")
	require.NoError(t, err)

	err = f.Close()
	require.NoError(t, err)

	u, err := url.Parse("sqlite:" + f.Name())
	require.NoError(t, err)

	dbm := dbmate.New(u)
	dbm.MigrationsDir = []string{"../../db/migrations"}
	dbm.AutoDumpSchema = false

	err = dbm.CreateAndMigrate()
	require.NoError(t, err)

	conn, err := sql.Open("sqlite", f.Name())
	require.NoError(t, err)

	t.Cleanup(func() {
		err := conn.Close()
		require.NoError(t, err)

		err = os.Remove(f.Name())
		require.NoError(t, err)
	})

	return conn
}

func TestInsertNote(t *testing.T) {
	conn := createDB(t)

	tx, err := conn.Begin()
	require.NoError(t, err)

	queries := New(tx)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	note := Note{
		Path:    "test.md",
		Content: "test",
	}

	err = queries.InsertNote(ctx, note)
	require.NoError(t, err)

	err = queries.InsertNote(ctx, note)
	require.Error(t, err)
}
