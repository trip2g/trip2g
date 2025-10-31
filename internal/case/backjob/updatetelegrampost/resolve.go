package updatetelegrampost

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

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg updatetelegrampost_test . Env

type Env interface {
	SendTelegramRequest(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error
	UpdateTelegramPublishSentMessageContent(ctx context.Context, arg db.UpdateTelegramPublishSentMessageContentParams) error
}

func Resolve(ctx context.Context, env Env, params model.TelegramUpdatePostParams) error {
	post := params.Post

	// Edit the message in Telegram
	var editErr error
	if len(post.Images) > 0 {
		// Edit caption for photo message
		editMsg := tgbotapi.NewEditMessageCaption(params.TelegramChatID, int(params.MessageID), post.Content)
		editMsg.ParseMode = "HTML"
		editErr = env.SendTelegramRequest(ctx, params.DBChatID, editMsg)
	} else {
		// Edit text for text message
		editMsg := tgbotapi.NewEditMessageText(params.TelegramChatID, int(params.MessageID), post.Content)
		editMsg.ParseMode = "HTML"
		editErr = env.SendTelegramRequest(ctx, params.DBChatID, editMsg)
	}

	if editErr != nil {
		// If Telegram says content is the same, it's not an error - just update hash in DB
		if !strings.Contains(editErr.Error(), "are exactly the same as a current content") {
			return fmt.Errorf("failed to edit telegram message: %w", editErr)
		}
	}

	// Calculate content hash
	hash := sha256.Sum256([]byte(post.Content))
	contentHash := hex.EncodeToString(hash[:])

	// Update the database
	updateParams := db.UpdateTelegramPublishSentMessageContentParams{
		ContentHash: contentHash,
		Content:     post.Content,
		NotePathID:  params.NotePathID,
		ChatID:      params.DBChatID,
		MessageID:   params.MessageID,
	}

	err := env.UpdateTelegramPublishSentMessageContent(ctx, updateParams)
	if err != nil {
		return fmt.Errorf("failed to update sent message content in DB: %w", err)
	}

	return nil
}
