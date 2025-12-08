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

type accountChat struct {
	AccountID      int64
	TelegramChatID int64
}

func Resolve(ctx context.Context, env Env, params model.SendTelegramPublishPostParams) error {
	log := logger.WithPrefix(env.Logger(), "sendtelegramaccountpublishpost:")
	nvs := env.LatestNoteViews()

	noteView := nvs.GetByPathID(params.NotePathID)
	if noteView == nil {
		return fmt.Errorf("note view not found for path ID %d", params.NotePathID)
	}

	// Get account chats that should receive this post
	accountChats, err := getAccountChats(ctx, env, params)
	if err != nil {
		return err
	}

	if len(accountChats) == 0 {
		if params.Instant {
			return nil
		}
		return fmt.Errorf("no account chats found for note path ID %d", params.NotePathID)
	}

	log.Info("sending via accounts",
		"note_path_id", params.NotePathID,
		"instant", params.Instant,
		"account_chat_count", len(accountChats),
	)

	// Send the post to each account chat
	for _, chat := range accountChats {
		sendErr := enqueuePostToChat(ctx, env, params, noteView, chat)
		if sendErr != nil {
			return sendErr
		}
	}

	// Mark as published for non-instant posts
	if !params.Instant {
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

func getAccountChats(ctx context.Context, env Env, params model.SendTelegramPublishPostParams) ([]accountChat, error) {
	var result []accountChat

	if params.Instant {
		chats, err := env.ListTelegramAccountInstantChatsByNotePathID(ctx, params.NotePathID)
		if err != nil {
			return nil, fmt.Errorf("failed to get instant account chats for note: %w", err)
		}
		for _, c := range chats {
			result = append(result, accountChat{
				AccountID:      c.AccountID,
				TelegramChatID: c.TelegramChatID,
			})
		}
	} else {
		chats, err := env.ListTelegramAccountChatsByNotePathID(ctx, params.NotePathID)
		if err != nil {
			return nil, fmt.Errorf("failed to get account chats for note: %w", err)
		}
		for _, c := range chats {
			result = append(result, accountChat{
				AccountID:      c.AccountID,
				TelegramChatID: c.TelegramChatID,
			})
		}
	}

	return result, nil
}

func enqueuePostToChat(ctx context.Context, env Env, params model.SendTelegramPublishPostParams, noteView *model.NoteView, chat accountChat) error {
	source := model.TelegramPostSource{
		NoteView:       noteView,
		ChatID:         0, // Not used for account publishing
		TelegramChatID: chat.TelegramChatID,
		Instant:        params.Instant,
	}

	post, convertErr := env.ConvertNoteViewToTelegramPost(ctx, source)
	if convertErr != nil {
		return fmt.Errorf("failed to convert note to telegram post: %w", convertErr)
	}

	sendParams := model.TelegramAccountSendPostParams{
		NotePathID:        params.NotePathID,
		AccountID:         chat.AccountID,
		TelegramChatID:    chat.TelegramChatID,
		Post:              *post,
		Instant:           params.Instant,
		UpdateLinkedPosts: params.UpdateLinkedPosts,
	}

	sendErr := env.EnqueueSendTelegramAccountMessage(ctx, sendParams)
	if sendErr != nil {
		return fmt.Errorf("failed to enqueue telegram account post for account %d, chat %d: %w",
			chat.AccountID, chat.TelegramChatID, sendErr)
	}

	return nil
}
