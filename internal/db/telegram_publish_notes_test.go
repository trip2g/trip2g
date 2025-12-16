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

	// Create necessary telegram infrastructure for query to work
	var adminUserID, botID, chatID, tagID int64
	var err error
	var notes []db.TelegramPublishNote
	var noteIDs []int64

	err = conn.QueryRow(`
		insert into users (email, created_via)
		values ('admin@test.com', 'test')
		returning id
	`).Scan(&adminUserID)
	require.NoError(t, err)

	mustExec(t, conn, `insert into admins (user_id) values (?)`, adminUserID)

	err = conn.QueryRow(`
		insert into tg_bots (name, token, created_by)
		values ('test_bot', 'test_token', ?)
		returning id
	`, adminUserID).Scan(&botID)
	require.NoError(t, err)

	err = conn.QueryRow(`
		insert into tg_bot_chats (bot_id, telegram_id, chat_type, chat_title)
		values (?, -1001234567890, 'supergroup', 'Test Chat')
		returning id
	`, botID).Scan(&chatID)
	require.NoError(t, err)

	// Create a tag for the chat
	err = conn.QueryRow(`
		insert into telegram_publish_tags (label)
		values ('test_tag')
		returning id
	`).Scan(&tagID)
	require.NoError(t, err)

	// Associate the chat with the tag
	mustExec(t, conn, `
		insert into telegram_publish_chats (chat_id, tag_id, created_by)
		values (?, ?, ?)
	`, chatID, tagID, adminUserID)

	// Insert telegram publish notes FIRST (required by foreign key in note_tags)
	// 1. Scheduled for PAST (should appear in scheduled - ready to send)
	mustExec(t, conn, `
		insert into telegram_publish_notes (note_path_id, created_at, publish_at, published_at)
		values (?, datetime('now', '-2 hours'), datetime('now', '-1 hour'), null)
	`, path1)

	// Tag path1 with the test tag (so it will be included in scheduled list)
	mustExec(t, conn, `
		insert into telegram_publish_note_tags (note_path_id, tag_id)
		values (?, ?)
	`, path1, tagID)

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
		notes, err = queries.ListAllTelegramPublishNotes(ctx, params)
		require.NoError(t, err)

		// Should return path1 and path3 (unpublished)
		require.Len(t, notes, 2)

		// Verify specific notes returned (ordered by publish_at)
		require.Equal(t, path1, notes[0].NotePathID) // path1 has earlier publish_at (-1 hour)
		require.Equal(t, path3, notes[1].NotePathID) // path3 has later publish_at (+1 day)

		// Verify no published notes are included
		require.Nil(t, notes[0].PublishedAt)
		require.Nil(t, notes[1].PublishedAt)
	})

	t.Run("ListAllTelegramPublishNotes_show_all", func(t *testing.T) {
		params := db.ListAllTelegramPublishNotesParams{
			ShowScheduled: true,
			ShowSent:      true,
			ShowOutdated:  true,
		}
		notes, err = queries.ListAllTelegramPublishNotes(ctx, params)
		require.NoError(t, err)

		// Should include all notes (path1, path2, path3)
		require.Len(t, notes, 3)

		// Verify specific notes returned (ordered by publish_at)
		require.Equal(t, path2, notes[0].NotePathID) // published note (earliest publish_at: -1 day)
		require.Equal(t, path1, notes[1].NotePathID) // scheduled note (publish_at: -1 hour)
		require.Equal(t, path3, notes[2].NotePathID) // future note (latest publish_at: +1 day)

		// Verify published status
		require.NotNil(t, notes[0].PublishedAt) // path2 is published
		require.Nil(t, notes[1].PublishedAt)    // path1 is not published
		require.Nil(t, notes[2].PublishedAt)    // path3 is not published
	})

	t.Run("ListScheduledTelegramPublishNoteIDs_critical_bug_test", func(t *testing.T) {
		noteIDs, err = queries.ListSheduledTelegarmPublishNoteIDs(ctx)
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

func TestTelegramPublishSentMessagesPartialUniqueIndex(t *testing.T) {
	conn, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Create test data
	notePath1 := insertTestNotePath(t, conn, "/test/unique-index-1")
	notePath2 := insertTestNotePath(t, conn, "/test/unique-index-2")

	// Create admin user
	var adminUserID int64
	var botID int64
	var chatID int64
	var err error

	err = conn.QueryRow(`
		insert into users (email, created_via)
		values ('admin@test.com', 'test')
		returning id
	`).Scan(&adminUserID)
	require.NoError(t, err)

	_, err = conn.Exec(`
		insert into admins (user_id)
		values (?)
	`, adminUserID)
	require.NoError(t, err)

	// Create telegram bot
	err = conn.QueryRow(`
		insert into tg_bots (name, token, created_by)
		values ('test_bot', 'test_token', ?)
		returning id
	`, adminUserID).Scan(&botID)
	require.NoError(t, err)

	// Create telegram chat
	err = conn.QueryRow(`
		insert into tg_bot_chats (bot_id, telegram_id, chat_type, chat_title)
		values (?, 123456, 'supergroup', 'Test Chat')
		returning id
	`, botID).Scan(&chatID)
	require.NoError(t, err)

	t.Run("can_insert_only_one_scheduled_message_per_chat_and_note", func(t *testing.T) {
		// Insert first scheduled message (instant = 0)
		_, testErr := conn.Exec(`
			insert into telegram_publish_sent_messages
			(note_path_id, chat_id, message_id, instant, content_hash, content)
			values (?, ?, 1001, 0, 'hash1', 'content1')
		`, notePath1, chatID)
		require.NoError(t, testErr, "First scheduled message should insert successfully")

		// Try to insert duplicate scheduled message (instant = 0) - should FAIL
		_, testErr = conn.Exec(`
			insert into telegram_publish_sent_messages
			(note_path_id, chat_id, message_id, instant, content_hash, content)
			values (?, ?, 1002, 0, 'hash2', 'content2')
		`, notePath1, chatID)
		require.Error(t, testErr, "Duplicate scheduled message should fail")
		require.Contains(t, testErr.Error(), "UNIQUE constraint failed", "Error should mention unique constraint")
	})

	t.Run("can_insert_multiple_instant_messages_per_chat_and_note", func(t *testing.T) {
		// Insert first instant message (instant = 1)
		_, testErr := conn.Exec(`
			insert into telegram_publish_sent_messages
			(note_path_id, chat_id, message_id, instant, content_hash, content)
			values (?, ?, 2001, 1, 'instant_hash1', 'instant_content1')
		`, notePath2, chatID)
		require.NoError(t, testErr, "First instant message should insert successfully")

		// Insert second instant message (instant = 1) - should SUCCEED
		_, testErr = conn.Exec(`
			insert into telegram_publish_sent_messages
			(note_path_id, chat_id, message_id, instant, content_hash, content)
			values (?, ?, 2002, 1, 'instant_hash2', 'instant_content2')
		`, notePath2, chatID)
		require.NoError(t, testErr, "Multiple instant messages should be allowed")

		// Insert third instant message (instant = 1) - should SUCCEED
		_, testErr = conn.Exec(`
			insert into telegram_publish_sent_messages
			(note_path_id, chat_id, message_id, instant, content_hash, content)
			values (?, ?, 2003, 1, 'instant_hash3', 'instant_content3')
		`, notePath2, chatID)
		require.NoError(t, testErr, "Multiple instant messages should be allowed")

		// Verify all 3 instant messages exist
		var count int
		testErr = conn.QueryRow(`
			select count(*)
			from telegram_publish_sent_messages
			where note_path_id = ? and chat_id = ? and instant = 1
		`, notePath2, chatID).Scan(&count)
		require.NoError(t, testErr)
		require.Equal(t, 3, count, "Should have 3 instant messages")
	})

	t.Run("unique_constraint_only_applies_to_scheduled_messages", func(t *testing.T) {
		notePath3 := insertTestNotePath(t, conn, "/test/unique-index-3")

		// Insert scheduled message (instant = 0)
		_, testErr := conn.Exec(`
			insert into telegram_publish_sent_messages
			(note_path_id, chat_id, message_id, instant, content_hash, content)
			values (?, ?, 3001, 0, 'scheduled_hash', 'scheduled_content')
		`, notePath3, chatID)
		require.NoError(t, testErr)

		// Insert instant message with same chat_id and note_path_id - should SUCCEED
		_, testErr = conn.Exec(`
			insert into telegram_publish_sent_messages
			(note_path_id, chat_id, message_id, instant, content_hash, content)
			values (?, ?, 3002, 1, 'instant_hash', 'instant_content')
		`, notePath3, chatID)
		require.NoError(t, testErr, "Instant message should coexist with scheduled message")

		// Try to insert another scheduled message - should FAIL
		_, testErr = conn.Exec(`
			insert into telegram_publish_sent_messages
			(note_path_id, chat_id, message_id, instant, content_hash, content)
			values (?, ?, 3003, 0, 'scheduled_hash2', 'scheduled_content2')
		`, notePath3, chatID)
		require.Error(t, testErr, "Second scheduled message should fail")
	})
}
