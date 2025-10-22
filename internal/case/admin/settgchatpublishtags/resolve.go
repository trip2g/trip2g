package settgchatpublishtags

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
	DeleteTelegramPublishChatsByChatID(ctx context.Context, chatID int64) error
	InsertTelegramPublishChat(ctx context.Context, arg db.InsertTelegramPublishChatParams) error
}

// Input is an alias for SetTgChatPublishTagsInput for cleaner code.
type Input = model.SetTgChatPublishTagsInput

// Payload is an alias for SetTgChatPublishTagsOrErrorPayload for cleaner code.
type Payload = model.SetTgChatPublishTagsOrErrorPayload

// validateRequest validates input and returns ErrorPayload if invalid.
func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.ChatID, validation.Required),
		// SubgraphIds can be empty array (to clear all), so just check it's not nil
		validation.Field(&r.TagIds, validation.NotNil),
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

	// First, delete all existing publishtags for this chat
	err = env.DeleteTelegramPublishChatsByChatID(ctx, input.ChatID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete existing chat subgraph publishtags: %w", err)
	}

	// Then insert all the new publishtags
	for _, tagID := range input.TagIds {
		params := db.InsertTelegramPublishChatParams{
			ChatID:    input.ChatID,
			TagID:     tagID,
			CreatedBy: int64(adminToken.ID),
		}

		insertErr := env.InsertTelegramPublishChat(ctx, params)
		if insertErr != nil {
			// If we fail partway through, the transaction should rollback
			// but since we don't have transaction support in this interface,
			// we'll just return the error
			return nil, fmt.Errorf("failed to insert chat subgraph publishtag %d: %w", tagID, insertErr)
		}
	}

	// Define payload as separate variable
	payload := model.SetTgChatPublishTagsPayload{
		ChatID:  input.ChatID,
		Success: true,
	}

	return &payload, nil
}
