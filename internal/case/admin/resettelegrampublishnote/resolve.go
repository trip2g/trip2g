package resettelegrampublishnote

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/usertoken"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg resettelegrampublishnote_test . Env

type SentMessageRow = db.ListTelegramPublishSentMessagesByNotePathIDRow
type SentAccountMessageRow = db.ListTelegramPublishSentAccountMessagesByNotePathIDRow

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	Logger() logger.Logger

	// Bot messages
	ListTelegramPublishSentMessagesByNotePathID(ctx context.Context, notePathID int64) ([]SentMessageRow, error)
	DeleteTelegramPublishSentMessagesByNotePathID(ctx context.Context, notePathID int64) error
	SendTelegramRequest(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error

	// Account messages
	ListTelegramPublishSentAccountMessagesByNotePathID(ctx context.Context, notePathID int64) ([]SentAccountMessageRow, error)
	DeleteTelegramPublishSentAccountMessagesByNotePathID(ctx context.Context, notePathID int64) error
	GetTelegramAccountByID(ctx context.Context, id int64) (db.TelegramAccount, error)
	DeleteTelegramAccountMessage(ctx context.Context, account db.TelegramAccount, chatID, messageID int64) error

	// Common
	ResetTelegramPublishNote(ctx context.Context, notePathID int64) error
	GetTelegramPublishNoteByNotePathID(ctx context.Context, notePathID int64) (db.TelegramPublishNote, error)
}

type Input = model.ResetTelegramPublishNoteInput
type Payload = model.ResetTelegramPublishNoteOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return nil // No validation needed for simple int64 ID
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	log := logger.WithPrefix(env.Logger(), "resettelegrampublishnote:")

	publishNote, err := env.GetTelegramPublishNoteByNotePathID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &model.ErrorPayload{Message: "Telegram publish note not found"}, nil
		}
		return nil, fmt.Errorf("failed to get telegram publish note: %w", err)
	}

	// Get all bot sent messages for this note before deleting them
	botSentMessages, err := env.ListTelegramPublishSentMessagesByNotePathID(ctx, publishNote.NotePathID)
	if err != nil {
		return nil, fmt.Errorf("failed to list bot sent messages: %w", err)
	}

	// Get all account sent messages for this note before deleting them
	accountSentMessages, err := env.ListTelegramPublishSentAccountMessagesByNotePathID(ctx, publishNote.NotePathID)
	if err != nil {
		return nil, fmt.Errorf("failed to list account sent messages: %w", err)
	}

	// Reset the publish note in database
	err = env.ResetTelegramPublishNote(ctx, publishNote.NotePathID)
	if err != nil {
		return nil, fmt.Errorf("failed to reset telegram publish note: %w", err)
	}

	// Delete bot sent message records
	err = env.DeleteTelegramPublishSentMessagesByNotePathID(ctx, publishNote.NotePathID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete bot sent message records: %w", err)
	}

	// Delete account sent message records
	err = env.DeleteTelegramPublishSentAccountMessagesByNotePathID(ctx, publishNote.NotePathID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete account sent message records: %w", err)
	}

	// Get the updated publish note to return
	updatedNote, err := env.GetTelegramPublishNoteByNotePathID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated telegram publish note: %w", err)
	}

	// Delete bot messages from Telegram
	for _, sentMsg := range botSentMessages {
		deleteMsg := tgbotapi.NewDeleteMessage(sentMsg.TelegramID, int(sentMsg.MessageID))
		deleteErr := env.SendTelegramRequest(ctx, sentMsg.ChatID, deleteMsg)
		if deleteErr != nil {
			log.Error("failed to delete bot message", "chat_id", sentMsg.ChatID, "message_id", sentMsg.MessageID, "error", deleteErr)
		}
	}

	// Delete account messages from Telegram
	for _, sentMsg := range accountSentMessages {
		account, accountErr := env.GetTelegramAccountByID(ctx, sentMsg.AccountID)
		if accountErr != nil {
			log.Error("failed to get account for message deletion", "account_id", sentMsg.AccountID, "error", accountErr)
			continue
		}

		deleteErr := env.DeleteTelegramAccountMessage(ctx, account, sentMsg.TelegramChatID, sentMsg.MessageID)
		if deleteErr != nil {
			log.Error("failed to delete account message",
				"account_id", sentMsg.AccountID,
				"chat_id", sentMsg.TelegramChatID,
				"message_id", sentMsg.MessageID,
				"error", deleteErr,
			)
		}
	}

	payload := model.ResetTelegramPublishNotePayload{
		PublishNote: &updatedNote,
	}

	return &payload, nil
}
