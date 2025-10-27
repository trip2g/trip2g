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

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	Logger() logger.Logger

	ListTelegramPublishSentMessagesByNotePathID(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error)
	ResetTelegramPublishNote(ctx context.Context, notePathID int64) error
	DeleteTelegramPublishSentMessagesByNotePathID(ctx context.Context, notePathID int64) error
	SendTelegramRequest(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error
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

	logger := logger.WithPrefix(env.Logger(), "resettelegrampublishnote")

	publishNote, err := env.GetTelegramPublishNoteByNotePathID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &model.ErrorPayload{Message: "Telegram publish note not found"}, nil
		}
		return nil, fmt.Errorf("failed to get telegram publish note: %w", err)
	}

	// Get all sent messages for this note before deleting them
	sentMessages, err := env.ListTelegramPublishSentMessagesByNotePathID(ctx, publishNote.NotePathID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sent messages: %w", err)
	}

	// Reset the publish note in database
	err = env.ResetTelegramPublishNote(ctx, publishNote.NotePathID)
	if err != nil {
		return nil, fmt.Errorf("failed to reset telegram publish note: %w", err)
	}

	// Delete sent message records
	err = env.DeleteTelegramPublishSentMessagesByNotePathID(ctx, publishNote.NotePathID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete sent message records: %w", err)
	}

	// Get the updated publish note to return
	updatedNote, err := env.GetTelegramPublishNoteByNotePathID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated telegram publish note: %w", err)
	}

	// Delete messages from Telegram after all database operations are successful
	for _, sentMsg := range sentMessages {
		deleteMsg := tgbotapi.NewDeleteMessage(sentMsg.TelegramID, int(sentMsg.MessageID))
		err = env.SendTelegramRequest(ctx, sentMsg.ChatID, deleteMsg)
		if err != nil {
			logger.Error("failed to delete message", "chat_id", sentMsg.ChatID, "message_id", sentMsg.MessageID, "error", err)
		}
	}

	payload := model.ResetTelegramPublishNotePayload{
		PublishNote: &updatedNote,
	}

	return &payload, nil
}
