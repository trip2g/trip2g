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

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg sendtelegramaccountmessage_test . Env

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
	LogLevel() string
	// Access hash cache (tgtd.ClientEnv)
	GetTelegramPublishAccountChatAccessHash(ctx context.Context, arg db.GetTelegramPublishAccountChatAccessHashParams) (*string, error)
	GetTelegramPublishAccountInstantChatAccessHash(ctx context.Context, arg db.GetTelegramPublishAccountInstantChatAccessHashParams) (*string, error)
	UpdateTelegramPublishAccountChatAccessHash(ctx context.Context, arg db.UpdateTelegramPublishAccountChatAccessHashParams) error
	UpdateTelegramPublishAccountInstantChatAccessHash(ctx context.Context, arg db.UpdateTelegramPublishAccountInstantChatAccessHashParams) error
}

// maxAttempts bounds the total telegram rate-limit retries (initial try + retries).
const maxAttempts = 3

func Resolve(ctx context.Context, env Env, params model.TelegramAccountSendPostParams) error {
	// 10 minutes timeout for large file uploads (videos can be 300MB+).
	// Derive from the parent ctx so app shutdown cancels the job.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
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

// failSend records a send failure on the note and returns it wrapped.
//
// last_error both surfaces in the admin and removes the note from scheduling —
// both publish queues filter on `last_error is null` — which is the right
// outcome for a failure that will not fix itself by being retried.
func failSend(ctx context.Context, env Env, params model.TelegramAccountSendPostParams, sendErr error) error {
	errMsg := sendErr.Error()

	env.Logger().Warn("telegram account send failed",
		"note_path_id", params.NotePathID,
		"account_id", params.AccountID,
		"error", errMsg,
	)

	setErr := env.SetTelegramPublishNoteLastError(ctx, db.SetTelegramPublishNoteLastErrorParams{
		LastError:  &errMsg,
		NotePathID: params.NotePathID,
	})
	if setErr != nil {
		env.Logger().Error("failed to set last_error", "error", setErr)
	}

	return fmt.Errorf("failed to send via account: %w", sendErr)
}

// send picks the transport for one post and sends it. Rich comes first because
// it is not a variation on the classic branches: it carries its own block tree
// and its own limits, so neither the truncated content nor the media count
// applies to it.
func send(
	ctx context.Context,
	env Env,
	client *tgtd.Client,
	sessionData []byte,
	account db.TelegramAccount,
	params model.TelegramAccountSendPostParams,
	content string,
) (*tgtd.SendMessageResult, error) {
	post := params.Post

	if post.IsRich() {
		messageID, err := sendRich(ctx, env, client, sessionData, account, params)
		if err != nil {
			return nil, err
		}

		return &tgtd.SendMessageResult{MessageID: messageID}, nil
	}

	switch len(post.Media) {
	case 0:
		return client.SendMessage(ctx, sessionData, tgtd.SendMessageParams{
			ChatID:    params.TelegramChatID,
			Message:   content,
			NoWebpage: post.DisableWebPagePreview,
		})
	case 1:
		mediaURL := post.Media[0]
		if tgtd.IsVideoURL(mediaURL) {
			return client.SendVideo(ctx, sessionData, tgtd.SendVideoParams{
				ChatID:   params.TelegramChatID,
				VideoURL: mediaURL,
				Caption:  content,
			})
		}

		return client.SendPhoto(ctx, sessionData, tgtd.SendPhotoParams{
			ChatID:   params.TelegramChatID,
			PhotoURL: mediaURL,
			Caption:  content,
		})
	default:
		// 2-10 media files.
		return client.SendMediaGroup(ctx, sessionData, tgtd.SendMediaGroupParams{
			ChatID:    params.TelegramChatID,
			MediaURLs: post.Media,
			Caption:   content,
		})
	}
}

func resolve(ctx context.Context, env Env, params model.TelegramAccountSendPostParams) error {
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

	post := params.Post

	// Rich is gated on the account holding Premium, enforced server-side by the
	// send call itself. Checking before the session is opened means the admin
	// reads the precondition instead of RICH_MESSAGE_UNSUPPORTED, and costs no
	// connection for a send that cannot succeed.
	if post.IsRich() {
		capability := richCapability(account)
		if !capability.Allowed {
			return failSend(ctx, env, params, fmt.Errorf("cannot send a rich message: %s", capability.Reason))
		}
	}

	// Decrypt session data
	sessionData, err := env.DecryptData(account.SessionData)
	if err != nil {
		return fmt.Errorf("failed to decrypt session data: %w", err)
	}

	// A rich post is stored as 'rich' regardless of media: it carries its media
	// inside the block tree, and the stored type is what the edit path
	// dispatches on later — a rich post typed 'text' would let a classic edit
	// flatten it.
	mediaCount := len(post.Media)
	postType := db.TelegramPublishSentMessagePostTypeFor(post.IsRich(), mediaCount)

	// Truncate content to telegram limits
	maxLength := 4096
	if mediaCount > 0 {
		maxLength = env.TelegramCaptionLengthLimit(ctx, &params.AccountID)
	}
	content := telegram.TruncateContent(post.Content, maxLength)

	// Create tgtd client and send message
	client := tgtd.NewClient(env, account.ID, int(account.ApiID), account.ApiHash)

	result, sendErr := send(ctx, env, client, sessionData, account, params, content)
	if sendErr != nil {
		return failSend(ctx, env, params, sendErr)
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
