package generatetgattachcode

import (
	"context"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentUserToken(ctx context.Context) (*usertoken.Data, error)
	DeleteTgAttachCodesByUser(ctx context.Context, userID int64) error
	InsertTgAttachCode(ctx context.Context, arg db.InsertTgAttachCodeParams) error
	TgBot(ctx context.Context, id int64) (db.TgBot, error)
	GenerateTgAttachCode() string
	BotStartLink(botID int64, param string) (string, error)
}

// Type aliases for cleaner code.
type Input = model.GenerateTgAttachCodeInput
type Payload = model.GenerateTgAttachCodeOrErrorPayload

// validateRequest validates input and returns ErrorPayload if invalid.
func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.BotID, ozzo.Required, ozzo.Min(int64(1))),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	// Always validate input first
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil // User-visible errors go in ErrorPayload
	}

	// Get the current user from the token
	userToken, err := env.CurrentUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	if userToken == nil {
		return &model.ErrorPayload{Message: "Authentication required"}, nil
	}

	// Verify the bot exists
	_, err = env.TgBot(ctx, input.BotID)
	if err != nil {
		if db.IsNoFound(err) {
			return &model.ErrorPayload{Message: "Bot not found"}, nil
		}
		return nil, fmt.Errorf("failed to get bot: %w", err)
	}

	// Delete any existing attach codes for this user
	err = env.DeleteTgAttachCodesByUser(ctx, int64(userToken.ID))
	if err != nil {
		return nil, fmt.Errorf("failed to delete existing attach codes: %w", err)
	}

	// Generate random 8-character code via environment
	code := env.GenerateTgAttachCode()

	// Insert the attach code
	params := db.InsertTgAttachCodeParams{
		UserID: int64(userToken.ID),
		BotID:  input.BotID,
		Code:   code,
	}

	err = env.InsertTgAttachCode(ctx, params)
	if err != nil {
		if db.IsUniqueViolation(err) {
			// Code collision is very rare but possible, retry once
			code = env.GenerateTgAttachCode()
			params.Code = code
			err = env.InsertTgAttachCode(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to insert attach code on retry: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to insert attach code: %w", err)
		}
	}

	// Generate the Telegram URL using the bot's method
	url, err := env.BotStartLink(input.BotID, fmt.Sprintf("attach_%s", code))
	if err != nil {
		return nil, fmt.Errorf("failed to generate bot start link: %w", err)
	}

	// Define payload as separate variable
	payload := model.GenerateTgAttachCodePayload{
		Code: code,
		URL:  url,
	}

	return &payload, nil
}
