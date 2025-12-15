package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"trip2g/internal/case/handletgupdate"
	"trip2g/internal/db"
	appmodel "trip2g/internal/model"
	"trip2g/internal/tgbots"
	"trip2g/internal/tgtd"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"maragu.dev/goqite/jobs"
)

func (a *app) initTelegramDeps(ctx context.Context) error {
	// API queue - for Telegram API calls (send messages, edit messages, etc.)
	// Limited to 1 concurrent job to avoid rate limits
	apiQueue := a.createQueue(ctx, "tg_api_jobs", jobs.NewRunnerOpts{
		Limit:        1,
		PollInterval: time.Second * 2,
	})
	a.telegramAPIQueue = apiQueue

	// Task queue - for telegram-related background tasks (processing posts, etc.)
	taskQueue := a.createQueue(ctx, "tg_task_jobs", jobs.NewRunnerOpts{
		Limit:        1,
		PollInterval: time.Second,
	})
	a.telegramTaskQueue = taskQueue

	// Long-running queue - for jobs that can take hours (channel imports, etc.)
	// Higher Extend (60s) to reduce DB updates during long jobs
	longRunningQueue := a.createQueue(ctx, "tg_long_jobs", jobs.NewRunnerOpts{
		Limit:        1,
		PollInterval: time.Second * 30,
		Extend:       time.Minute * 1,
	})
	a.telegramLongRunningQueue = longRunningQueue

	// Initialize telegram auth manager for MTProto user account authentication
	a.telegramAuthManager = tgtd.NewAuthManager()

	return a.initTelegramBots(ctx)
}

// TelegramAccountStartAuth starts authentication for a phone number.
func (a *app) TelegramAccountStartAuth(ctx context.Context, phone string, apiID int, apiHash string) (*appmodel.TelegramStartAuthResult, error) {
	pending, err := a.telegramAuthManager.StartAuth(ctx, phone, apiID, apiHash)
	if err != nil {
		return nil, err
	}

	return &appmodel.TelegramStartAuthResult{
		Phone:        pending.Phone,
		State:        mapAuthState(pending.State),
		PasswordHint: pending.PasswordHint,
	}, nil
}

// TelegramAccountCompleteAuth completes authentication with code and optional password.
func (a *app) TelegramAccountCompleteAuth(ctx context.Context, phone, code, password string) (*appmodel.TelegramCompleteAuthResult, error) {
	// Get API credentials from pending auth
	apiID, apiHash, exists := a.telegramAuthManager.GetPendingAuthAPICredentials(phone)
	if !exists {
		return nil, fmt.Errorf("no pending authentication for phone %s", phone)
	}

	result, err := a.telegramAuthManager.CompleteAuth(ctx, phone, code, password)
	if err != nil {
		return nil, err
	}

	return &appmodel.TelegramCompleteAuthResult{
		SessionData: result.SessionData,
		DisplayName: result.DisplayName,
		IsPremium:   result.IsPremium,
		APIID:       apiID,
		APIHash:     apiHash,
	}, nil
}

// TelegramAccountCancelAuth cancels a pending authentication.
func (a *app) TelegramAccountCancelAuth(phone string) error {
	return a.telegramAuthManager.CancelAuth(phone)
}

// TelegramAccountGetPasswordHint returns the password hint for a pending authentication.
func (a *app) TelegramAccountGetPasswordHint(phone string) string {
	pending := a.telegramAuthManager.GetPendingAuth(phone)
	if pending == nil {
		return ""
	}
	return pending.PasswordHint
}

// TelegramAccountGetAppConfig fetches app config from Telegram API.
func (a *app) TelegramAccountGetAppConfig(ctx context.Context, accountID int64) (string, error) {
	account, err := a.GetTelegramAccountByID(ctx, accountID)
	if err != nil {
		return "", fmt.Errorf("failed to get account: %w", err)
	}

	sessionData, err := a.DecryptSessionData(account.SessionData)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt session data: %w", err)
	}

	client := tgtd.NewClient(a, int(account.ApiID), account.ApiHash)
	config, err := client.GetAppConfig(ctx, sessionData)
	if err != nil {
		return "", err
	}
	return config.JSON, nil
}

// TelegramAccountGetUserInfo fetches user info (premium status) from Telegram API.
func (a *app) TelegramAccountGetUserInfo(ctx context.Context, accountID int64) (*tgtd.UserInfo, error) {
	account, err := a.GetTelegramAccountByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	sessionData, err := a.DecryptSessionData(account.SessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt session data: %w", err)
	}

	client := tgtd.NewClient(a, int(account.ApiID), account.ApiHash)
	return client.GetUserInfo(ctx, sessionData)
}

