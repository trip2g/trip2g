package updatetelegrampublishposts

import (
	"context"
	"fmt"
	"trip2g/internal/logger"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg updatetelegrampublishposts_test . Env

type Env interface {
	Logger() logger.Logger
	ListDistinctChatIDsFromSentMessages(ctx context.Context) ([]int64, error)
	EnqueueUpdateAllChatTelegramPublishPosts(ctx context.Context, chatID int64) error
}

type ResultChat struct {
	ChatID int64 `json:"chat_id"`
	Error  error `json:"error,omitempty"`
}

type Result struct {
	Chats []ResultChat `json:"chats"`
}

func Resolve(ctx context.Context, env Env) (any, error) {
	logger := logger.WithPrefix(env.Logger(), "updatetelegrampublishposts:")

	chatIDs, err := env.ListDistinctChatIDsFromSentMessages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ListDistinctChatIDsFromSentMessages: %w", err)
	}

	res := Result{}

	logger.Debug("chats found", "count", len(chatIDs))

	for _, chatID := range chatIDs {
		enqueueErr := env.EnqueueUpdateAllChatTelegramPublishPosts(ctx, chatID)

		res.Chats = append(res.Chats, ResultChat{
			ChatID: chatID,
			Error:  enqueueErr,
		})

		if enqueueErr != nil {
			logger.Error("failed to EnqueueUpdateAllChatTelegramPublishPosts", "chat_id", chatID, "error", enqueueErr)
		}
	}

	logger.Info("completed enqueueing chat updates", "total_chats", len(chatIDs))
	return res, nil
}
