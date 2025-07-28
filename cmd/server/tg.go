package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"trip2g/internal/case/handletgupdate"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type botEnv struct {
	*app

	bot *tgbotapi.BotAPI

	id int64
}

func (be *botEnv) Send(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
	return be.bot.Send(msg)
}

func (be *botEnv) Request(msg tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	return be.bot.Request(msg)
}

func (be *botEnv) BotID() int64 {
	return be.id
}

func (be *botEnv) GetChatMemberStatus(ctx context.Context, chatID, userID int64) (string, error) {
	getChatMemberConfig := tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: chatID,
			UserID: userID,
		},
	}

	resp, err := be.Request(getChatMemberConfig)
	if err != nil {
		return "", fmt.Errorf("failed to request chat member info: %w", err)
	}

	if !resp.Ok {
		return "", fmt.Errorf("telegram API error: %s", resp.Description)
	}

	var chatMember tgbotapi.ChatMember
	err = json.Unmarshal(resp.Result, &chatMember)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal chat member response: %w", err)
	}

	return chatMember.Status, nil
}

func (a *app) RunTgBots(ctx context.Context) error {
	bots, queryErr := a.ListEnabledTgBots(ctx)
	if queryErr != nil {
		return fmt.Errorf("failed to list enabled tg bots: %w", queryErr)
	}

	var waitGroup sync.WaitGroup

	for _, botConfig := range bots {
		bot, err := tgbotapi.NewBotAPI(botConfig.Token)
		if err != nil {
			return fmt.Errorf("failed to create bot %+v: %w", bot.Self, err)
		}

		if a.config.DevMode {
			bot.Debug = true
		}

		be := &botEnv{
			app: a,
			bot: bot,
			id:  botConfig.ID,
		}

		a.log.Info("Starting bot", "name", bot.Self.UserName)

		updateConfig := tgbotapi.NewUpdate(0)
		updateConfig.Timeout = 60

		updates := bot.GetUpdatesChan(updateConfig)

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			for {
				select {
				case <-ctx.Done():
					return

				case update := <-updates:
					handleErr := handletgupdate.Resolve(ctx, be, update)
					if handleErr != nil {
						a.log.Error("Error handling update", "update", update, "error", handleErr)
					}
				}
			}
		}()
	}

	waitGroup.Wait()

	return nil
}
