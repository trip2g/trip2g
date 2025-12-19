package sendtelegramaccountmessage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/telegram"
	"trip2g/internal/tgtd"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg sendtelegramaccountmessage_test . Env

type Env interface {
	InsertTelegramPublishSentAccountMessage(ctx context.Context, arg db.InsertTelegramPublishSentAccountMessageParams) error
	CheckTelegramPublishSentAccountMessageExists(ctx context.Context, arg db.CheckTelegramPublishSentAccountMessageExistsParams) (int64, error)
	LatestNoteViews() *model.NoteViews
	UpdateTelegramAccountPublishPost(ctx context.Context, notePathID int64) error
	SetTelegramPublishNoteLastError(ctx context.Context, arg db.SetTelegramPublishNoteLastErrorParams) error
	ClearTelegramPublishNoteLastError(ctx context.Context, notePathID int64) error
	GetTelegramAccountByID(ctx context.Context, id int64) (db.TelegramAccount, error)
	DecryptData(ciphertext []byte) ([]byte, error)
	TelegramCaptionLengthLimit(ctx context.Context, accountID *int64) int
	Logger() logger.Logger
	// Access hash cache (tgtd.ClientEnv)
	GetTelegramPublishAccountChatAccessHash(ctx context.Context, arg db.GetTelegramPublishAccountChatAccessHashParams) (*string, error)
	GetTelegramPublishAccountInstantChatAccessHash(ctx context.Context, arg db.GetTelegramPublishAccountInstantChatAccessHashParams) (*string, error)
	UpdateTelegramPublishAccountChatAccessHash(ctx context.Context, arg db.UpdateTelegramPublishAccountChatAccessHashParams) error
	UpdateTelegramPublishAccountInstantChatAccessHash(ctx context.Context, arg db.UpdateTelegramPublishAccountInstantChatAccessHashParams) error
}

func Resolve(ctx context.Context, env Env, params model.TelegramAccountSendPostParams) error {
	// 10 minutes timeout for large file uploads (videos can be 300MB+)
	jobTimeout := 10 * time.Minute

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

func resolve1(ctx context.Context, env Env, params model.TelegramAccountSendPostParams) error {
	// Check if message already exists before sending to avoid duplicate messages
	exists, err := env.CheckTelegramPublishSentAccountMessageExists(ctx, db.CheckTelegramPublishSentAccountMessageExistsParams{
		NotePathID:     params.NotePathID,
		AccountID:      params.AccountID,
		TelegramChatID: params.TelegramChatID,
	})
	if err != nil {
		return fmt.Errorf("failed to check if message exists: %w", err)
	}

	// If message already exists, skip sending (this can happen with job retries)
	if exists != 0 {
		env.Logger().Info("telegram account message already sent, skipping",
			"note_path_id", params.NotePathID,
			"account_id", params.AccountID,
			"telegram_chat_id", params.TelegramChatID,
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

	post := params.Post

	// Determine post type based on media count
	mediaCount := len(post.Media)
	postType := db.TelegramPublishSentMessagePostTypeFromMediaCount(mediaCount)

	// Truncate content to telegram limits
	maxLength := 4096
	if mediaCount > 0 {
		maxLength = env.TelegramCaptionLengthLimit(ctx, &params.AccountID)
	}
	content := telegram.TruncateContent(post.Content, maxLength)

	// Create tgtd client and send message
	client := tgtd.NewClient(env, account.ID, int(account.ApiID), account.ApiHash)

	var result *tgtd.SendMessageResult
	var sendErr error

	switch mediaCount {
	case 0:
		// Send as text message
		result, sendErr = client.SendMessage(ctx, sessionData, tgtd.SendMessageParams{
			ChatID:    params.TelegramChatID,
			Message:   content,
			NoWebpage: post.DisableWebPagePreview,
		})
	case 1:
		// Send as single photo or video
		mediaURL := post.Media[0]
		if tgtd.IsVideoURL(mediaURL) {
			result, sendErr = client.SendVideo(ctx, sessionData, tgtd.SendVideoParams{
				ChatID:   params.TelegramChatID,
				VideoURL: mediaURL,
				Caption:  content,
			})
		} else {
			result, sendErr = client.SendPhoto(ctx, sessionData, tgtd.SendPhotoParams{
				ChatID:   params.TelegramChatID,
				PhotoURL: mediaURL,
				Caption:  content,
			})
		}
	default:
		// Send as media group (2-10 media files)
		result, sendErr = client.SendMediaGroup(ctx, sessionData, tgtd.SendMediaGroupParams{
			ChatID:    params.TelegramChatID,
			MediaURLs: post.Media,
			Caption:   content,
		})
	}

	if sendErr != nil {
		// Store error message
		errMsg := sendErr.Error()
		setErr := env.SetTelegramPublishNoteLastError(ctx, db.SetTelegramPublishNoteLastErrorParams{
			LastError:  &errMsg,
			NotePathID: params.NotePathID,
		})
		if setErr != nil {
			env.Logger().Error("failed to set last_error", "error", setErr)
		}
		return fmt.Errorf("failed to send via account: %w", sendErr)
	}

	// Calculate content hash
	hash := sha256.Sum256([]byte(content))
	contentHash := hex.EncodeToString(hash[:])

	var instant int64
	if params.Instant {
		instant = 1
	}

	sentParams := db.InsertTelegramPublishSentAccountMessageParams{
		NotePathID:     params.NotePathID,
		AccountID:      params.AccountID,
		TelegramChatID: params.TelegramChatID,
		MessageID:      result.MessageID,
		Instant:        instant,
		ContentHash:    contentHash,
		Content:        content,
		PostType:       postType,
	}

	err = env.InsertTelegramPublishSentAccountMessage(ctx, sentParams)
	if err != nil {
		return fmt.Errorf("failed to InsertTelegramPublishSentAccountMessage: %w", err)
	}

	// Clear last_error on successful send
	err = env.ClearTelegramPublishNoteLastError(ctx, params.NotePathID)
	if err != nil {
		env.Logger().Error("failed to clear last_error", "error", err)
	}

	// If requested, enqueue updates for linked posts
	if params.UpdateLinkedPosts {
		nvs := env.LatestNoteViews()
		noteView := nvs.GetByPathID(params.NotePathID)
		if noteView == nil {
			return nil
		}

		for inLink := range noteView.InLinks {
			inNote, ok := nvs.Map[inLink]
			if ok && inNote.IsTelegramPublishPost() {
				updateErr := env.UpdateTelegramAccountPublishPost(ctx, inNote.PathID)
				if updateErr != nil {
					return fmt.Errorf("failed to update linked post %d: %w", inNote.PathID, updateErr)
				}
			}
		}
	}

	return nil
}
