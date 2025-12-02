package gettelegramcustomemojies

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg gettelegramcustomemojies_test . Env

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

type Env interface {
	ListTelegramCustomEmojies(ctx context.Context, ids []string) ([]db.TelegramCustomEmojy, error)
	InsertTelegramCustomEmoji(ctx context.Context, arg db.InsertTelegramCustomEmojiParams) error
	GetTelegramCustomEmojiStickers(ctx context.Context, emojiIDs []string) ([]appmodel.CustomEmojiSticker, error)
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
		err = fetchAndCacheMissingEmojies(ctx, env, missingIDs, cachedMap)
		if err != nil {
			return nil, err
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

func fetchAndCacheMissingEmojies(ctx context.Context, env Env, missingIDs []string, cachedMap map[string]db.TelegramCustomEmojy) error {
	stickers, err := env.GetTelegramCustomEmojiStickers(ctx, missingIDs)
	if err != nil {
		return fmt.Errorf("failed to get custom emoji stickers from telegram: %w", err)
	}

	for _, sticker := range stickers {
		params := db.InsertTelegramCustomEmojiParams{
			ID:         sticker.ID,
			Base64Data: sticker.Base64Data,
		}

		insertErr := env.InsertTelegramCustomEmoji(ctx, params)
		if insertErr != nil {
			return fmt.Errorf("failed to insert telegram custom emoji: %w", insertErr)
		}

		cachedMap[sticker.ID] = db.TelegramCustomEmojy{
			ID:         sticker.ID,
			Base64Data: sticker.Base64Data,
		}
	}

	return nil
}
