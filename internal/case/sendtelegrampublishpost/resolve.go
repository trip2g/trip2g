package sendtelegrampublishpost

import (
	"context"
	"database/sql"
	"fmt"
	"trip2g/internal/case/convertnoteviewtotgpost"
	"trip2g/internal/db"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Env interface {
	// Database methods for getting note and chat information
	ListTgBotChatsByTelegramPublishNotePathID(ctx context.Context, notePathID int64) ([]db.TgBotChat, error)
	UpdateTelegramPublishNoteAsPublished(ctx context.Context, arg db.UpdateTelegramPublishNoteAsPublishedParams) error

	// Telegram bot methods for sending messages
	SendTelegramMessage(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error

	// Content access methods
	LatestNoteViews() *model.NoteViews
}

func Resolve(ctx context.Context, env Env, notePathID int64) error {
	noteView := env.LatestNoteViews().GetByPathID(notePathID)
	if noteView == nil {
		return fmt.Errorf("note view not found for path ID %d", notePathID)
	}

	// Convert note to Telegram post
	post, err := convertnoteviewtotgpost.Resolve(ctx, struct{}{}, noteView)
	if err != nil {
		return fmt.Errorf("failed to convert note to telegram post: %w", err)
	}

	if len(post.Warnings) > 0 {
		return fmt.Errorf("conversion produced warnings: %v", post.Warnings)
	}

	// Get chat IDs that should receive this post
	chats, err := env.ListTgBotChatsByTelegramPublishNotePathID(ctx, notePathID)
	if err != nil {
		return fmt.Errorf("failed to get chat IDs for note: %w", err)
	}

	if len(chats) == 0 {
		return fmt.Errorf("no chat IDs found for note path ID %d", notePathID)
	}

	// Send the post to each chat
	for _, chat := range chats {
		msg := tgbotapi.NewMessage(chat.TelegramID, post.Content)
		msg.ParseMode = "HTML"

		err = env.SendTelegramMessage(ctx, chat.ID, msg)
		if err != nil {
			return fmt.Errorf("failed to send telegram message to chat %d: %w", chat.ID, err)
		}
	}

	// Mark the note as published
	updateParams := db.UpdateTelegramPublishNoteAsPublishedParams{
		PublishedVersionID: sql.NullInt64{Int64: noteView.VersionID, Valid: true},
		NotePathID:         notePathID,
	}

	err = env.UpdateTelegramPublishNoteAsPublished(ctx, updateParams)
	if err != nil {
		return fmt.Errorf("failed to mark note as published: %w", err)
	}

	return nil
}