// TelegramCaptionLengthLimit returns the caption length limit based on account premium status.
// If accountID is nil, uses any first account's config. Returns 1024 if no accounts found.
func (a *app) TelegramCaptionLengthLimit(ctx context.Context, accountID *int64) int {
	const defaultLimit = 1024

	var account db.TelegramAccount
	var err error

	if accountID != nil {
		account, err = a.GetTelegramAccountByID(ctx, *accountID)
	} else {
		accounts, listErr := a.ListAllTelegramAccounts(ctx)
		if listErr != nil || len(accounts) == 0 {
			return defaultLimit
		}
		account = accounts[0]
	}

	if err != nil {
		return defaultLimit
	}

	// Parse app_config JSON
	var config map[string]interface{}
	if jsonErr := json.Unmarshal([]byte(account.AppConfig), &config); jsonErr != nil {
		return defaultLimit
	}

	// Get limit based on premium status
	limitKey := "caption_length_limit_default"
	if account.IsPremium == 1 {
		limitKey = "caption_length_limit_premium"
	}

	if limit, ok := config[limitKey].(float64); ok {
		return int(limit)
	}

	return defaultLimit
}

func mapAuthState(state tgtd.AuthState) appmodel.TelegramAuthState {
	switch state {
	case tgtd.AuthStateWaitingForCode:
		return appmodel.TelegramAuthStateWaitingForCode
	case tgtd.AuthStateWaitingForPassword:
		return appmodel.TelegramAuthStateWaitingForPassword
	case tgtd.AuthStateAuthorized:
		return appmodel.TelegramAuthStateAuthorized
	case tgtd.AuthStateError:
		return appmodel.TelegramAuthStateError
	}
	return appmodel.TelegramAuthStateError
}

func (a *app) initTelegramBots(ctx context.Context) error {
	var err error

	a.TgBots, err = tgbots.New(ctx, a, tgbots.DefaultConfig())
	if err != nil {
		return fmt.Errorf("failed to create Telegram bots: %w", err)
	}

	a.TgBots.SetHandler(func(ctx context.Context, io *tgbots.HandlerIO, update tgbotapi.Update) error {
		var be struct {
			*app
			*tgbots.HandlerIO
		}

		be.app = a
		be.HandlerIO = io

		return handletgupdate.Resolve(ctx, be, update)
	})

	return nil
}

func (a *app) SendTelegramMessage(ctx context.Context, chatID int64, msg tgbotapi.Chattable) (int64, error) {
	a.log.Debug("sending telegram message", "chat_id", chatID, "msg", msg)

	chat, err := a.TgBotChat(ctx, chatID)
	if err != nil {
		return 0, fmt.Errorf("failed to get Telegram chat: %w", err)
	}

	handlerIO := a.TgBots.GetHandlerIO(chat.BotID)

	if handlerIO == nil {
		return 0, fmt.Errorf("telegram bot handler IO not found for chat ID %d", chatID)
	}

	// Check if this is a media group (which returns array of messages)
	if _, isMediaGroup := msg.(tgbotapi.MediaGroupConfig); isMediaGroup {
		apiResp, err := handlerIO.Request(msg) //nolint:govet // shadow: intentional new err scope for media group branch
		if err != nil {
			return 0, fmt.Errorf("failed to send Telegram message: %w", err)
		}

		if !apiResp.Ok {
			return 0, fmt.Errorf("telegram API error: %s", apiResp.Description)
		}

		// MediaGroup returns array of messages, we need the first one's ID
		var messages []tgbotapi.Message
		err = json.Unmarshal(apiResp.Result, &messages)
		if err != nil {
			return 0, fmt.Errorf("failed to unmarshal media group response: %w", err)
		}

		if len(messages) == 0 {
			return 0, errors.New("no messages returned from media group")
		}

		return int64(messages[0].MessageID), nil
	}

	// For non-media-group messages, use regular Send
	res, err := handlerIO.Send(msg)
	if err != nil {
		return 0, fmt.Errorf("failed to send Telegram message: %w", err)
	}

	return int64(res.MessageID), nil
}

func (a *app) SendTelegramRequest(ctx context.Context, chatID int64, msg tgbotapi.Chattable) error {
	a.log.Debug("sending telegram request", "chat_id", chatID, "msg", msg)

	chat, err := a.TgBotChat(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to get Telegram chat: %w", err)
	}

	handlerIO := a.TgBots.GetHandlerIO(chat.BotID)

	if handlerIO == nil {
		return fmt.Errorf("telegram bot handler IO not found for chat ID %d", chatID)
	}

	resp, err := handlerIO.Request(msg)
	if err != nil {
		a.log.Debug("telegram request error", "chat_id", chatID, "error", err.Error())
		return fmt.Errorf("failed to send Telegram message: %w", err)
	}

	a.log.Debug("telegram request success", "chat_id", chatID, "resp", resp)

	return nil
}

