package updatetelegramaccountmessage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/telegram"
	"trip2g/internal/tgtd"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg updatetelegramaccountmessage_test . Env

type Env interface {
	Logger() logger.Logger
	GetTelegramPublishSentAccountMessageContentHash(ctx context.Context, arg db.GetTelegramPublishSentAccountMessageContentHashParams) (string, error)
	GetTelegramPublishSentAccountMessagePostType(ctx context.Context, arg db.GetTelegramPublishSentAccountMessagePostTypeParams) (string, error)
	UpdateTelegramPublishSentAccountMessageContent(ctx context.Context, arg db.UpdateTelegramPublishSentAccountMessageContentParams) error
	GetTelegramAccountByID(ctx context.Context, id int64) (db.TelegramAccount, error)
}

func Resolve(ctx context.Context, env Env, params model.TelegramAccountUpdatePostParams) error {
	jobTimeout := time.Minute

	jobCtx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()

	err := resolve1(jobCtx, env, params)
	if err != nil {
		shouldRetry, delay := telegram.HandleRateLimit(err)
		if shouldRetry {
			env.Logger().Info("telegram rate limit hit, retrying after delay",
				"delay", delay,
				"job", JobID,
			)
			time.Sleep(delay)
			err = Resolve(ctx, env, params)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func resolve1(ctx context.Context, env Env, params model.TelegramAccountUpdatePostParams) error {
	logger := logger.WithPrefix(env.Logger(), "backjob/updatetelegramaccountmessage:")
	post := params.Post

	// Get current post type from database
	currentPostType, err := env.GetTelegramPublishSentAccountMessagePostType(ctx, db.GetTelegramPublishSentAccountMessagePostTypeParams{
		NotePathID:     params.NotePathID,
		AccountID:      params.AccountID,
		TelegramChatID: params.TelegramChatID,
		MessageID:      params.MessageID,
	})
	if err != nil {
		return fmt.Errorf("failed to get current post type: %w", err)
	}

	// Determine new post type
	mediaCount := len(post.Media)
	var newPostType string
	switch mediaCount {
	case 0:
		newPostType = db.TelegramPublishSentMessagePostTypeText
	case 1:
		newPostType = db.TelegramPublishSentMessagePostTypePhoto
	default:
		newPostType = db.TelegramPublishSentMessagePostTypeMediaGroup
	}

	// Check if we can update this post type
	if currentPostType != db.TelegramPublishSentMessagePostTypeText && currentPostType != db.TelegramPublishSentMessagePostTypePhoto {
		logger.Warn("only text and photo messages can be updated via account",
			"current_type", currentPostType,
			"note_path_id", params.NotePathID,
		)
		return nil
	}

	// Can't convert to media_group (need to delete and resend)
	if newPostType == db.TelegramPublishSentMessagePostTypeMediaGroup {
		logger.Warn("cannot convert to media_group, would need delete and resend",
			"current_type", currentPostType,
			"new_type", newPostType,
			"note_path_id", params.NotePathID,
		)
		return nil
	}

	// Truncate content to telegram limits (media posts have lower limit)
	hasMedia := mediaCount > 0
	content := telegram.TruncateContent(post.Content, hasMedia)

	// Calculate content hash for new content
	hash := sha256.Sum256([]byte(content))
	newContentHash := hex.EncodeToString(hash[:])

	// Get current content hash from database
	currentContentHash, err := env.GetTelegramPublishSentAccountMessageContentHash(ctx, db.GetTelegramPublishSentAccountMessageContentHashParams{
		NotePathID:     params.NotePathID,
		AccountID:      params.AccountID,
		TelegramChatID: params.TelegramChatID,
		MessageID:      params.MessageID,
	})
	if err != nil {
		return fmt.Errorf("failed to get current content hash: %w", err)
	}

	// Skip update if content hasn't changed
	if currentContentHash == newContentHash {
		logger.Info("skip, content unchanged",
			"note_path_id", params.NotePathID,
			"account_id", params.AccountID,
			"telegram_chat_id", params.TelegramChatID,
			"message_id", params.MessageID,
		)
		return nil
	}

	// Get account for API credentials
	account, err := env.GetTelegramAccountByID(ctx, params.AccountID)
	if err != nil {
		return fmt.Errorf("failed to get telegram account: %w", err)
	}

	// Create tgtd client
	client := tgtd.NewClient(env, int(account.ApiID), account.ApiHash)

	logger.Info("updating message",
		"note_path_id", params.NotePathID,
		"account_id", params.AccountID,
		"telegram_chat_id", params.TelegramChatID,
		"message_id", params.MessageID,
		"current_type", currentPostType,
		"new_type", newPostType,
		"content_preview", content[:min(100, len(content))],
	)

	var editErr error

	// Determine which edit method to use based on post types
	if currentPostType == db.TelegramPublishSentMessagePostTypeText && newPostType == db.TelegramPublishSentMessagePostTypePhoto {
		// Add photo to existing text message
		editErr = client.EditMessageWithPhoto(ctx, account.SessionData, tgtd.EditMessageWithPhotoParams{
			ChatID:    params.TelegramChatID,
			MessageID: params.MessageID,
			PhotoURL:  post.Media[0],
			Caption:   content,
		})
	} else {
		// Regular text edit
		editErr = client.EditMessage(ctx, account.SessionData, tgtd.EditMessageParams{
			ChatID:    params.TelegramChatID,
			MessageID: params.MessageID,
			Message:   content,
		})
	}

	if editErr != nil {
		logger.Debug("edit error",
			"error", editErr.Error(),
			"note_path_id", params.NotePathID,
			"account_id", params.AccountID,
			"telegram_chat_id", params.TelegramChatID,
			"message_id", params.MessageID,
		)

		// If Telegram says content is the same, it's not an error
		if !strings.Contains(editErr.Error(), "MESSAGE_NOT_MODIFIED") {
			return fmt.Errorf("failed to edit telegram message: %w", editErr)
		}

		logger.Info("already up-to-date",
			"note_path_id", params.NotePathID,
			"account_id", params.AccountID,
			"telegram_chat_id", params.TelegramChatID,
			"message_id", params.MessageID,
		)
	} else {
		logger.Debug("updated",
			"note_path_id", params.NotePathID,
			"account_id", params.AccountID,
			"telegram_chat_id", params.TelegramChatID,
			"message_id", params.MessageID,
			"from_type", currentPostType,
			"to_type", newPostType,
		)
	}

	// Update the database with new content hash and post type
	updateParams := db.UpdateTelegramPublishSentAccountMessageContentParams{
		ContentHash:    newContentHash,
		Content:        content,
		PostType:       newPostType,
		NotePathID:     params.NotePathID,
		AccountID:      params.AccountID,
		TelegramChatID: params.TelegramChatID,
		MessageID:      params.MessageID,
	}

	err = env.UpdateTelegramPublishSentAccountMessageContent(ctx, updateParams)
	if err != nil {
		return fmt.Errorf("failed to update sent message content in DB: %w", err)
	}

	return nil
}
