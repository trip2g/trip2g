package settelegramaccountchatpublishinstanttags

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
	DeleteTelegramPublishAccountInstantChatsByAccountAndChatID(
		ctx context.Context,
		arg db.DeleteTelegramPublishAccountInstantChatsByAccountAndChatIDParams,
	) error
	InsertTelegramPublishAccountInstantChat(
		ctx context.Context,
		arg db.InsertTelegramPublishAccountInstantChatParams,
	) error
}

type Input = model.AdminSetTelegramAccountChatPublishInstantTagsInput
type Payload = model.AdminSetTelegramAccountChatPublishInstantTagsOrErrorPayload

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

	// First, delete all existing instant tags for this account+chat
	err = env.DeleteTelegramPublishAccountInstantChatsByAccountAndChatID(ctx, db.DeleteTelegramPublishAccountInstantChatsByAccountAndChatIDParams{
		AccountID:      input.AccountID,
		TelegramChatID: telegramChatID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to delete existing chat instant tags: %w", err)
	}

	// Then insert all the new tags
	for _, tagID := range input.TagIds {
		params := db.InsertTelegramPublishAccountInstantChatParams{
			AccountID:      input.AccountID,
			TelegramChatID: telegramChatID,
			TagID:          tagID,
			CreatedBy:      int64(adminToken.ID),
		}

		insertErr := env.InsertTelegramPublishAccountInstantChat(ctx, params)
		if insertErr != nil {
			return nil, fmt.Errorf("failed to insert chat instant tag %d: %w", tagID, insertErr)
		}
	}

	payload := model.AdminSetTelegramAccountChatPublishInstantTagsPayload{
		Success: true,
	}

	return &payload, nil
}
