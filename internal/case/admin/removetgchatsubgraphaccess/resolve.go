package removetgchatsubgraphaccess

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
	TgChatSubgraphAccess(ctx context.Context, id int64) (db.TgChatSubgraphAccess, error)
	DeleteTgChatSubgraphAccess(ctx context.Context, id int64) error
}

func Resolve(ctx context.Context, env Env, input model.RemoveTgChatSubgraphAccessInput) (model.RemoveTgChatSubgraphAccessOrErrorPayload, error) {
	// Validate input
	err := ozzo.ValidateStruct(&input,
		ozzo.Field(&input.ID, ozzo.Required),
	)
	if err != nil {
		return model.NewOzzoError(err), nil
	}

	// Check if access exists
	_, err = env.TgChatSubgraphAccess(ctx, input.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &model.ErrorPayload{Message: "Chat subgraph access not found"}, nil
		}
		return nil, fmt.Errorf("failed to get chat subgraph access: %w", err)
	}

	// Delete the access
	err = env.DeleteTgChatSubgraphAccess(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to remove chat subgraph access: %w", err)
	}

	return &model.RemoveTgChatSubgraphAccessPayload{
		DeletedID: input.ID,
	}, nil
}
