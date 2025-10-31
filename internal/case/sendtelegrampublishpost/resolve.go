package sendtelegrampublishpost

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"trip2g/internal/db"
	"trip2g/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg sendtelegrampublishpost_test . Env

type Env interface {
	// Database methods for getting note and chat information
	ListTgBotChatsByTelegramPublishNotePathID(ctx context.Context, notePathID int64) ([]db.TgBotChat, error)
	ListTgBotInstantChatsByTelegramPublishNotePathID(ctx context.Context, notePathID int64) ([]db.TgBotChat, error)
	UpdateTelegramPublishNoteAsPublished(ctx context.Context, arg db.UpdateTelegramPublishNoteAsPublishedParams) error
	InsertTelegramPublishSentMessage(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error

	// Telegram bot methods for sending messages
	SendTelegramMessage(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error)

	ConvertNoteViewToTelegramPost(ctx context.Context, source model.TelegramPostSource) (*model.TelegramPost, error)

	// Content access methods
	LatestNoteViews() *model.NoteViews

	// Update linked posts
	UpdateTelegramPublishPost(ctx context.Context, notePathID int64) error
}

func Resolve(ctx context.Context, env Env, notePathID int64, instant bool) error {
	nvs := env.LatestNoteViews()

	noteView := nvs.GetByPathID(notePathID)
	if noteView == nil {
		return fmt.Errorf("note view not found for path ID %d", notePathID)
	}

	// Get chat IDs that should receive this post
	var chats []db.TgBotChat
	var err error

	if instant {
		chats, err = env.ListTgBotInstantChatsByTelegramPublishNotePathID(ctx, notePathID)
	} else {
		chats, err = env.ListTgBotChatsByTelegramPublishNotePathID(ctx, notePathID)
	}

	if err != nil {
		return fmt.Errorf("failed to get chat IDs for note: %w", err)
	}

	if len(chats) == 0 {
		if instant {
			return nil
		}

		return fmt.Errorf("no chat IDs found for note path ID %d", notePathID)
	}

	// Send the post to each chat
	for _, chat := range chats {
		source := model.TelegramPostSource{
			NoteView: noteView,
			ChatID:   chat.ID,
			Instant:  instant,
		}

		// Convert note to Telegram post
		post, convertErr := env.ConvertNoteViewToTelegramPost(ctx, source)
		if convertErr != nil {
			return fmt.Errorf("failed to convert note to telegram post: %w", convertErr)
		}

		if len(post.Warnings) > 0 {
			return fmt.Errorf("conversion produced warnings: %v", post.Warnings)
		}

		var messageID int64

		params := sendPhotoParams{
			chat: &chat,
			post: post,
			url:  getFirstImageURL(noteView),
		}

		messageID, sendErr := tryToSendPhono(ctx, env, params)
		if sendErr != nil {
			return sendErr
		}

		if messageID == 0 {
			msg := tgbotapi.NewMessage(chat.TelegramID, post.Content)
			msg.ParseMode = "HTML"

			messageID, convertErr = env.SendTelegramMessage(ctx, chat.ID, msg)
			if convertErr != nil {
				return fmt.Errorf("failed to send telegram message to chat %d: %w", chat.ID, convertErr)
			}
		}

		// Calculate content hash
		// TODO: add a hash of media too
		hash := sha256.Sum256([]byte(post.Content))
		contentHash := hex.EncodeToString(hash[:])

		sentParams := db.InsertTelegramPublishSentMessageParams{
			NotePathID:  notePathID,
			ChatID:      chat.ID,
			MessageID:   messageID,
			Instant:     instant,
			ContentHash: contentHash,
			Content:     post.Content,
		}

		insertErr := env.InsertTelegramPublishSentMessage(ctx, sentParams)
		if insertErr != nil {
			return fmt.Errorf("failed to record sent message for chat %d: %w", chat.ID, insertErr)
		}
	}

	if !instant {
		// Mark the note as published
		updateParams := db.UpdateTelegramPublishNoteAsPublishedParams{
			PublishedVersionID: sql.NullInt64{Int64: noteView.VersionID, Valid: true},
			NotePathID:         notePathID,
		}

		err = env.UpdateTelegramPublishNoteAsPublished(ctx, updateParams)
		if err != nil {
			return fmt.Errorf("failed to mark note as published: %w", err)
		}
	}

	for inLink := range noteView.InLinks {
		inNote, ok := nvs.Map[inLink]
		if ok && inNote.IsTelegramPublishPost() {
			err = env.UpdateTelegramPublishPost(ctx, inNote.PathID)
			if err != nil {
				return fmt.Errorf("failed to update linked telegram publish post for note path ID %d: %w", inNote.PathID, err)
			}
		}
	}

	return nil
}

func getFirstImageURL(noteView *model.NoteView) *string {
	for path := range noteView.Assets {
		_, ok := noteView.AssetReplaces[path]
		if !ok {
			continue
		}

		return &noteView.AssetReplaces[path].URL
	}

	return nil
}

func tryToSendPhono(ctx context.Context, env Env, params sendPhotoParams) (int64, error) {
	if params.url == nil {
		return 0, nil
	}

	messageID, convertErr := sendPhoto(ctx, env, params)
	if convertErr != nil {
		// workaround for localhost minio or something similar.
		if strings.Contains(convertErr.Error(), "wrong HTTP URL specified") {
			params.stream = true
			messageID, convertErr = sendPhoto(ctx, env, params)
		}

		if convertErr != nil {
			return 0, fmt.Errorf("failed to sendPhoto: %w", convertErr)
		}
	}

	return messageID, nil
}

type sendPhotoParams struct {
	chat   *db.TgBotChat
	post   *model.TelegramPost
	url    *string
	stream bool
}

func sendPhoto(ctx context.Context, env Env, params sendPhotoParams) (int64, error) {
	var file tgbotapi.RequestFileData

	if !params.stream {
		file = tgbotapi.FileURL(*params.url)
	} else {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, *params.url, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to create request for image URL: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch image URL: %w", err)
		}

		defer resp.Body.Close()

		file = tgbotapi.FileReader{
			Name:   filepath.Base(*params.url),
			Reader: resp.Body,
		}
	}

	photo := tgbotapi.NewPhoto(params.chat.TelegramID, file)
	photo.Caption = params.post.Content
	photo.ParseMode = "HTML"

	return env.SendTelegramMessage(ctx, params.chat.ID, photo)
}