func (a *app) KickTelegramChatMember(ctx context.Context, chatID, userID int64) error {
	// Get the user to find their Telegram ID
	user, err := a.UserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user by ID %d: %w", userID, err)
	}

	if !user.TgUserID.Valid {
		return fmt.Errorf("user %d does not have a Telegram ID", userID)
	}

	chat, err := a.TgBotChat(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to get Telegram chat: %w", err)
	}

	handlerIO := a.TgBots.GetHandlerIO(chat.BotID)

	if handlerIO == nil {
		return fmt.Errorf("telegram bot handler IO not found for chat ID %d", chatID)
	}

	err = handlerIO.KickChatMember(ctx, chat.TelegramID, user.TgUserID.Int64, chat.ChatType)
	if err != nil {
		return fmt.Errorf("failed to kick Telegram chat member: %w", err)
	}

	return nil
}

func (a *app) UnbanTelegramChatMember(ctx context.Context, chatID, userID int64) error {
	// Get the user to find their Telegram ID
	user, err := a.UserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user by ID %d: %w", userID, err)
	}

	if !user.TgUserID.Valid {
		return fmt.Errorf("user %d does not have a Telegram ID", userID)
	}

	chat, err := a.TgBotChat(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to get Telegram chat: %w", err)
	}

	handlerIO := a.TgBots.GetHandlerIO(chat.BotID)

	if handlerIO == nil {
		return fmt.Errorf("telegram bot handler IO not found for chat ID %d", chatID)
	}

	err = handlerIO.UnbanChatMember(ctx, chat.TelegramID, user.TgUserID.Int64)
	if err != nil {
		return fmt.Errorf("failed to unban Telegram chat member: %w", err)
	}

	return nil
}

func (a *app) BotStartLink(botID int64, param string) (string, error) {
	handlerIO := a.TgBots.GetHandlerIO(botID)
	if handlerIO == nil {
		return "", fmt.Errorf("bot with ID %d not found or not active", botID)
	}
	return handlerIO.BotStartLink(param), nil
}

// ListTelegramAccountDialogs fetches dialogs (users, channels, groups) for a telegram account.
func (a *app) ListTelegramAccountDialogs(ctx context.Context, accountID int64) ([]appmodel.TelegramAccountDialog, error) {
	// Get the account to retrieve api credentials and session data
	account, err := a.GetTelegramAccountByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram account: %w", err)
	}

	sessionData, err := a.DecryptSessionData(account.SessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt session data: %w", err)
	}

	// Create tgtd client
	client := tgtd.NewClient(a, int(account.ApiID), account.ApiHash)

	// List dialogs from Telegram
	dialogs, err := client.ListDialogs(ctx, sessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to list dialogs: %w", err)
	}

	// Convert to model
	result := make([]appmodel.TelegramAccountDialog, 0, len(dialogs))
	for _, d := range dialogs {
		result = append(result, appmodel.TelegramAccountDialog{
			AccountID: accountID,
			ID:        d.ID,
			Username:  d.Username,
			Title:     d.Title,
			Type:      appmodel.TelegramAccountDialogType(d.Type),
		})
	}

	return result, nil
}

// GetTelegramAccountDialogPublishTags returns publish tags for a specific dialog.
func (a *app) GetTelegramAccountDialogPublishTags(ctx context.Context, accountID, telegramChatID int64) ([]db.TelegramPublishTag, error) {
	return a.ListTelegramPublishTagsByAccountChatID(ctx, db.ListTelegramPublishTagsByAccountChatIDParams{
		AccountID:      accountID,
		TelegramChatID: telegramChatID,
	})
}

// GetTelegramAccountDialogPublishInstantTags returns instant publish tags for a specific dialog.
func (a *app) GetTelegramAccountDialogPublishInstantTags(ctx context.Context, accountID, telegramChatID int64) ([]db.TelegramPublishTag, error) {
	return a.ListTelegramPublishInstantTagsByAccountChatID(ctx, db.ListTelegramPublishInstantTagsByAccountChatIDParams{
		AccountID:      accountID,
		TelegramChatID: telegramChatID,
	})
}

// DeleteTelegramAccountMessage deletes a message via user account (MTProto).
func (a *app) DeleteTelegramAccountMessage(ctx context.Context, account db.TelegramAccount, chatID, messageID int64) error {
	sessionData, err := a.DecryptSessionData(account.SessionData)
	if err != nil {
		return fmt.Errorf("failed to decrypt session data: %w", err)
	}

	client := tgtd.NewClient(a, int(account.ApiID), account.ApiHash)
	return client.DeleteMessage(ctx, sessionData, tgtd.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	})
}

// TelegramClient creates a new tgtd.Client with placeholder credentials.
// Used by background jobs that need to perform multiple operations.
func (a *app) TelegramClient() *tgtd.Client {
	// Return a client with placeholder credentials - actual credentials
	// come from the account when RunWithAPI is called
	return tgtd.NewClient(a, 0, "")
}

// TelegramClientForAccount creates a tgtd.Client for a specific account.
func (a *app) TelegramClientForAccount(account db.TelegramAccount) *tgtd.Client {
	return tgtd.NewClient(a, int(account.ApiID), account.ApiHash)
}

// DecryptSessionData decrypts the encrypted session data from a telegram account.
func (a *app) DecryptSessionData(encryptedSession []byte) ([]byte, error) {
	return a.DecryptData(encryptedSession)
}
