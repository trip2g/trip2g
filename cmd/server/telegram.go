package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"trip2g/internal/case/backjob/sendtelegrampost"
	"trip2g/internal/case/backjob/updatetelegrampost"
	"trip2g/internal/case/handletgupdate"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/tgbots"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"maragu.dev/goqite"
	"maragu.dev/goqite/jobs"
)

const sendTelegramPostJobID = "send_message"
const updateTelegramPostJobID = "update_message"

func (a *app) initTelegramDeps(ctx context.Context) error {
	a.telegramQueue = goqite.New(goqite.NewOpts{
		DB:   a.writeConn,
		Name: "telegram_jobs",
	})

	a.telegramRunner = jobs.NewRunner(jobs.NewRunnerOpts{
		Limit:        1,
		Log:          logger.WithPrefix(a.log, "telegram-runner:"),
		PollInterval: time.Second,
		Queue:        a.telegramQueue,
	})

	a.telegramRunner.Register(sendTelegramPostJobID, func(ctx context.Context, m []byte) error {
		var params model.TelegramSendPostParams

		err := json.Unmarshal(m, &params)
		if err != nil {
			return fmt.Errorf("failed to unmarshal send_telegram_post params: %w", err)
		}

		return sendtelegrampost.Resolve(ctx, a, params)
	})

	a.telegramRunner.Register(updateTelegramPostJobID, func(ctx context.Context, m []byte) error {
		var params model.TelegramUpdatePostParams

		err := json.Unmarshal(m, &params)
		if err != nil {
			return fmt.Errorf("failed to unmarshal update_telegram_post params: %w", err)
		}

		return updatetelegrampost.Resolve(ctx, a, params)
	})

	// Start the shared runner
	go a.telegramRunner.Start(ctx)

	return a.initTelegramBots(ctx)
}

func (a *app) EnqueueSendTelegramPost(ctx context.Context, params model.TelegramSendPostParams) error {
	return a.enqueueJobToQ(ctx, a.telegramQueue, sendTelegramPostJobID, params)
}

func (a *app) EnqueueUpdateTelegramPost(ctx context.Context, params model.TelegramUpdatePostParams) error {
	return a.enqueueJobToQ(ctx, a.telegramQueue, updateTelegramPostJobID, params)
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
	chat, err := a.TgBotChat(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to get Telegram chat: %w", err)
	}

	handlerIO := a.TgBots.GetHandlerIO(chat.BotID)

	if handlerIO == nil {
		return fmt.Errorf("telegram bot handler IO not found for chat ID %d", chatID)
	}

	_, err = handlerIO.Request(msg)
	if err != nil {
		return fmt.Errorf("failed to send Telegram message: %w", err)
	}

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
