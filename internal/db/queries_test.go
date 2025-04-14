package db_test

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"testing"
	"time"
	"trip2g/internal/db"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/sqlite"

	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func createDB(t *testing.T) *sql.DB {
	f, err := os.CreateTemp(t.TempDir(), "trip2g_test_db_")
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
		cleanupErr := conn.Close()
		require.NoError(t, cleanupErr)

		cleanupErr = os.Remove(f.Name())
		require.NoError(t, cleanupErr)
	})

	return conn
}

func createTxQueries(t *testing.T) *db.Queries {
	conn := createDB(t)

	tx, err := conn.Begin()
	require.NoError(t, err)

	return db.New(tx)
}

func TestInsertNote(t *testing.T) {
	queries := createTxQueries(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	note := db.Note{
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

	// Collision found: 'a7ex' and 'c5kt' → base64prefix: eqoF1k

	note0 := db.Note{
		Path:    "a7ex",
		Content: "a7ex",
	}

	note1 := db.Note{
		Path:    "c5kt",
		Content: "c5kt",
	}

	err := queries.InsertNote(ctx, note0)
	require.NoError(t, err)

	err = queries.InsertNote(ctx, note1)
	require.NoError(t, err)

	paths, err := queries.AllNotePaths(ctx)
	require.NoError(t, err)

	require.Len(t, paths, 2)
	require.Equal(t, "eqoF1k", paths[0].ValueHash)
	require.Equal(t, "eqoF1kt", paths[1].ValueHash)
}
