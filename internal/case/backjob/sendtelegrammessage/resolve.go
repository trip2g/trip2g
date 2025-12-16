package sendtelegrammessage

import (
	"context"
	"crypto/sha256"
	"database/sql"
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
	"trip2g/internal/tgtd"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const parseMode = "HTML"

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg sendtelegrammessage_test . Env

type Env interface {
	SendTelegramMessage(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error)
	InsertTelegramPublishSentMessage(ctx context.Context, arg db.InsertTelegramPublishSentMessageParams) error
	CheckTelegramPublishSentMessageExists(ctx context.Context, arg db.CheckTelegramPublishSentMessageExistsParams) (int64, error)
	LatestNoteViews() *model.NoteViews
	UpdateTelegramPublishPost(ctx context.Context, notePathID int64) error
	SetTelegramPublishNoteLastError(ctx context.Context, arg db.SetTelegramPublishNoteLastErrorParams) error
	ClearTelegramPublishNoteLastError(ctx context.Context, notePathID int64) error
	TelegramCaptionLengthLimit(ctx context.Context, accountID *int64) int
	Logger() logger.Logger
}

func Resolve(ctx context.Context, env Env, params model.TelegramSendPostParams) error {
	// 10 minutes timeout for large file uploads (videos can be 300MB+)
	jobTimeout := 10 * time.Minute

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
	// Check if message already exists before sending to avoid duplicate messages
	exists, err := env.CheckTelegramPublishSentMessageExists(ctx, db.CheckTelegramPublishSentMessageExistsParams{
		NotePathID: params.NotePathID,
		ChatID:     params.DBChatID,
	})
	if err != nil {
		return fmt.Errorf("failed to check if message exists: %w", err)
	}

	// If message already exists, skip sending (this can happen with job retries)
	if exists != 0 {
		env.Logger().Info("telegram message already sent, skipping",
			"note_path_id", params.NotePathID,
			"chat_id", params.DBChatID,
		)
		return nil
	}

	var (
		messageID int64
		postType  string
	)

	post := params.Post

	// Determine post type based on media count
	mediaCount := len(post.Media)
	postType = db.TelegramPublishSentMessagePostTypeFromMediaCount(mediaCount)

	// Truncate content to telegram limits
	maxLength := 4096
	if mediaCount > 0 {
		maxLength = env.TelegramCaptionLengthLimit(ctx, nil)
	}
	content := telegram.TruncateContent(post.Content, maxLength)

	switch mediaCount {
	case 0:
		// Send as text message
		msg := tgbotapi.NewMessage(params.TelegramChatID, content)
		msg.ParseMode = parseMode
		msg.DisableNotification = params.DisableNotification
		msg.DisableWebPagePreview = post.DisableWebPagePreview

		messageID, err = env.SendTelegramMessage(ctx, params.DBChatID, msg)
	case 1:
		// Send as single photo (can be edited later)
		paramsCopy := params
		paramsCopy.Post.Content = content
		messageID, err = tryToSendPhoto(ctx, env, paramsCopy)
	default:
		// Send as media group (2-10 media files)
		paramsCopy := params
		paramsCopy.Post.Content = content
		messageID, err = tryToSendMediaGroup(ctx, env, paramsCopy)
	}

	if err != nil {
		// Store error message - post will not be retried
		errMsg := err.Error()
		setErr := env.SetTelegramPublishNoteLastError(ctx, db.SetTelegramPublishNoteLastErrorParams{
			LastError: sql.NullString{
				String: errMsg,
				Valid:  true,
			},
			NotePathID: params.NotePathID,
		})
		if setErr != nil {
			env.Logger().Error("failed to set last_error", "error", setErr)
		}
		return fmt.Errorf("failed to send: %w", err)
	}

	// Use truncated content for hash and storage
	hash := sha256.Sum256([]byte(content))
	contentHash := hex.EncodeToString(hash[:])

	sentParams := db.InsertTelegramPublishSentMessageParams{
		NotePathID:  params.NotePathID,
		ChatID:      params.DBChatID,
		MessageID:   messageID,
		Instant:     params.Instant,
		ContentHash: contentHash,
		Content:     content,
		PostType:    postType,
	}

	err = env.InsertTelegramPublishSentMessage(ctx, sentParams)
	if err != nil {
		return fmt.Errorf("failed to InsertTelegramPublishSentMessage: %w", err)
	}

	// Clear last_error on successful send (in case this was a manual retry)
	err = env.ClearTelegramPublishNoteLastError(ctx, params.NotePathID)
	if err != nil {
		env.Logger().Error("failed to clear last_error", "error", err)
		// Don't fail the job if clear fails - the message was sent successfully
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

func tryToSendPhoto(ctx context.Context, env Env, params model.TelegramSendPostParams) (int64, error) {
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

func tryToSendMediaGroup(ctx context.Context, env Env, params model.TelegramSendPostParams) (int64, error) {
	messageID, convertErr := sendMediaGroup(ctx, env, params, false)
	if convertErr != nil {
		// workaround for localhost minio or unreachable URLs
		if strings.Contains(convertErr.Error(), "wrong HTTP URL specified") ||
			strings.Contains(convertErr.Error(), "EXTERNAL_URL_INVALID") {
			messageID, convertErr = sendMediaGroup(ctx, env, params, true)
		}

		if convertErr != nil {
			return 0, fmt.Errorf("failed to send media group: %w", convertErr)
		}
	}

	return messageID, nil
}

func sendPhoto(ctx context.Context, env Env, params model.TelegramSendPostParams, stream bool) (int64, error) {
	var file tgbotapi.RequestFileData

	mediaURL := params.Post.Media[0]

	if !stream {
		file = tgbotapi.FileURL(mediaURL)
	} else {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to create request for media URL: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch media URL: %w", err)
		}

		defer resp.Body.Close()

		file = tgbotapi.FileReader{
			Name:   filepath.Base(mediaURL),
			Reader: resp.Body,
		}
	}

	// Check if it's a video or photo
	if tgtd.IsVideoURL(mediaURL) {
		video := tgbotapi.NewVideo(params.TelegramChatID, file)
		video.Caption = params.Post.Content
		video.ParseMode = parseMode
		video.DisableNotification = params.DisableNotification
		return env.SendTelegramMessage(ctx, params.DBChatID, video)
	}

	photo := tgbotapi.NewPhoto(params.TelegramChatID, file)
	photo.Caption = params.Post.Content
	photo.ParseMode = parseMode
	photo.DisableNotification = params.DisableNotification

	return env.SendTelegramMessage(ctx, params.DBChatID, photo)
}

func sendMediaGroup(ctx context.Context, env Env, params model.TelegramSendPostParams, stream bool) (int64, error) {
	var mediaGroup []interface{}

	for i, mediaURL := range params.Post.Media {
		// Determine media type based on extension
		ext := strings.ToLower(filepath.Ext(mediaURL))

		var fileData tgbotapi.RequestFileData

		if !stream {
			fileData = tgbotapi.FileURL(mediaURL)
		} else {
			// Download the file and send as stream
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
			if err != nil {
				return 0, fmt.Errorf("failed to create request for media URL %s: %w", mediaURL, err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return 0, fmt.Errorf("failed to fetch media URL %s: %w", mediaURL, err)
			}

			defer resp.Body.Close()

			fileData = tgbotapi.FileReader{
				Name:   filepath.Base(mediaURL),
				Reader: resp.Body,
			}
		}

		var media interface{}

		switch ext {
		case ".mp4", ".avi", ".mov", ".mkv", ".webm", ".m4v":
			video := tgbotapi.NewInputMediaVideo(fileData)
			// Only first media item gets caption
			if i == 0 {
				video.Caption = params.Post.Content
				video.ParseMode = parseMode
			}
			media = video
		default:
			// Treat as photo by default
			photo := tgbotapi.NewInputMediaPhoto(fileData)
			// Only first media item gets caption
			if i == 0 {
				photo.Caption = params.Post.Content
				photo.ParseMode = parseMode
			}
			media = photo
		}

		mediaGroup = append(mediaGroup, media)
	}

	config := tgbotapi.NewMediaGroup(params.TelegramChatID, mediaGroup)
	config.DisableNotification = params.DisableNotification

	messageID, err := env.SendTelegramMessage(ctx, params.DBChatID, config)
	if err != nil {
		return 0, fmt.Errorf("failed to send media group: %w", err)
	}

	return messageID, nil
}
