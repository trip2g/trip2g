package removeexpiredtgchatmembers

import (
	"context"
	"fmt"
	"trip2g/internal/db"
)

type Env interface {
	ListActiveUserSubgraphs(ctx context.Context, userID int64) ([]string, error)
	ListTgBotChatSubgraphAccessByUserID(ctx context.Context, userID int64) (db.TgBotChatSubgraphAccess, error)
}

func Resolve(ctx context.Context, env Env, userID *int64) error {
	return nil
}

func processUser(ctx context.Context, env Env, userID int64) error {
	subgraphs, err := env.ListActiveUserSubgraphs(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to list active user subgraphs: %w", err)
	}

	accesses, err := env.ListTgBotChatSubgraphAccessByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to list tg bot chat subgraph access: %w", err)
	}

	return true, nil
}
