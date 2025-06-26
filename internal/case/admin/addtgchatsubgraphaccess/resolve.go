package addtgchatsubgraphaccess

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

type Env interface {
	GetTgBotChat(ctx context.Context, id int64) (db.TgBotChat, error)
	SubgraphByID(ctx context.Context, id int64) (db.Subgraph, error)
	InsertTgChatSubgraphAccess(ctx context.Context, arg db.InsertTgChatSubgraphAccessParams) (db.TgChatSubgraphAccess, error)
}

func Resolve(ctx context.Context, env Env, input model.AddTgChatSubgraphAccessInput) (model.AddTgChatSubgraphAccessOrErrorPayload, error) {
	// Validate input
	err := ozzo.ValidateStruct(&input,
		ozzo.Field(&input.ChatID, ozzo.Required),
		ozzo.Field(&input.SubgraphID, ozzo.Required),
	)
	if err != nil {
		return model.NewOzzoError(err), nil
	}

	// Check if chat exists
	_, err = env.GetTgBotChat(ctx, input.ChatID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &model.ErrorPayload{Message: "Chat not found"}, nil
		}
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	// Check if subgraph exists
	_, err = env.SubgraphByID(ctx, input.SubgraphID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &model.ErrorPayload{Message: "Subgraph not found"}, nil
		}
		return nil, fmt.Errorf("failed to get subgraph: %w", err)
	}

	// Create the access
	access, err := env.InsertTgChatSubgraphAccess(ctx, db.InsertTgChatSubgraphAccessParams{
		ChatID:     input.ChatID,
		SubgraphID: input.SubgraphID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add chat subgraph access: %w", err)
	}

	return &model.AddTgChatSubgraphAccessPayload{
		ChatSubgraphAccess: &access,
	}, nil
}
