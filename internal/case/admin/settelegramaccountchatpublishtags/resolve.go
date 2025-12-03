package settelegramaccountchatpublishtags

import (
	"context"
	"fmt"
	"strconv"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	DeleteTelegramPublishAccountChatsByAccountAndChatID(ctx context.Context, arg db.DeleteTelegramPublishAccountChatsByAccountAndChatIDParams) error
	InsertTelegramPublishAccountChat(ctx context.Context, arg db.InsertTelegramPublishAccountChatParams) error
}

type Input = model.AdminSetTelegramAccountChatPublishTagsInput
type Payload = model.AdminSetTelegramAccountChatPublishTagsOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.AccountID, validation.Required),
		validation.Field(&r.TelegramChatID, validation.Required),
		validation.Field(&r.TagIds, validation.NotNil),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	// Get admin token from context for created_by field
	adminToken, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin user token: %w", err)
	}

	// Parse telegram chat ID
	telegramChatID, err := strconv.ParseInt(input.TelegramChatID, 10, 64)
	if err != nil {
		return &model.ErrorPayload{Message: "Invalid telegram chat ID"}, nil //nolint:nilerr // ErrorPayload pattern
	}

	// First, delete all existing publish tags for this account+chat
	err = env.DeleteTelegramPublishAccountChatsByAccountAndChatID(ctx, db.DeleteTelegramPublishAccountChatsByAccountAndChatIDParams{
		AccountID:      input.AccountID,
		TelegramChatID: telegramChatID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to delete existing chat publish tags: %w", err)
	}

	// Then insert all the new tags
	for _, tagID := range input.TagIds {
		params := db.InsertTelegramPublishAccountChatParams{
			AccountID:      input.AccountID,
			TelegramChatID: telegramChatID,
			TagID:          tagID,
			CreatedBy:      int64(adminToken.ID),
		}

		insertErr := env.InsertTelegramPublishAccountChat(ctx, params)
		if insertErr != nil {
			return nil, fmt.Errorf("failed to insert chat publish tag %d: %w", tagID, insertErr)
		}
	}

	payload := model.AdminSetTelegramAccountChatPublishTagsPayload{
		AccountID:      input.AccountID,
		TelegramChatID: telegramChatID,
		Success:        true,
	}

	return &payload, nil
}
