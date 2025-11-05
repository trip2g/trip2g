package sendtelegrampost

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/telegram"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg sendtelegrampost_test . Env

type Env interface {
	SendTelegramMessage(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error)
	InsertTelegramPublishSentMessage(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error
	LatestNoteViews() *model.NoteViews
	UpdateTelegramPublishPost(ctx context.Context, notePathID int64) error
	Logger() logger.Logger
}

func Resolve(ctx context.Context, env Env, params model.TelegramSendPostParams) error {
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

func Resolve1(ctx context.Context, env Env, params model.TelegramSendPostParams) error {
	var (
		messageID int64
		err       error
	)

	post := params.Post

	if len(post.Images) > 0 {
		messageID, err = tryToSendPhono(ctx, env, params)
	} else {
		msg := tgbotapi.NewMessage(params.TelegramChatID, post.Content)
		msg.ParseMode = "HTML"

		messageID, err = env.SendTelegramMessage(ctx, params.DBChatID, msg)
	}

	if err != nil {
		return fmt.Errorf("failed to send: %w", err)
	}

	hash := sha256.Sum256([]byte(post.Content))
	contentHash := hex.EncodeToString(hash[:])

	sentParams := db.InsertTelegramPublishSentMessageParams{
		NotePathID:  params.NotePathID,
		ChatID:      params.DBChatID,
		MessageID:   messageID,
		Instant:     params.Instant,
		ContentHash: contentHash,
		Content:     post.Content,
	}

	err = env.InsertTelegramPublishSentMessage(ctx, sentParams)
	if err != nil {
		return fmt.Errorf("failed to InsertTelegramPublishSentMessage: %w", err)
	}

	// If requested, enqueue updates for linked posts
	if params.UpdateLinkedPosts {
		nvs := env.LatestNoteViews()
		noteView := nvs.GetByPathID(params.NotePathID)
		if noteView == nil {
			// Note not found, but this is not an error - it might have been deleted
			return nil
		}

		// Enqueue update for each inbound link that is a telegram publish post
		for inLink := range noteView.InLinks {
			inNote, ok := nvs.Map[inLink]
			if ok && inNote.IsTelegramPublishPost() {
				updateErr := env.UpdateTelegramPublishPost(ctx, inNote.PathID)
				if updateErr != nil {
					return fmt.Errorf("failed to update linked post %d: %w", inNote.PathID, updateErr)
				}
			}
		}
	}

	return nil
}

func tryToSendPhono(ctx context.Context, env Env, params model.TelegramSendPostParams) (int64, error) {
	messageID, convertErr := sendPhoto(ctx, env, params, false)
	if convertErr != nil {
		// workaround for localhost minio or something similar.
		if strings.Contains(convertErr.Error(), "wrong HTTP URL specified") {
			messageID, convertErr = sendPhoto(ctx, env, params, true)
		}

		if convertErr != nil {
			return 0, fmt.Errorf("failed to sendPhoto: %w", convertErr)
		}
	}

	return messageID, nil
}

func sendPhoto(ctx context.Context, env Env, params model.TelegramSendPostParams, stream bool) (int64, error) {
	var file tgbotapi.RequestFileData

	imageURL := params.Post.Images[0]

	if !stream {
		file = tgbotapi.FileURL(imageURL)
	} else {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to create request for image URL: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch image URL: %w", err)
		}

		defer resp.Body.Close()

		file = tgbotapi.FileReader{
			Name:   filepath.Base(imageURL),
			Reader: resp.Body,
		}
	}

	photo := tgbotapi.NewPhoto(params.TelegramChatID, file)
	photo.Caption = params.Post.Content
	photo.ParseMode = "HTML"

	return env.SendTelegramMessage(ctx, params.DBChatID, photo)
}
