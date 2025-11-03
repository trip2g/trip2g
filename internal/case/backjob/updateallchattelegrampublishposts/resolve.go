package updateallchattelegrampublishposts

import (
	"context"
	"fmt"
	"trip2g/internal/case/updatetelegrampublishpost"
	"trip2g/internal/db"
	"trip2g/internal/logger"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg updateallchattelegrampublishposts_test . Env

type Params struct {
	ChatID int64
}

type Env interface {
	Logger() logger.Logger
	ListTelegramPublishSentMessagesByChatID(ctx context.Context, chatID int64) ([]db.ListTelegramPublishSentMessagesByChatIDRow, error)
	updatetelegrampublishpost.Env
}

func Resolve(ctx context.Context, env Env, params Params) error {
	logger := logger.WithPrefix(env.Logger(), "updateallchattelegrampublishposts:")

	// Get all sent messages for this chat
	sentMessages, err := env.ListTelegramPublishSentMessagesByChatID(ctx, params.ChatID)
	if err != nil {
		return fmt.Errorf("failed to list sent messages for chat %d: %w", params.ChatID, err)
	}

	if len(sentMessages) == 0 {
		logger.Debug("no sent messages found for chat", "chat_id", params.ChatID)
		return nil
	}

	// Extract unique note_path_ids
	notePathIDs := make(map[int64]bool)
	for _, msg := range sentMessages {
		notePathIDs[msg.NotePathID] = true
	}

	logger.Info("processing chat updates", "chat_id", params.ChatID, "unique_notes", len(notePathIDs), "total_messages", len(sentMessages))

	// For each unique note, call updatetelegrampublishpost.Resolve
	for notePathID := range notePathIDs {
		err = updatetelegrampublishpost.Resolve(ctx, env, notePathID)
		if err != nil {
			logger.Error("failed to update telegram publish post", "note_path_id", notePathID, "chat_id", params.ChatID, "error", err)
			return fmt.Errorf("failed to update telegram publish post for note %d: %w", notePathID, err)
		}
	}

	logger.Info("completed chat updates", "chat_id", params.ChatID, "notes_processed", len(notePathIDs))
	return nil
}
