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

	// For now, only support text messages
	if currentPostType != "text" {
		logger.Warn("only text messages can be updated via account",
			"current_type", currentPostType,
			"note_path_id", params.NotePathID,
		)
		return nil
	}

	// Truncate content to telegram limits
	content := telegram.TruncateContent(post.Content, false)

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

	// Create tgtd client and edit message
	client := tgtd.NewClient(int(account.ApiID), account.ApiHash)

	editErr := client.EditMessage(ctx, account.SessionData, tgtd.EditMessageParams{
		ChatID:    params.TelegramChatID,
		MessageID: params.MessageID,
		Message:   content,
	})

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
		)
	}

	// Update the database with new content hash
	updateParams := db.UpdateTelegramPublishSentAccountMessageContentParams{
		ContentHash:    newContentHash,
		Content:        content,
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
