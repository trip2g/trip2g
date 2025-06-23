package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"trip2g/internal/case/handletgupdate"
	"trip2g/internal/db"

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

func (a *app) RunTgBots(ctx context.Context) error {
	bots, err := a.ListEnabledTgBots(ctx)
	if err != nil {
		return fmt.Errorf("failed to list enabled tg bots: %w", err)
	}

	var waitGroup sync.WaitGroup

	for _, botConfig := range bots {
		bot, err := tgbotapi.NewBotAPI(botConfig.Token)
		if err != nil {
			return fmt.Errorf("failed to create bot %s: %w", bot.Self.UserName, err)
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

		updateParams := db.UpdateTgBotNameParams{
			Token: botConfig.Token,
			Name:  sql.NullString{Valid: true, String: bot.Self.UserName},
		}

		err = a.UpdateTgBotName(ctx, updateParams)
		if err != nil {
			return fmt.Errorf("failed to update bot name for %s: %w", bot.Self.UserName, err)
		}

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
