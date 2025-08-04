package settgchatsubgraphs

import (
	"context"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

type Env interface {
	DeleteTgChatSubgraphAccessesByChatID(ctx context.Context, chatID int64) error
	InsertTgChatSubgraphAccess(ctx context.Context, arg db.InsertTgChatSubgraphAccessParams) (db.TgChatSubgraphAccess, error)
}

// Input is an alias for SetTgChatSubgraphsInput for cleaner code.
type Input = model.SetTgChatSubgraphsInput

// Payload is an alias for SetTgChatSubgraphsOrErrorPayload for cleaner code.
type Payload = model.SetTgChatSubgraphsOrErrorPayload

// validateRequest validates input and returns ErrorPayload if invalid.
func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.ChatID, validation.Required),
		// SubgraphIds can be empty array (to clear all), so just check it's not nil
		validation.Field(&r.SubgraphIds, validation.NotNil),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	// Always validate input first
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil // User-visible errors go in ErrorPayload
	}

	// First, delete all existing subgraphs for this chat
	err := env.DeleteTgChatSubgraphAccessesByChatID(ctx, input.ChatID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete existing chat subgraphs: %w", err)
	}

	// Then insert all the new subgraphs
	for _, subgraphID := range input.SubgraphIds {
		params := db.InsertTgChatSubgraphAccessParams{
			ChatID:     input.ChatID,
			SubgraphID: subgraphID,
		}

		_, insertErr := env.InsertTgChatSubgraphAccess(ctx, params)
		if insertErr != nil {
			// If we fail partway through, the transaction should rollback
			// but since we don't have transaction support in this interface,
			// we'll just return the error
			return nil, fmt.Errorf("failed to insert chat subgraph %d: %w", subgraphID, insertErr)
		}
	}

	// Define payload as separate variable
	payload := model.SetTgChatSubgraphsPayload{
		ChatID:  input.ChatID,
		Success: true,
	}

	return &payload, nil
}
