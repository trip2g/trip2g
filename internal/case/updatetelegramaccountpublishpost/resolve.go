package updatetelegramaccountpublishpost

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/model"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg updatetelegramaccountpublishpost_test . Env

type Env interface {
	LatestNoteViews() *model.NoteViews
	ListTelegramPublishSentAccountMessagesByNotePathID(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentAccountMessagesByNotePathIDRow, error)
	ConvertNoteViewToTelegramPost(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error)
	EnqueueUpdateTelegramAccountMessage(ctx context.Context, params model.TelegramAccountUpdatePostParams) error
	GetTelegramAccountByID(ctx context.Context, id int64) (db.TelegramAccount, error)
}

func Resolve(ctx context.Context, env Env, notePathID int64) error {
	noteView := env.LatestNoteViews().GetByPathID(notePathID)
	if noteView == nil {
		return fmt.Errorf("note view not found for path ID %d", notePathID)
	}

	// Get all sent account messages for this note
	sentMessages, err := env.ListTelegramPublishSentAccountMessagesByNotePathID(ctx, notePathID)
	if err != nil {
		return fmt.Errorf("failed to get sent account messages for note: %w", err)
	}

	if len(sentMessages) == 0 {
		return nil
	}

	// Cache accounts to avoid repeated lookups
	accountCache := make(map[int64]db.TelegramAccount)

	// Enqueue update for each sent message
	for _, sentMsg := range sentMessages {
		source := model.TelegramPostSource{
			NoteView: noteView,
			ChatID:   0, // Not used for account publishing
			Instant:  false,
		}

		// Convert note to Telegram post
		post, convertErr := env.ConvertNoteViewToTelegramPost(ctx, source)
		if convertErr != nil {
			return fmt.Errorf("failed to convert note to telegram post: %w", convertErr)
		}

		// Calculate current content hash
		hash := sha256.Sum256([]byte(post.Content))
		currentHash := hex.EncodeToString(hash[:])

		// Skip update if content hasn't changed
		if currentHash == sentMsg.ContentHash {
			continue
		}

		// Get account (from cache or database)
		account, ok := accountCache[sentMsg.AccountID]
		if !ok {
			var accountErr error
			account, accountErr = env.GetTelegramAccountByID(ctx, sentMsg.AccountID)
			if accountErr != nil {
				return fmt.Errorf("failed to get account %d: %w", sentMsg.AccountID, accountErr)
			}
			accountCache[sentMsg.AccountID] = account
		}

		// Prepare update params
		params := model.TelegramAccountUpdatePostParams{
			TelegramAccountSendPostParams: model.TelegramAccountSendPostParams{
				NotePathID:        notePathID,
				AccountID:         sentMsg.AccountID,
				TelegramChatID:    sentMsg.TelegramChatID,
				SessionData:       account.SessionData,
				Post:              *post,
				Instant:           false,
				UpdateLinkedPosts: false,
			},
			MessageID: sentMsg.MessageID,
		}

		// Enqueue the update job
		enqueueErr := env.EnqueueUpdateTelegramAccountMessage(ctx, params)
		if enqueueErr != nil {
			return fmt.Errorf("failed to enqueue account update job for account %d, chat %d: %w",
				sentMsg.AccountID, sentMsg.TelegramChatID, enqueueErr)
		}
	}

	return nil
}
