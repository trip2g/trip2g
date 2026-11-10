package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllLiveNotesSkipsHiddenNotes(t *testing.T) {
	conn, queries, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	adminID := insertTestAdminUser(t, conn)

	visiblePathID := insertTestNotePath(t, conn, "visible.md")
	visibleVersionID := insertTestNoteVersion(t, conn, visiblePathID, "# visible")
	hiddenPathID := insertTestNotePath(t, conn, "hidden.md")
	hiddenVersionID := insertTestNoteVersion(t, conn, hiddenPathID, "# hidden")

	mustExec(t, conn, "insert into releases (id, created_by, is_live) values (1, ?, true)", adminID)
	for _, versionID := range []int64{visibleVersionID, hiddenVersionID} {
		mustExec(t, conn, "insert into release_note_versions (release_id, note_version_id) values (1, ?)", versionID)
		mustExec(t, conn, `insert into note_version_embeddings (version_id, embedding, model_id, content_hash, tokens)
			values (?, x'00', 1, x'00', 1)`, versionID)
	}

	mustExec(t, conn, "update note_paths set hidden_by = ?, hidden_at = current_timestamp where id = ?", adminID, hiddenPathID)

	notes, err := queries.AllLiveNotes(ctx)
	require.NoError(t, err)
	require.Len(t, notes, 1)
	require.Equal(t, visiblePathID, notes[0].PathID)

	embeddings, err := queries.AllLiveNoteEmbeddings(ctx)
	require.NoError(t, err)
	require.Len(t, embeddings, 1)
	require.Equal(t, visibleVersionID, embeddings[0].VersionID)
}
