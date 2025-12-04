package sendtelegramaccountpublishpost

import (
	"context"
	"database/sql"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg sendtelegramaccountpublishpost_test . Env

type Env interface {
	// Database methods for getting note and chat information
	ListTelegramAccountChatsByNotePathID(ctx context.Context, notePathID int64) ([]db.ListTelegramAccountChatsByNotePathIDRow, error)
	ListTelegramAccountInstantChatsByNotePathID(ctx context.Context, notePathID int64) ([]db.ListTelegramAccountInstantChatsByNotePathIDRow, error)
	UpdateTelegramPublishNoteAsPublished(ctx context.Context, arg db.UpdateTelegramPublishNoteAsPublishedParams) error

	// Telegram account message queue
	EnqueueSendTelegramAccountMessage(ctx context.Context, params model.TelegramAccountSendPostParams) error

	ConvertNoteViewToTelegramPost(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error)

	// Content access methods
	LatestNoteViews() *model.NoteViews

	Logger() logger.Logger
}

func Resolve(ctx context.Context, env Env, params model.SendTelegramPublishPostParams) error {
	logger := logger.WithPrefix(env.Logger(), "sendtelegramaccountpublishpost:")
	nvs := env.LatestNoteViews()

	noteView := nvs.GetByPathID(params.NotePathID)
	if noteView == nil {
		return fmt.Errorf("note view not found for path ID %d", params.NotePathID)
	}

	// Get account chats that should receive this post
	var chats []db.ListTelegramAccountChatsByNotePathIDRow
	var instantChats []db.ListTelegramAccountInstantChatsByNotePathIDRow
	var err error

	if params.Instant {
		instantChats, err = env.ListTelegramAccountInstantChatsByNotePathID(ctx, params.NotePathID)
		if err != nil {
			return fmt.Errorf("failed to get instant account chats for note: %w", err)
		}
	} else {
		chats, err = env.ListTelegramAccountChatsByNotePathID(ctx, params.NotePathID)
		if err != nil {
			return fmt.Errorf("failed to get account chats for note: %w", err)
		}
	}

	// Combine chats into unified slice for processing
	type accountChat struct {
		AccountID      int64
		TelegramChatID int64
		SessionData    []byte
	}

	var accountChats []accountChat

	if params.Instant {
		for _, c := range instantChats {
			accountChats = append(accountChats, accountChat{
				AccountID:      c.AccountID,
				TelegramChatID: c.TelegramChatID,
				SessionData:    c.SessionData,
			})
		}
	} else {
		for _, c := range chats {
			accountChats = append(accountChats, accountChat{
				AccountID:      c.AccountID,
				TelegramChatID: c.TelegramChatID,
				SessionData:    c.SessionData,
			})
		}
	}

	if len(accountChats) == 0 {
		if params.Instant {
			// For instant posts, no chats is fine
			return nil
		}
		return fmt.Errorf("no account chats found for note path ID %d", params.NotePathID)
	}

	logger.Info("sending via accounts",
		"note_path_id", params.NotePathID,
		"instant", params.Instant,
		"account_chat_count", len(accountChats),
	)

	// Send the post to each account chat
	for _, chat := range accountChats {
		source := model.TelegramPostSource{
			NoteView: noteView,
			ChatID:   0, // Not used for account publishing
			Instant:  params.Instant,
		}

		// Convert note to Telegram post
		post, convertErr := env.ConvertNoteViewToTelegramPost(ctx, source)
		if convertErr != nil {
			return fmt.Errorf("failed to convert note to telegram post: %w", convertErr)
		}

		// Prepare send params
		sendParams := model.TelegramAccountSendPostParams{
			NotePathID:        params.NotePathID,
			AccountID:         chat.AccountID,
			TelegramChatID:    chat.TelegramChatID,
			SessionData:       chat.SessionData,
			Post:              *post,
			Instant:           params.Instant,
			UpdateLinkedPosts: params.UpdateLinkedPosts,
		}

		// Enqueue the message to be sent via telegram account queue
		sendErr := env.EnqueueSendTelegramAccountMessage(ctx, sendParams)
		if sendErr != nil {
			return fmt.Errorf("failed to enqueue telegram account post for account %d, chat %d: %w",
				chat.AccountID, chat.TelegramChatID, sendErr)
		}
	}

	// Mark as published is handled by the bot pipeline, not here
	// This avoids double-updating the published state
	if !params.Instant && len(chats) > 0 {
		// Only mark as published if this is the only pipeline (no bot chats)
		// Actually, let the bot pipeline handle this since both share the same telegram_publish_notes table
		updateParams := db.UpdateTelegramPublishNoteAsPublishedParams{
			PublishedVersionID: sql.NullInt64{Int64: noteView.VersionID, Valid: true},
			NotePathID:         params.NotePathID,
		}

		err = env.UpdateTelegramPublishNoteAsPublished(ctx, updateParams)
		if err != nil {
			return fmt.Errorf("failed to mark note as published: %w", err)
		}
	}

	return nil
}
