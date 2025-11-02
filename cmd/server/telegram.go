package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"trip2g/internal/case/backjob/sendtelegrampost"
	"trip2g/internal/case/backjob/updatetelegrampost"
	"trip2g/internal/case/handletgupdate"
	"trip2g/internal/model"
	"trip2g/internal/tgbots"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"maragu.dev/goqite/jobs"
)

const sendTelegramPostJobID = "send_message"
const updateTelegramPostJobID = "update_message"
const sendPublishPostJobID = "send_publish_post"

type sendPublishPostParams struct {
	NotePathID int64 `json:"note_path_id"`
	Instant    bool  `json:"instant"`
}

func (a *app) initTelegramDeps(ctx context.Context) error {
	appQ := a.createQueue(ctx, "tg_jobs", jobs.NewRunnerOpts{
		Limit:        1,
		PollInterval: time.Second,
	})

	a.telegramQueue = appQ

	jobTimeout := time.Minute

	a.telegramQueue.runner.Register(sendTelegramPostJobID, func(ctx context.Context, m []byte) error {
		var params model.TelegramSendPostParams

		err := json.Unmarshal(m, &params)
		if err != nil {
			return fmt.Errorf("failed to unmarshal send_telegram_post params: %w", err)
		}

		// independent timeout context from stop cancelations
		jobCtx, cancel := context.WithTimeout(context.Background(), jobTimeout)
		defer cancel()

		err = sendtelegrampost.Resolve(jobCtx, a, params)
		if err != nil {
			shouldRetry, delay := handleTelegramRateLimit(err)
			if shouldRetry {
				a.log.Info("telegram rate limit hit, retrying after delay",
					"delay", delay,
					"job", sendTelegramPostJobID,
				)
				time.Sleep(delay)
				err = sendtelegrampost.Resolve(ctx, a, params)
			}

			if err != nil {
				return err
			}
		}

		return nil
	})

	a.telegramQueue.runner.Register(updateTelegramPostJobID, func(ctx context.Context, m []byte) error {
		var params model.TelegramUpdatePostParams

		err := json.Unmarshal(m, &params)
		if err != nil {
			return fmt.Errorf("failed to unmarshal update_telegram_post params: %w", err)
		}

		// independent timeout context from stop cancelations
		jobCtx, cancel := context.WithTimeout(context.Background(), jobTimeout)
		defer cancel()

		err = updatetelegrampost.Resolve(jobCtx, a, params)
		if err != nil {
			shouldRetry, delay := handleTelegramRateLimit(err)
			if shouldRetry {
				a.log.Info("telegram rate limit hit, retrying after delay",
					"delay", delay,
					"job", updateTelegramPostJobID,
				)
				time.Sleep(delay)
				err = updatetelegrampost.Resolve(ctx, a, params)
			}

			if err != nil {
				return err
			}
		}

		return nil
	})

	a.telegramQueue.runner.Register(sendPublishPostJobID, func(ctx context.Context, m []byte) error {
		var params sendPublishPostParams

		err := json.Unmarshal(m, &params)
		if err != nil {
			return fmt.Errorf("failed to unmarshal send_publish_post params: %w", err)
		}

		// independent timeout context from stop cancelations
		jobCtx, cancel := context.WithTimeout(context.Background(), jobTimeout)
		defer cancel()

		return a.SendTelegramPublishPostWithTx(jobCtx, params.NotePathID, params.Instant)
	})

	appQ.start() // after register all handlers

	return a.initTelegramBots(ctx)
}

func (a *app) EnqueueSendTelegramPost(ctx context.Context, params model.TelegramSendPostParams) error {
	return a.enqueueJobToQ(ctx, a.telegramQueue, sendTelegramPostJobID, params)
}

func (a *app) EnqueueUpdateTelegramPost(ctx context.Context, params model.TelegramUpdatePostParams) error {
	return a.enqueueJobToQ(ctx, a.telegramQueue, updateTelegramPostJobID, params)
}

func (a *app) EnqueueSendTelegramPublishPost(ctx context.Context, notePathID int64, instant bool) error {
	params := sendPublishPostParams{NotePathID: notePathID, Instant: instant}
	return a.enqueueJobToQ(ctx, a.telegramQueue, sendPublishPostJobID, params)
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

	a.log.Debug("telegram request success", "chat_id", chatID, "ok", resp.Ok, "description", resp.Description)

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

// handleTelegramRateLimit checks if error is "Too Many Requests" and returns retry delay.
func handleTelegramRateLimit(err error) (bool, time.Duration) {
	if err == nil {
		return false, 0
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Too Many Requests") {
		return false, 0
	}

	// Try to parse "retry after X" from error message
	re := regexp.MustCompile(`retry after (\d+)`)
	matches := re.FindStringSubmatch(errMsg)

	seconds := 10 // default delay
	if len(matches) > 1 {
		parsed, parseErr := strconv.Atoi(matches[1])
		if parseErr == nil {
			seconds = parsed
		}
	}

	// Add +1 second to the delay
	return true, time.Duration(seconds+1) * time.Second
}
