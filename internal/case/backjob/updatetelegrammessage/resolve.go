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

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg updatetelegrammessage_test . Env

type Env interface {
	Logger() logger.Logger
	GetTelegramPublishSentMessageContentHash(ctx context.Context, arg db.GetTelegramPublishSentMessageContentHashParams) (string, error)
	GetTelegramPublishSentMessagePostType(ctx context.Context, arg db.GetTelegramPublishSentMessagePostTypeParams) (string, error)
	SendTelegramRequest(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error
	UpdateTelegramPublishSentMessageContent(ctx context.Context, arg db.UpdateTelegramPublishSentMessageContentParams) error
}

func Resolve(ctx context.Context, env Env, params model.TelegramUpdatePostParams) error {
	jobTimeout := time.Minute

	jobCtx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()

	err := Resolve1(jobCtx, env, params)
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

func Resolve1(ctx context.Context, env Env, params model.TelegramUpdatePostParams) error {
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

	// Determine new post type based on current media
	mediaCount := len(post.Media)
	var newPostType string
	switch mediaCount {
	case 0:
		newPostType = "text"
	case 1:
		newPostType = "photo"
	default:
		newPostType = "media_group"
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
	hasMedia := currentPostType == "photo" || currentPostType == "media_group"

	// Truncate content to telegram limits (minus 3 for '...')
	content := telegram.TruncateContent(post.Content, hasMedia)

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
	if currentPostType == "text" {
		// Edit text for text message
		editMsg := tgbotapi.NewEditMessageText(params.TelegramChatID, int(params.MessageID), content)
		editMsg.ParseMode = "HTML"
		editErr = env.SendTelegramRequest(ctx, params.DBChatID, editMsg)
	} else {
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
