package updatetelegrammessage

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
	"trip2g/internal/tgrich"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg updatetelegrammessage_test . Env

type Env interface {
	Logger() logger.Logger
	GetTelegramPublishSentMessageContentHash(ctx context.Context, arg db.GetTelegramPublishSentMessageContentHashParams) (string, error)
	GetTelegramPublishSentMessagePostType(ctx context.Context, arg db.GetTelegramPublishSentMessagePostTypeParams) (string, error)
	SendTelegramRequest(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error
	EditTelegramRichMessage(ctx context.Context, chatID int64, req tgrich.EditRequest) (tgrich.SendResult, error)
	UpdateTelegramPublishSentMessageContent(ctx context.Context, arg db.UpdateTelegramPublishSentMessageContentParams) error
	TelegramCaptionLengthLimit(ctx context.Context, accountID *int64) int
}

// maxAttempts bounds the total telegram rate-limit retries (initial try + retries).
const maxAttempts = 3

func Resolve(ctx context.Context, env Env, params model.TelegramUpdatePostParams) error {
	// 5 minutes timeout - updates mostly edit captions, but can replace photos.
	// Derive from the parent ctx so app shutdown cancels the job.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = resolve(ctx, env, params)
		if err == nil {
			return nil
		}

		shouldRetry, delay := telegram.HandleRateLimit(err)
		if !shouldRetry || attempt == maxAttempts {
			return err
		}

		env.Logger().Info("telegram rate limit hit, retrying after delay",
			"delay", delay,
			"attempt", attempt,
			"job", JobID,
		)

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return err
}

func resolve(ctx context.Context, env Env, params model.TelegramUpdatePostParams) error {
	logger := logger.WithPrefix(env.Logger(), "backjob/updatetelegrammessage:")
	post := params.Post

	// Get current post type from database
	currentPostType, err := env.GetTelegramPublishSentMessagePostType(ctx, db.GetTelegramPublishSentMessagePostTypeParams{
		NotePathID: params.NotePathID,
		ChatID:     params.DBChatID,
		MessageID:  params.MessageID,
	})
	if err != nil {
		return fmt.Errorf("failed to get current post type: %w", err)
	}

	// Determine new post type based on current media. Rich must be classified
	// before media count is consulted, otherwise a rich post reads as 'text' and
	// looks like it changed type on every single edit.
	mediaCount := len(post.Media)
	newPostType := db.TelegramPublishSentMessagePostTypeFor(post.IsRich(), mediaCount)

	isRich := currentPostType == db.TelegramPublishSentMessagePostTypeRich

	// A rich message cannot be edited by either classic branch below: one
	// flattens the block tree into a plain string, the other targets a caption it
	// does not have. Both are irreversible, so once the note stops producing
	// blocks the published message is left exactly as it is.
	if isRich && newPostType != db.TelegramPublishSentMessagePostTypeRich {
		logger.Warn("refusing to rewrite a rich post as a classic one, leaving it untouched",
			"new_type", newPostType,
			"note_path_id", params.NotePathID,
			"chat_id", params.DBChatID,
			"message_id", params.MessageID,
		)

		return nil
	}

	// Check if post type changed - if so, add warning and use original type
	postTypeChanged := currentPostType != newPostType
	if postTypeChanged {
		warning := fmt.Sprintf(
			"Cannot change post type from '%s' to '%s' after publishing. "+
				"To update media, reset the post in admin panel and republish.",
			currentPostType,
			newPostType,
		)
		post.Warnings = append(post.Warnings, warning)
		logger.Info(
			"post type change detected, ignoring media changes",
			"current_type", currentPostType,
			"new_type", newPostType,
			"note_path_id", params.NotePathID,
		)
	}

	// Use current post type for determining content length limit
	hasMedia := currentPostType == db.TelegramPublishSentMessagePostTypePhoto || currentPostType == db.TelegramPublishSentMessagePostTypeMediaGroup

	// Truncate content to telegram limits
	maxLength := 4096
	if hasMedia {
		maxLength = env.TelegramCaptionLengthLimit(ctx, nil)
	}
	content := telegram.TruncateContent(post.Content, maxLength)

	// Calculate content hash for new content
	hash := sha256.Sum256([]byte(content))
	newContentHash := hex.EncodeToString(hash[:])

	// Get current content hash from database
	currentContentHash, err := env.GetTelegramPublishSentMessageContentHash(ctx, db.GetTelegramPublishSentMessageContentHashParams{
		NotePathID: params.NotePathID,
		ChatID:     params.DBChatID,
		MessageID:  params.MessageID,
	})
	if err != nil {
		return fmt.Errorf("failed to get current content hash: %w", err)
	}

	// Skip update if content hasn't changed
	if currentContentHash == newContentHash {
		logger.Info("skip, content unchanged", "note_path_id", params.NotePathID, "chat_id", params.DBChatID, "message_id", params.MessageID)
		return nil
	}

	// Edit the message in Telegram based on current (saved) post type
	var editErr error
	switch {
	case isRich:
		editErr = editRich(ctx, env, params)
	case currentPostType == db.TelegramPublishSentMessagePostTypeText:
		// Edit text for text message
		editMsg := tgbotapi.NewEditMessageText(params.TelegramChatID, int(params.MessageID), content)
		editMsg.ParseMode = "HTML"
		editErr = env.SendTelegramRequest(ctx, params.DBChatID, editMsg)
	default:
		// Edit caption for photo or media_group
		editMsg := tgbotapi.NewEditMessageCaption(params.TelegramChatID, int(params.MessageID), content)
		editMsg.ParseMode = "HTML"
		editErr = env.SendTelegramRequest(ctx, params.DBChatID, editMsg)
	}

	if editErr != nil {
		logger.Debug("edit error", "error", editErr.Error(), "note_path_id", params.NotePathID, "chat_id", params.DBChatID, "message_id", params.MessageID)

		// If Telegram says content is the same, it's not an error - just update hash in DB
		if !strings.Contains(editErr.Error(), "are exactly the same as a current content") {
			return fmt.Errorf("failed to edit telegram message: %w", editErr)
		}

		logger.Info("already up-to-date", "note_path_id", params.NotePathID, "chat_id", params.DBChatID, "message_id", params.MessageID)
	} else {
		logger.Debug("updated", "note_path_id", params.NotePathID, "chat_id", params.DBChatID, "message_id", params.MessageID)
	}

	// Update the database with new content hash
	updateParams := db.UpdateTelegramPublishSentMessageContentParams{
		ContentHash: newContentHash,
		Content:     content,
		NotePathID:  params.NotePathID,
		ChatID:      params.DBChatID,
		MessageID:   params.MessageID,
	}

	err = env.UpdateTelegramPublishSentMessageContent(ctx, updateParams)
	if err != nil {
		return fmt.Errorf("failed to update sent message content in DB: %w", err)
	}

	return nil
}
