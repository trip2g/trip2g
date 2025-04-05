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

func createTxQueries(t *testing.T) *Queries {
	conn := createDB(t)

	tx, err := conn.Begin()
	require.NoError(t, err)

	return New(tx)
}

func TestInsertNote(t *testing.T) {
	queries := createTxQueries(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	note := Note{
		Path:    "test.md",
		Content: "test",
	}

	err := queries.InsertNote(ctx, note)
	require.NoError(t, err)

	err = queries.InsertNote(ctx, note)
	require.Error(t, err)

	note.Content = "test2"

	err = queries.InsertNote(ctx, note)
	require.NoError(t, err)

	paths, err := queries.AllNotePaths(ctx)
	require.NoError(t, err)
	require.Len(t, paths, 1)

	versions, err := queries.AllNoteVersions(ctx)
	require.NoError(t, err)
	require.Len(t, versions, 2)
}

func TestCheckPathHashCollisions(t *testing.T) {
	queries := createTxQueries(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// fek2 and ip7f → base64prefix: 0q5foe

	note0 := Note{
		Path:    "fek2",
		Content: "fek2",
	}

	note1 := Note{
		Path:    "ip7f",
		Content: "ip7f",
	}

	err := queries.InsertNote(ctx, note0)
	require.NoError(t, err)

	err = queries.InsertNote(ctx, note1)
	require.NoError(t, err)

	paths, err := queries.AllNotePaths(ctx)
	require.NoError(t, err)

	require.Len(t, paths, 2)
	require.Equal(t, "0q5foe", paths[0].PathHash)
	require.Equal(t, "0q5foeA", paths[1].PathHash)
}
