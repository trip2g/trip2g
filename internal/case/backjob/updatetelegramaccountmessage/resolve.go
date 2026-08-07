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

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg updatetelegramaccountmessage_test . Env

type Env interface {
	Logger() logger.Logger
	LogLevel() string
	GetTelegramPublishSentAccountMessageContentHash(ctx context.Context, arg db.GetTelegramPublishSentAccountMessageContentHashParams) (string, error)
	GetTelegramPublishSentAccountMessagePostType(ctx context.Context, arg db.GetTelegramPublishSentAccountMessagePostTypeParams) (string, error)
	UpdateTelegramPublishSentAccountMessageContent(ctx context.Context, arg db.UpdateTelegramPublishSentAccountMessageContentParams) error
	GetTelegramAccountByID(ctx context.Context, id int64) (db.TelegramAccount, error)
	DecryptData(ciphertext []byte) ([]byte, error)
	TelegramCaptionLengthLimit(ctx context.Context, accountID *int64) int
	// Access hash cache (tgtd.ClientEnv)
	GetTelegramPublishAccountChatAccessHash(ctx context.Context, arg db.GetTelegramPublishAccountChatAccessHashParams) (*string, error)
	GetTelegramPublishAccountInstantChatAccessHash(ctx context.Context, arg db.GetTelegramPublishAccountInstantChatAccessHashParams) (*string, error)
	UpdateTelegramPublishAccountChatAccessHash(ctx context.Context, arg db.UpdateTelegramPublishAccountChatAccessHashParams) error
	UpdateTelegramPublishAccountInstantChatAccessHash(ctx context.Context, arg db.UpdateTelegramPublishAccountInstantChatAccessHashParams) error
}

// maxAttempts bounds the total telegram rate-limit retries (initial try + retries).
const maxAttempts = 3

func Resolve(ctx context.Context, env Env, params model.TelegramAccountUpdatePostParams) error {
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

//nolint:funlen,gocognit // complex edit logic with multiple post types
func resolve(ctx context.Context, env Env, params model.TelegramAccountUpdatePostParams) error {
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

	// Every edit branch below is a classic one, and each of them destroys a rich
	// message: they replace the block tree with a flat string or address a
	// caption it does not have. The account path cannot create a rich message
	// today — sendRichMessage has no MTProto equivalent — so this refuses rather
	// than guesses, and stays correct on the day it can.
	//
	// Both directions are refused: editing a stored rich post would flatten it,
	// and letting a rich post in would write post_type 'rich' onto a row whose
	// message was just edited classically, which is the same corruption one step
	// later.
	if currentPostType == db.TelegramPublishSentMessagePostTypeRich || post.IsRich() {
		return fmt.Errorf("cannot edit a rich post through the account path: message %d in chat %d stays as published",
			params.MessageID, params.TelegramChatID)
	}

	// Determine new post type
	mediaCount := len(post.Media)
	newPostType := db.TelegramPublishSentMessagePostTypeFor(post.IsRich(), mediaCount)

	// Check if post type changed
	// Supported transitions: text→photo (add media)
	// Not supported: photo→text, any→media_group, media_group→any
	postTypeChanged := currentPostType != newPostType
	if postTypeChanged {
		canConvert := currentPostType == db.TelegramPublishSentMessagePostTypeText &&
			newPostType == db.TelegramPublishSentMessagePostTypePhoto
		if !canConvert {
			logger.Warn("unsupported post type change, ignoring media changes",
				"current_type", currentPostType,
				"new_type", newPostType,
				"note_path_id", params.NotePathID,
			)
		}
	}

	// Use target post type for determining content length limit
	// When converting text→photo, we need caption limit (1024) not text limit (4096)
	targetType := currentPostType
	if postTypeChanged && currentPostType == db.TelegramPublishSentMessagePostTypeText &&
		newPostType == db.TelegramPublishSentMessagePostTypePhoto {
		targetType = newPostType
	}
	hasMedia := targetType == db.TelegramPublishSentMessagePostTypePhoto || targetType == db.TelegramPublishSentMessagePostTypeMediaGroup

	// Truncate content to telegram limits
	maxLength := 4096
	if hasMedia {
		maxLength = env.TelegramCaptionLengthLimit(ctx, &params.AccountID)
	}
	content := telegram.TruncateContent(post.Content, maxLength)

	// Calculate content hash for new content
	// For photo (including text→photo conversion): include media URL (can be changed)
	// For media_group: only text (can't change media, only caption)
	hashInput := content
	if targetType == db.TelegramPublishSentMessagePostTypePhoto && len(post.Media) > 0 {
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
	client := tgtd.NewClient(env, account.ID, int(account.ApiID), account.ApiHash)

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

	// Determine which edit method to use
	// For text→photo: use EditMessageWithPhoto to add media
	// For photo→text: not supported (would need to delete and resend)
	// Otherwise: use method based on current type
	switch currentPostType {
	case db.TelegramPublishSentMessagePostTypeText:
		if newPostType == db.TelegramPublishSentMessagePostTypePhoto && len(post.Media) > 0 {
			// Convert text message to photo by adding media
			mediaURL := post.Media[0]
			editErr = client.EditMessageWithPhoto(ctx, sessionData, tgtd.EditMessageWithPhotoParams{
				ChatID:    params.TelegramChatID,
				MessageID: params.MessageID,
				PhotoURL:  mediaURL,
				Caption:   content,
			})
		} else {
			// Edit text message
			editErr = client.EditMessage(ctx, sessionData, tgtd.EditMessageParams{
				ChatID:    params.TelegramChatID,
				MessageID: params.MessageID,
				Message:   content,
			})
		}
	case db.TelegramPublishSentMessagePostTypePhoto:
		// Edit photo/video with caption
		if len(post.Media) > 0 {
			mediaURL := post.Media[0]
			if tgtd.IsVideoURL(mediaURL) {
				// For video, only edit caption (cannot replace video via this API)
				editErr = client.EditMessageCaption(ctx, sessionData, tgtd.EditMessageCaptionParams{
					ChatID:    params.TelegramChatID,
					MessageID: params.MessageID,
					Caption:   content,
				})
			} else {
				// For photo, can replace the image
				editErr = client.EditMessageWithPhoto(ctx, sessionData, tgtd.EditMessageWithPhotoParams{
					ChatID:    params.TelegramChatID,
					MessageID: params.MessageID,
					PhotoURL:  mediaURL,
					Caption:   content,
				})
			}
		} else {
			// No media in update, just edit caption
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
