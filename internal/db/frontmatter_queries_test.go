package db_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
)

func TestRefreshNoteVersionFrontmatterKeyVisibilityUsesLatestVersions(t *testing.T) {
	conn, _, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	pathID := insertTestNotePath(t, conn, "notes/a.md")
	versionID := insertTestNoteVersion(t, conn, pathID, "# note")
	mustExec(t, conn, "update note_paths set version_count = 1 where id = ?", pathID)

	queries := db.NewWriteQueries(conn)
	require.NoError(t, queries.UpsertNoteVersionFrontmatterKey(ctx, db.UpsertNoteVersionFrontmatterKeyParams{
		Value:              "fleet_id",
		CreatedByVersionID: versionID,
	}))
	require.NoError(t, queries.InsertNoteVersionFrontmatterKey(ctx, db.InsertNoteVersionFrontmatterKeyParams{
		NoteVersionID: versionID,
		KeyID:         "fleet_id",
	}))

	require.NoError(t, queries.RefreshNoteVersionFrontmatterKeyVisibility(ctx))
	assertFrontmatterKeyHiddenAt(t, conn, "fleet_id", false)

	require.NoError(t, queries.DeleteNoteVersionFrontmatterKeys(ctx, versionID))
	require.NoError(t, queries.RefreshNoteVersionFrontmatterKeyVisibility(ctx))
	assertFrontmatterKeyHiddenAt(t, conn, "fleet_id", true)
}

func TestFilterNotePathIDsByFrontmatterEquals(t *testing.T) {
	conn, queries, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	pathID := insertTestNotePath(t, conn, "fleet-filter.md")
	versionID := insertTestNoteVersion(t, conn, pathID, "---\nfleet_id: codellm\n---\n")
	mustExec(t, conn, "update note_paths set version_count = 1 where id = ?", pathID)
	mustExec(t, conn, "insert into note_version_frontmatters (version_id, data) values (?, ?)", versionID, `{"fleet_id":"codellm"}`)

	key := "fleet_id"
	pathIDs, err := queries.FilterNotePathIDsByFrontmatterEquals(
		ctx,
		db.FilterNotePathIDsByFrontmatterEqualsParams{
			Key:   &key,
			Value: "codellm",
		},
	)
	require.NoError(t, err)
	require.Equal(t, []int64{pathID}, pathIDs)
}

func assertFrontmatterKeyHiddenAt(t *testing.T, conn *sql.DB, key string, hidden bool) {
	t.Helper()
	var hiddenAt *string
	err := conn.QueryRow(
		"select hidden_at from note_version_frontmatter_key_values where value = ?",
		key,
	).Scan(&hiddenAt)
	require.NoError(t, err)
	if hidden {
		require.NotNil(t, hiddenAt)
		return
	}
	require.Nil(t, hiddenAt)
}
