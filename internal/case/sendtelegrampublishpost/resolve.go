package sendtelegrampublishpost

import (
	"context"
	"database/sql"
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
}

func Resolve(ctx context.Context, env Env, notePathID int64, instant bool) error {
	noteView := env.LatestNoteViews().GetByPathID(notePathID)
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

		var firstImageURL *string
		var messageID int64

		for path := range noteView.Assets {
			_, ok := noteView.AssetReplaces[path]
			if !ok {
				continue
			}

			firstImageURL = &noteView.AssetReplaces[path].URL
			break
		}

		if firstImageURL != nil {
			params := sendPhotoParams{
				chat: &chat,
				post: post,
				url:  *firstImageURL,
			}

			messageID, convertErr = sendPhoto(ctx, env, params)
			if convertErr != nil {
				// workaround for localhost minio or something similar.
				if strings.Contains(convertErr.Error(), "wrong HTTP URL specified") {
					params.stream = true
					messageID, convertErr = sendPhoto(ctx, env, params)
				}

				if convertErr != nil {
					return fmt.Errorf("failed to send photo message to chat %d: %w", chat.ID, convertErr)
				}
			}
		} else {
			msg := tgbotapi.NewMessage(chat.TelegramID, post.Content)
			msg.ParseMode = "HTML"

			messageID, convertErr = env.SendTelegramMessage(ctx, chat.ID, msg)
			if convertErr != nil {
				return fmt.Errorf("failed to send telegram message to chat %d: %w", chat.ID, convertErr)
			}
		}

		instantInt := int64(0)
		if instant {
			instantInt = 1
		}

		sentParams := db.InsertTelegramPublishSentMessageParams{
			NotePathID: notePathID,
			ChatID:     chat.ID,
			MessageID:  messageID,
			Instant:    instantInt,
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

	return nil
}

type sendPhotoParams struct {
	chat   *db.TgBotChat
	post   *model.TelegramPost
	url    string
	stream bool
}

func sendPhoto(ctx context.Context, env Env, params sendPhotoParams) (int64, error) {
	var file tgbotapi.RequestFileData

	if !params.stream {
		file = tgbotapi.FileURL(params.url)
	} else {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, params.url, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to create request for image URL: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch image URL: %w", err)
		}

		defer resp.Body.Close()

		file = tgbotapi.FileReader{
			Name:   filepath.Base(params.url),
			Reader: resp.Body,
		}
	}

	photo := tgbotapi.NewPhoto(params.chat.TelegramID, file)
	photo.Caption = params.post.Content
	photo.ParseMode = "HTML"

	return env.SendTelegramMessage(ctx, params.chat.ID, photo)
}
