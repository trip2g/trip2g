package settgchatsubgraphinvites

import (
	"context"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	DeleteTgChatSubgraphInvitesByChatID(ctx context.Context, chatID int64) error
	InsertTgChatSubgraphInvite(ctx context.Context, arg db.InsertTgChatSubgraphInviteParams) (db.TgBotChatSubgraphInvite, error)
}

// Input is an alias for SetTgChatSubgraphInvitesInput for cleaner code.
type Input = model.SetTgChatSubgraphInvitesInput

// Payload is an alias for SetTgChatSubgraphInvitesOrErrorPayload for cleaner code.
type Payload = model.SetTgChatSubgraphInvitesOrErrorPayload

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

	// Get admin token from context for created_by field
	adminToken, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin user token: %w", err)
	}

	// First, delete all existing invites for this chat
	err = env.DeleteTgChatSubgraphInvitesByChatID(ctx, input.ChatID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete existing chat subgraph invites: %w", err)
	}

	// Then insert all the new invites
	for _, subgraphID := range input.SubgraphIds {
		params := db.InsertTgChatSubgraphInviteParams{
			ChatID:     input.ChatID,
			SubgraphID: subgraphID,
			CreatedBy:  int64(adminToken.ID),
		}

		_, insertErr := env.InsertTgChatSubgraphInvite(ctx, params)
		if insertErr != nil {
			// If we fail partway through, the transaction should rollback
			// but since we don't have transaction support in this interface,
			// we'll just return the error
			return nil, fmt.Errorf("failed to insert chat subgraph invite %d: %w", subgraphID, insertErr)
		}
	}

	// Define payload as separate variable
	payload := model.SetTgChatSubgraphInvitesPayload{
		ChatID:  input.ChatID,
		Success: true,
	}

	return &payload, nil
}
