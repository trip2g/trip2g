package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"trip2g/internal/case/handletgupdate"
	"trip2g/internal/db"
	graphmodel "trip2g/internal/graph/model"
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
		PollInterval: time.Second * 1,
	})
	a.telegramAPIQueue = apiQueue

	// Task queue - for telegram-related background tasks (processing posts, etc.)
	taskQueue := a.createQueue(ctx, "tg_task_jobs", jobs.NewRunnerOpts{
		Limit:        1,
		PollInterval: time.Second * 1,
	})

	a.telegramTaskQueue = taskQueue

	// Initialize telegram auth manager for MTProto user account authentication
	a.telegramAuthManager = tgtd.NewAuthManager()

	return a.initTelegramBots(ctx)
}

// TelegramAuthManager returns the auth manager for telegram user accounts
func (a *app) TelegramAuthManager() *tgtd.AuthManager {
	return a.telegramAuthManager
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
		apiResp, err := handlerIO.Request(msg)
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

// TelegramAccountChats fetches chats for a telegram account and enriches them with publish tag info
func (a *app) TelegramAccountChats(ctx context.Context, accountID int64) ([]graphmodel.AdminTelegramAccountChat, error) {
	// Get the account to retrieve api credentials and session data
	account, err := a.GetTelegramAccountByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram account: %w", err)
	}

	// Create tgtd client
	client := tgtd.NewClient(int(account.ApiID), account.ApiHash)

	// List chats from Telegram
	chats, err := client.ListChats(ctx, account.SessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to list chats: %w", err)
	}

	// Get all publish tags and instant tags for this account's chats
	publishChats, err := a.ListTelegramPublishAccountChatsByAccountID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list publish chats: %w", err)
	}

	instantChats, err := a.ListTelegramPublishAccountInstantChatsByAccountID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list instant chats: %w", err)
	}

	// Build map of chat_id -> tags
	publishTagsByChat := make(map[int64][]int64)
	for _, pc := range publishChats {
		publishTagsByChat[pc.TelegramChatID] = append(publishTagsByChat[pc.TelegramChatID], pc.TagID)
	}

	instantTagsByChat := make(map[int64][]int64)
	for _, ic := range instantChats {
		instantTagsByChat[ic.TelegramChatID] = append(instantTagsByChat[ic.TelegramChatID], ic.TagID)
	}

	// Get all tags for lookup
	allTags, err := a.ListAllTelegramPublishTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list publish tags: %w", err)
	}

	tagsByID := make(map[int64]db.TelegramPublishTag)
	for _, tag := range allTags {
		tagsByID[tag.ID] = tag
	}

	// Build result
	result := make([]graphmodel.AdminTelegramAccountChat, 0, len(chats))
	for _, chat := range chats {
		publishTags := make([]db.TelegramPublishTag, 0)
		for _, tagID := range publishTagsByChat[chat.ID] {
			if tag, ok := tagsByID[tagID]; ok {
				publishTags = append(publishTags, tag)
			}
		}

		instantTags := make([]db.TelegramPublishTag, 0)
		for _, tagID := range instantTagsByChat[chat.ID] {
			if tag, ok := tagsByID[tagID]; ok {
				instantTags = append(instantTags, tag)
			}
		}

		result = append(result, graphmodel.AdminTelegramAccountChat{
			TelegramChatID:     strconv.FormatInt(chat.ID, 10),
			ChatTitle:          chat.Title,
			ChatType:           chat.ChatType,
			PublishTags:        publishTags,
			PublishInstantTags: instantTags,
		})
	}

	return result, nil
}
