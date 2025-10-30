package updatetelegrampublishpost

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"trip2g/internal/db"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg updatetelegrampublishpost_test . Env

type Env interface {
	// Database methods for getting sent messages
	ListTelegramPublishSentMessagesByNotePathID(ctx context.Context, notePathID int64) ([]db.ListTelegramPublishSentMessagesByNotePathIDRow, error)

	// Telegram bot methods for editing messages
	SendTelegramRequest(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error

	ConvertNoteViewToTelegramPost(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error)

	// Content access methods
	LatestNoteViews() *model.NoteViews
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

	// Get first image URL if exists
	var firstImageURL *string
	for path := range noteView.Assets {
		_, ok := noteView.AssetReplaces[path]
		if !ok {
			continue
		}

		firstImageURL = &noteView.AssetReplaces[path].URL
		break
	}

	// Update each sent message
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
			continue
		}

		var sendErr error

		// Edit the message
		if firstImageURL != nil {
			// Edit caption for photo message
			editMsg := tgbotapi.NewEditMessageCaption(sentMsg.TelegramID, int(sentMsg.MessageID), post.Content)
			editMsg.ParseMode = "HTML"

			sendErr = env.SendTelegramRequest(ctx, sentMsg.ChatID, editMsg)
		} else {
			// Edit text for text message
			editMsg := tgbotapi.NewEditMessageText(sentMsg.TelegramID, int(sentMsg.MessageID), post.Content)
			editMsg.ParseMode = "HTML"

			sendErr = env.SendTelegramRequest(ctx, sentMsg.ChatID, editMsg)
		}

		if sendErr != nil {
			if strings.Contains(sendErr.Error(), "are exactly the same as a current content") {
				// TODO: update sent message content hash in DB
				continue
			}

			return fmt.Errorf("failed to edit telegram message in chat %d: %w", sentMsg.ChatID, sendErr)
		}
	}

	return nil
}
