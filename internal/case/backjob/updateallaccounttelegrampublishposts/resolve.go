package updateallaccounttelegrampublishposts

import (
	"context"
	"fmt"
	"trip2g/internal/case/updatetelegramaccountpublishpost"
	"trip2g/internal/db"
	"trip2g/internal/logger"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg updateallaccounttelegrampublishposts_test . Env

type Params struct {
	AccountID int64
}

type Env interface {
	Logger() logger.Logger
	ListTelegramPublishSentAccountMessagesByAccountID(ctx context.Context, accountID int64) ([]db.ListTelegramPublishSentAccountMessagesByAccountIDRow, error)
	updatetelegramaccountpublishpost.Env
}

func Resolve(ctx context.Context, env Env, params Params) error {
	logger := logger.WithPrefix(env.Logger(), "updateallaccounttelegrampublishposts:")

	// Get all sent messages for this account
	sentMessages, err := env.ListTelegramPublishSentAccountMessagesByAccountID(ctx, params.AccountID)
	if err != nil {
		return fmt.Errorf("failed to list sent messages for account %d: %w", params.AccountID, err)
	}

	if len(sentMessages) == 0 {
		logger.Debug("no sent messages found for account", "account_id", params.AccountID)
		return nil
	}

	// Extract unique note_path_ids
	notePathIDs := make(map[int64]bool)
	for _, msg := range sentMessages {
		notePathIDs[msg.NotePathID] = true
	}

	logger.Info("processing account updates", "account_id", params.AccountID, "unique_notes", len(notePathIDs), "total_messages", len(sentMessages))

	// For each unique note, call updatetelegramaccountpublishpost.Resolve
	for notePathID := range notePathIDs {
		err = updatetelegramaccountpublishpost.Resolve(ctx, env, notePathID)
		if err != nil {
			logger.Error("failed to update telegram account publish post", "note_path_id", notePathID, "account_id", params.AccountID, "error", err)
			return fmt.Errorf("failed to update telegram account publish post for note %d: %w", notePathID, err)
		}
	}

	logger.Info("completed account updates", "account_id", params.AccountID, "notes_processed", len(notePathIDs))
	return nil
}
