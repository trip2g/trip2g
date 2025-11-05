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

	// Truncate content to telegram limits (minus 3 for '...')
	content := telegram.TruncateContent(post.Content, len(post.Images) > 0)

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

	// Edit the message in Telegram
	var editErr error
	if len(post.Images) > 0 {
		// Edit caption for photo message
		editMsg := tgbotapi.NewEditMessageCaption(params.TelegramChatID, int(params.MessageID), content)
		editMsg.ParseMode = "HTML"
		editErr = env.SendTelegramRequest(ctx, params.DBChatID, editMsg)
	} else {
		// Edit text for text message
		editMsg := tgbotapi.NewEditMessageText(params.TelegramChatID, int(params.MessageID), content)
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
