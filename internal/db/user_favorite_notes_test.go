package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
)

func TestListUserFavoriteNotesSkipsHiddenNotes(t *testing.T) {
	conn, queries, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	adminID := insertTestAdminUser(t, conn)

	visiblePathID := insertTestNotePath(t, conn, "visible.md")
	visibleVersionID := insertTestNoteVersion(t, conn, visiblePathID, "# visible")
	hiddenPathID := insertTestNotePath(t, conn, "hidden.md")
	hiddenVersionID := insertTestNoteVersion(t, conn, hiddenPathID, "# hidden")

	writes := db.NewWriteQueries(conn)
	for _, versionID := range []int64{visibleVersionID, hiddenVersionID} {
		params := db.InsertUserFavoriteNoteParams{UserID: adminID, NoteVersionID: versionID}
		require.NoError(t, writes.InsertUserFavoriteNote(ctx, params))
	}

	mustExec(t, conn, "update note_paths set hidden_by = ?, hidden_at = current_timestamp where id = ?", adminID, hiddenPathID)

	rows, err := queries.ListUserFavoriteNotes(ctx, adminID)
	require.NoError(t, err)
	require.Equal(t, []db.ListUserFavoriteNotesRow{{PathID: visiblePathID, VersionID: visibleVersionID}}, rows)

	var stored int
	err = conn.QueryRow("select count(*) from user_favorite_notes where user_id = ?", adminID).Scan(&stored)
	require.NoError(t, err)
	require.Equal(t, 2, stored)
}
