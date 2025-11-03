package updatetelegrampublishpost

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/model"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg updatetelegrampublishpost_test . Env

type Env interface {
	LatestNoteViews() *model.NoteViews
	ListTelegramPublishSentMessagesByNotePathID(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error)
	ConvertNoteViewToTelegramPost(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error)
	QueueUpdateTelegramPost(ctx context.Context, params model.TelegramUpdatePostParams) error
}

func Resolve(ctx context.Context, env Env, notePathID int64) error {
	noteView := env.LatestNoteViews().GetByPathID(notePathID)
	if noteView == nil {
		return fmt.Errorf("note view not found for path ID %d", notePathID)
	}

	// Get all sent messages for this note
	sentMessages, err := env.ListTelegramPublishSentMessagesByNotePathID(ctx, notePathID)
	if err != nil {
		return fmt.Errorf("failed to get sent messages for note: %w", err)
	}

	if len(sentMessages) == 0 {
		return nil
	}

	// Enqueue update for each sent message
	for _, sentMsg := range sentMessages {
		source := model.TelegramPostSource{
			NoteView: noteView,
			ChatID:   sentMsg.ChatID,
			Instant:  false, // Updates are always non-instant
		}

		// Convert note to Telegram post
		post, convertErr := env.ConvertNoteViewToTelegramPost(ctx, source)
		if convertErr != nil {
			return fmt.Errorf("failed to convert note to telegram post: %w", convertErr)
		}

		if len(post.Warnings) > 0 {
			return fmt.Errorf("conversion produced warnings: %v", post.Warnings)
		}

		// Calculate current content hash
		hash := sha256.Sum256([]byte(post.Content))
		currentHash := hex.EncodeToString(hash[:])

		// Skip update if content hasn't changed
		if currentHash == sentMsg.ContentHash {
			// TODO: add logging here
			continue
		}

		// Prepare update params
		params := model.TelegramUpdatePostParams{
			TelegramSendPostParams: model.TelegramSendPostParams{
				NotePathID:        notePathID,
				DBChatID:          sentMsg.ChatID,
				TelegramChatID:    sentMsg.TelegramID,
				Post:              *post,
				Instant:           false,
				UpdateLinkedPosts: false,
			},
			MessageID: sentMsg.MessageID,
		}

		// Enqueue the update job
		enqueueErr := env.QueueUpdateTelegramPost(ctx, params)
		if enqueueErr != nil {
			return fmt.Errorf("failed to enqueue update job for chat %d: %w", sentMsg.ChatID, enqueueErr)
		}
	}

	return nil
}
