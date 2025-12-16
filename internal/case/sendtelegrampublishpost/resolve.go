package sendtelegrampublishpost

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg sendtelegrampublishpost_test . Env

type Env interface {
	// Database methods for getting note and chat information
	ListTgBotChatsByTelegramPublishNotePathID(ctx context.Context, notePathID int64) ([]db.TgBotChat, error)
	ListTgBotInstantChatsByTelegramPublishNotePathID(ctx context.Context, notePathID int64) ([]db.TgBotChat, error)
	UpdateTelegramPublishNoteAsPublished(ctx context.Context, arg db.UpdateTelegramPublishNoteAsPublishedParams) error

	// Telegram message queue
	EnqueueSendTelegramMessage(ctx context.Context, params model.TelegramSendPostParams) error

	ConvertNoteViewToTelegramPost(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error)

	// Content access methods
	LatestNoteViews() *model.NoteViews

	Logger() logger.Logger
}

func Resolve(ctx context.Context, env Env, params model.SendTelegramPublishPostParams) error {
	logger := logger.WithPrefix(env.Logger(), "sendtelegrampublishpost:")
	nvs := env.LatestNoteViews()

	noteView := nvs.GetByPathID(params.NotePathID)
	if noteView == nil {
		return fmt.Errorf("note view not found for path ID %d", params.NotePathID)
	}

	// Get chat IDs that should receive this post
	var chats []db.TgBotChat
	var err error

	if params.Instant {
		chats, err = env.ListTgBotInstantChatsByTelegramPublishNotePathID(ctx, params.NotePathID)
	} else {
		chats, err = env.ListTgBotChatsByTelegramPublishNotePathID(ctx, params.NotePathID)
	}

	if err != nil {
		return fmt.Errorf("failed to get chat IDs for note: %w", err)
	}

	if len(chats) == 0 {
		if params.Instant {
			return nil
		}

		return fmt.Errorf("no chat IDs found for note path ID %d", params.NotePathID)
	}

	logger.Info("sending", "note_path_id", params.NotePathID, "instant", params.Instant, "chat_count", len(chats))

	// Send the post to each chat
	for _, chat := range chats {
		source := model.TelegramPostSource{
			NoteView:       noteView,
			ChatID:         chat.ID,
			TelegramChatID: chat.TelegramID,
			Instant:        params.Instant,
		}

		// Convert note to Telegram post
		post, convertErr := env.ConvertNoteViewToTelegramPost(ctx, source)
		if convertErr != nil {
			return fmt.Errorf("failed to convert note to telegram post: %w", convertErr)
		}

		// if len(post.Warnings) > 0 {
		// 	return fmt.Errorf("conversion produced warnings: %v", post.Warnings)
		// }

		// Prepare send params
		sendParams := model.TelegramSendPostParams{
			NotePathID:        params.NotePathID,
			DBChatID:          chat.ID,
			TelegramChatID:    chat.TelegramID,
			Post:              *post,
			Instant:           params.Instant,
			UpdateLinkedPosts: params.UpdateLinkedPosts,
		}

		// Enqueue the message to be sent via telegram queue
		sendErr := env.EnqueueSendTelegramMessage(ctx, sendParams)
		if sendErr != nil {
			return fmt.Errorf("failed to enqueue telegram post for chat %d: %w", chat.ID, sendErr)
		}
	}

	if !params.Instant {
		// Mark the note as published
		updateParams := db.UpdateTelegramPublishNoteAsPublishedParams{
			PublishedVersionID: &noteView.VersionID,
			NotePathID:         params.NotePathID,
		}

		err = env.UpdateTelegramPublishNoteAsPublished(ctx, updateParams)
		if err != nil {
			return fmt.Errorf("failed to mark note as published: %w", err)
		}
	}

	return nil
}
