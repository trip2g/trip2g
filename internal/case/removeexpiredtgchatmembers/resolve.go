package removeexpiredtgchatmembers

import (
	"context"
	"database/sql"
	"fmt"
	"trip2g/internal/db"
)

type Env interface {
	ListActiveUserSubgraphs(ctx context.Context, userID int64) ([]string, error)
	ListTgBotChatSubgraphAccesses(ctx context.Context, userID sql.NullInt64) ([]db.ListTgBotChatSubgraphAccessesRow, error)
}

func Resolve(ctx context.Context, env Env, userID *int64) error {
	accesses := map[int64][]*db.ListTgBotChatSubgraphAccessesRow{}

	userIDParam := sql.NullInt64{}

	if userID != nil {
		userIDParam.Valid = true
		userIDParam.Int64 = *userID
	}

	rows, err := env.ListTgBotChatSubgraphAccesses(ctx, userIDParam)
	if err != nil {
		return fmt.Errorf("failed to list tg bot chat subgraph accesses: %w", err)
	}

	for _, row := range rows {
		userID := row.TgBotChatSubgraphAccess.UserID
		accesses[userID] = append(accesses[userID], &row)
	}

	return nil
}

// func processUser(ctx context.Context, env Env, userID int64) error {
// 	accesses, err := env.ListTgBotChatSubgraphAccessByUserID(ctx, userID)
// 	if err != nil {
// 		return fmt.Errorf("failed to list tg bot chat subgraph access: %w", err)
// 	}
//
// 	subgraphs, err := env.ListActiveUserSubgraphs(ctx, userID)
// 	if err != nil {
// 		return fmt.Errorf("failed to list active user subgraphs: %w", err)
// 	}
//
// 	return true, nil
// }
