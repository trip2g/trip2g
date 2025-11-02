package sendtelegrampublishpost

import (
	"context"
	"database/sql"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/model"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg sendtelegrampublishpost_test . Env

type Env interface {
	// Database methods for getting note and chat information
	ListTgBotChatsByTelegramPublishNotePathID(ctx context.Context, notePathID int64) ([]db.TgBotChat, error)
	ListTgBotInstantChatsByTelegramPublishNotePathID(ctx context.Context, notePathID int64) ([]db.TgBotChat, error)
	UpdateTelegramPublishNoteAsPublished(ctx context.Context, arg db.UpdateTelegramPublishNoteAsPublishedParams) error

	// Telegram post queue
	EnqueueSendTelegramPost(ctx context.Context, params model.TelegramSendPostParams) error

	ConvertNoteViewToTelegramPost(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error)

	// Content access methods
	LatestNoteViews() *model.NoteViews
}

func Resolve(ctx context.Context, env Env, notePathID int64, instant bool) error {
	nvs := env.LatestNoteViews()

	noteView := nvs.GetByPathID(notePathID)
	if noteView == nil {
		return fmt.Errorf("note view not found for path ID %d", notePathID)
	}

	// Get chat IDs that should receive this post
	var chats []db.TgBotChat
	var err error

	if instant {
		chats, err = env.ListTgBotInstantChatsByTelegramPublishNotePathID(ctx, notePathID)
	} else {
		chats, err = env.ListTgBotChatsByTelegramPublishNotePathID(ctx, notePathID)
	}

	if err != nil {
		return fmt.Errorf("failed to get chat IDs for note: %w", err)
	}

	if len(chats) == 0 {
		if instant {
			return nil
		}

		return fmt.Errorf("no chat IDs found for note path ID %d", notePathID)
	}

	// Send the post to each chat
	for _, chat := range chats {
		source := model.TelegramPostSource{
			NoteView: noteView,
			ChatID:   chat.ID,
			Instant:  instant,
		}

		// Convert note to Telegram post
		post, convertErr := env.ConvertNoteViewToTelegramPost(ctx, source)
		if convertErr != nil {
			return fmt.Errorf("failed to convert note to telegram post: %w", convertErr)
		}

		if len(post.Warnings) > 0 {
			return fmt.Errorf("conversion produced warnings: %v", post.Warnings)
		}

		// Prepare send params
		params := model.TelegramSendPostParams{
			NotePathID:        notePathID,
			DBChatID:          chat.ID,
			TelegramChatID:    chat.TelegramID,
			Post:              *post,
			Instant:           instant,
			UpdateLinkedPosts: !instant,
		}

		// Enqueue the post to be sent via telegram queue
		sendErr := env.EnqueueSendTelegramPost(ctx, params)
		if sendErr != nil {
			return fmt.Errorf("failed to enqueue telegram post for chat %d: %w", chat.ID, sendErr)
		}
	}

	if !instant {
		// Mark the note as published
		updateParams := db.UpdateTelegramPublishNoteAsPublishedParams{
			PublishedVersionID: sql.NullInt64{Int64: noteView.VersionID, Valid: true},
			NotePathID:         notePathID,
		}

		err = env.UpdateTelegramPublishNoteAsPublished(ctx, updateParams)
		if err != nil {
			return fmt.Errorf("failed to mark note as published: %w", err)
		}
	}

	return nil
}
