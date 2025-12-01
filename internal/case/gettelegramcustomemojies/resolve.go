package gettelegramcustomemojies

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg gettelegramcustomemojies_test . Env TgBotsInterface

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/tgbots"
)

type TgBotsInterface interface {
	GetBotIDs() []int64
	GetHandlerIO(botID int64) *tgbots.HandlerIO
}

type Env interface {
	ListTelegramCustomEmojies(ctx context.Context, ids []string) ([]db.TelegramCustomEmojy, error)
	InsertTelegramCustomEmoji(ctx context.Context, arg db.InsertTelegramCustomEmojiParams) error
	GetTgBots() TgBotsInterface
}

func Resolve(ctx context.Context, env Env, filter model.TelegramCustomEmojiesFilter) ([]model.TelegramCustomEmoji, error) {
	if len(filter.Ids) == 0 {
		return []model.TelegramCustomEmoji{}, nil
	}

	cachedEmojies, err := env.ListTelegramCustomEmojies(ctx, filter.Ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list cached telegram custom emojies: %w", err)
	}

	cachedMap := make(map[string]db.TelegramCustomEmojy)
	for _, emoji := range cachedEmojies {
		cachedMap[emoji.ID] = emoji
	}

	missingIDs := make([]string, 0)
	for _, id := range filter.Ids {
		if _, exists := cachedMap[id]; !exists {
			missingIDs = append(missingIDs, id)
		}
	}

	if len(missingIDs) > 0 {
		var botID int64
		if filter.BotID != nil {
			botID = *filter.BotID
		} else {
			botIDs := env.GetTgBots().GetBotIDs()
			if len(botIDs) == 0 {
				return nil, fmt.Errorf("no telegram bots available")
			}
			botID = botIDs[0]
		}

		handlerIO := env.GetTgBots().GetHandlerIO(botID)
		if handlerIO == nil {
			return nil, fmt.Errorf("telegram bot %d not found or not running", botID)
		}

		stickers, err := handlerIO.GetCustomEmojiStickers(ctx, missingIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to get custom emoji stickers from telegram: %w", err)
		}

		for _, sticker := range stickers {
			params := db.InsertTelegramCustomEmojiParams{
				ID:         sticker.ID,
				Base64Data: sticker.Base64Data,
			}
			err = env.InsertTelegramCustomEmoji(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to insert telegram custom emoji: %w", err)
			}

			cachedMap[sticker.ID] = db.TelegramCustomEmojy{
				ID:         sticker.ID,
				Base64Data: sticker.Base64Data,
			}
		}
	}

	result := make([]model.TelegramCustomEmoji, 0, len(filter.Ids))
	for _, id := range filter.Ids {
		if emoji, exists := cachedMap[id]; exists {
			result = append(result, model.TelegramCustomEmoji{
				ID:        emoji.ID,
				Base64Uri: emoji.Base64Data,
			})
		}
	}

	return result, nil
}
