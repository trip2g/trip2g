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
	DecryptData(ciphertext []byte) ([]byte, error)
	TelegramCaptionLengthLimit(ctx context.Context, accountID *int64) int
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

//nolint:funlen // complex edit logic with multiple post types
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
	newPostType := db.TelegramPublishSentMessagePostTypeFromMediaCount(mediaCount)

	// Check if post type changed - we cannot change post type after publishing
	postTypeChanged := currentPostType != newPostType
	if postTypeChanged {
		logger.Warn("post type change detected, ignoring media changes",
			"current_type", currentPostType,
			"new_type", newPostType,
			"note_path_id", params.NotePathID,
		)
	}

	// Use current post type for determining content length limit (like bot does)
	hasMedia := currentPostType == db.TelegramPublishSentMessagePostTypePhoto || currentPostType == db.TelegramPublishSentMessagePostTypeMediaGroup

	// Truncate content to telegram limits
	maxLength := 4096
	if hasMedia {
		maxLength = env.TelegramCaptionLengthLimit(ctx, &params.AccountID)
	}
	content := telegram.TruncateContent(post.Content, maxLength)

	// Calculate content hash for new content
	// For photo: include media URL (can be changed)
	// For media_group: only text (can't change media, only caption)
	hashInput := content
	if currentPostType == db.TelegramPublishSentMessagePostTypePhoto && len(post.Media) > 0 {
		hashInput += "|" + post.Media[0]
	}
	hash := sha256.Sum256([]byte(hashInput))
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

	// Decrypt session data
	sessionData, err := env.DecryptData(account.SessionData)
	if err != nil {
		return fmt.Errorf("failed to decrypt session data: %w", err)
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

	// Determine which edit method to use based on CURRENT post type (saved in DB)
	// We cannot change post type after publishing, so always use current type
	switch currentPostType {
	case db.TelegramPublishSentMessagePostTypeText:
		// Edit text message
		editErr = client.EditMessage(ctx, sessionData, tgtd.EditMessageParams{
			ChatID:    params.TelegramChatID,
			MessageID: params.MessageID,
			Message:   content,
		})
	case db.TelegramPublishSentMessagePostTypePhoto:
		// Edit photo with caption (can replace photo)
		if len(post.Media) > 0 {
			editErr = client.EditMessageWithPhoto(ctx, sessionData, tgtd.EditMessageWithPhotoParams{
				ChatID:    params.TelegramChatID,
				MessageID: params.MessageID,
				PhotoURL:  post.Media[0],
				Caption:   content,
			})
		} else {
			// No photo in update, just edit caption
			editErr = client.EditMessageCaption(ctx, sessionData, tgtd.EditMessageCaptionParams{
				ChatID:    params.TelegramChatID,
				MessageID: params.MessageID,
				Caption:   content,
			})
		}
	case db.TelegramPublishSentMessagePostTypeMediaGroup:
		// Edit caption only for media_group (cannot change media)
		editErr = client.EditMessageCaption(ctx, sessionData, tgtd.EditMessageCaptionParams{
			ChatID:    params.TelegramChatID,
			MessageID: params.MessageID,
			Caption:   content,
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
