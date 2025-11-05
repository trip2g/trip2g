package main

import (
	"context"
	"fmt"
	"time"
	"trip2g/internal/case/handletgupdate"
	"trip2g/internal/tgbots"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"maragu.dev/goqite/jobs"
)

func (a *app) RegisterTelegramAPIJob(id string, handler func(ctx context.Context, m []byte) error) {
	a.log.Info("registering telegram API job handler", "id", id)
	a.telegramAPIQueue.runner.Register(id, handler)
}

func (a *app) RegisterTelegramTaskJob(id string, handler func(ctx context.Context, m []byte) error) {
	a.log.Info("registering telegram task job handler", "id", id)
	a.telegramTaskQueue.runner.Register(id, handler)
}

func (a *app) initTelegramDeps(ctx context.Context) error {
	// API queue - for Telegram API calls (send messages, edit messages, etc.)
	// Limited to 1 concurrent job to avoid rate limits
	apiQueue := a.createQueue(ctx, "tg_api_jobs", jobs.NewRunnerOpts{
		Limit:        1,
		PollInterval: time.Second * 1,
	})
	a.telegramAPIQueue = apiQueue

	// Task queue - for telegram-related background tasks (processing posts, etc.)
	// Can run multiple jobs in parallel
	taskQueue := a.createQueue(ctx, "tg_task_jobs", jobs.NewRunnerOpts{
		Limit:        5,
		PollInterval: time.Second * 1,
	})
	a.telegramTaskQueue = taskQueue

	return a.initTelegramBots(ctx)
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
