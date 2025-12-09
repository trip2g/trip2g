package updatetelegrampublishposts

import (
	"context"
	"fmt"
	"trip2g/internal/logger"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg updatetelegrampublishposts_test . Env

type Env interface {
	Logger() logger.Logger
	// Bot posts
	ListDistinctChatIDsFromSentMessages(ctx context.Context) ([]int64, error)
	EnqueueUpdateAllChatTelegramPublishPosts(ctx context.Context, chatID int64) error
	// Account posts
	ListDistinctAccountIDsFromSentAccountMessages(ctx context.Context) ([]int64, error)
	EnqueueUpdateAllAccountTelegramPublishPosts(ctx context.Context, accountID int64) error
}

type ResultChat struct {
	ChatID int64 `json:"chat_id"`
	Error  error `json:"error,omitempty"`
}

type ResultAccount struct {
	AccountID int64 `json:"account_id"`
	Error     error `json:"error,omitempty"`
}

type Result struct {
	Chats    []ResultChat    `json:"chats"`
	Accounts []ResultAccount `json:"accounts"`
}

func Resolve(ctx context.Context, env Env) (any, error) {
	logger := logger.WithPrefix(env.Logger(), "updatetelegrampublishposts:")

	res := Result{}

	// Bot posts
	chatIDs, err := env.ListDistinctChatIDsFromSentMessages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ListDistinctChatIDsFromSentMessages: %w", err)
	}

	logger.Debug("bot chats found", "count", len(chatIDs))

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

	// Account posts
	accountIDs, err := env.ListDistinctAccountIDsFromSentAccountMessages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ListDistinctAccountIDsFromSentAccountMessages: %w", err)
	}

	logger.Debug("accounts found", "count", len(accountIDs))

	for _, accountID := range accountIDs {
		enqueueErr := env.EnqueueUpdateAllAccountTelegramPublishPosts(ctx, accountID)

		res.Accounts = append(res.Accounts, ResultAccount{
			AccountID: accountID,
			Error:     enqueueErr,
		})

		if enqueueErr != nil {
			logger.Error("failed to EnqueueUpdateAllAccountTelegramPublishPosts", "account_id", accountID, "error", enqueueErr)
		}
	}

	logger.Info("completed enqueueing updates", "total_chats", len(chatIDs), "total_accounts", len(accountIDs))
	return res, nil
}
