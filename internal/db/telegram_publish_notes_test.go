package db_test

import (
	"context"
	"testing"

	"trip2g/internal/db"

	"github.com/stretchr/testify/require"
)

func TestListAllTelegramPublishNotes(t *testing.T) {
	ctx := context.Background()
	conn, queries, cleanup := setupTestDB(t)
	defer cleanup()

	// Create test note paths
	path1 := insertTestNotePath(t, conn, "/test/scheduled")
	path2 := insertTestNotePath(t, conn, "/test/published")
	path3 := insertTestNotePath(t, conn, "/test/invalid") // publish_at < created_at

	// Create note versions
	insertTestNoteVersion(t, conn, path1, "Scheduled note content")
	version2 := insertTestNoteVersion(t, conn, path2, "Published note content")
	insertTestNoteVersion(t, conn, path3, "Invalid note content")

	// Insert telegram publish notes
	// 1. Scheduled for PAST (should appear in scheduled - ready to send)
	mustExec(t, conn, `
		insert into telegram_publish_notes (note_path_id, created_at, publish_at, published_at)
		values (?, datetime('now', '-2 hours'), datetime('now', '-1 hour'), null)
	`, path1)

	// 2. Already published (should not appear in scheduled)
	mustExec(t, conn, `
		insert into telegram_publish_notes (note_path_id, created_at, publish_at, published_at, published_version_id)
		values (?, datetime('now', '-2 days'), datetime('now', '-1 day'), datetime('now', '-1 hour'), ?)
	`, path2, version2)

	// 3. CRITICAL TEST: Scheduled for FUTURE (should NOT be scheduled yet)
	mustExec(t, conn, `
		insert into telegram_publish_notes (note_path_id, created_at, publish_at, published_at)
		values (?, datetime('now'), datetime('now', '+1 day'), null)
	`, path3)

	t.Run("ListAllTelegramPublishNotes_show_scheduled_only", func(t *testing.T) {
		params := db.ListAllTelegramPublishNotesParams{
			ShowScheduled: true,
			ShowSent:      false,
			ShowOutdated:  true,
		}
		notes, err := queries.ListAllTelegramPublishNotes(ctx, params)
		require.NoError(t, err)

		// Should return path1 and path3 (unpublished)
		require.Len(t, notes, 2)

		// Verify specific notes returned (ordered by publish_at)
		require.Equal(t, path1, notes[0].NotePathID) // path1 has earlier publish_at (-1 hour)
		require.Equal(t, path3, notes[1].NotePathID) // path3 has later publish_at (+1 day)

		// Verify no published notes are included
		require.False(t, notes[0].PublishedAt.Valid)
		require.False(t, notes[1].PublishedAt.Valid)
	})

	t.Run("ListAllTelegramPublishNotes_show_all", func(t *testing.T) {
		params := db.ListAllTelegramPublishNotesParams{
			ShowScheduled: true,
			ShowSent:      true,
			ShowOutdated:  true,
		}
		notes, err := queries.ListAllTelegramPublishNotes(ctx, params)
		require.NoError(t, err)

		// Should include all notes (path1, path2, path3)
		require.Len(t, notes, 3)

		// Verify specific notes returned (ordered by publish_at)
		require.Equal(t, path2, notes[0].NotePathID) // published note (earliest publish_at: -1 day)
		require.Equal(t, path1, notes[1].NotePathID) // scheduled note (publish_at: -1 hour)
		require.Equal(t, path3, notes[2].NotePathID) // future note (latest publish_at: +1 day)

		// Verify published status
		require.True(t, notes[0].PublishedAt.Valid)  // path2 is published
		require.False(t, notes[1].PublishedAt.Valid) // path1 is not published
		require.False(t, notes[2].PublishedAt.Valid) // path3 is not published
	})

	t.Run("ListScheduledTelegramPublishNoteIDs_critical_bug_test", func(t *testing.T) {
		noteIDs, err := queries.ListSheduledTelegarmPublishNoteIDs(ctx)
		require.NoError(t, err)

		// Should ONLY return path1 (publish_at has passed - ready to send)
		// Should NOT return path3 (scheduled for future - not ready yet)
		require.Len(t, noteIDs, 1)
		require.Equal(t, path1, noteIDs[0])

		// CRITICAL: Verify path3 (future post) is NOT in the scheduled list
		for _, id := range noteIDs {
			require.NotEqual(t, path3, id, "Note scheduled for future should NOT be sent yet")
		}
	})
}
