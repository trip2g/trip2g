package completetelegramaccountauth

import (
	"context"
	"fmt"
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

type Env interface {
	TelegramAccountCompleteAuth(ctx context.Context, phone, code, password string) (*appmodel.TelegramCompleteAuthResult, error)
	TelegramAccountGetPasswordHint(phone string) string
	InsertTelegramAccount(ctx context.Context, arg db.InsertTelegramAccountParams) (db.TelegramAccount, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Input = model.AdminCompleteTelegramAccountAuthInput
type Payload = model.AdminCompleteTelegramAccountAuthOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	err := ozzo.ValidateStruct(&input,
		ozzo.Field(&input.Phone, ozzo.Required),
		ozzo.Field(&input.Code, ozzo.Required),
	)
	if err != nil {
		return model.NewOzzoError(err), nil
	}

	// Get admin user token for created_by
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	phone := strings.TrimSpace(input.Phone)
	code := strings.TrimSpace(input.Code)
	password := ""
	if input.Password != nil {
		password = strings.TrimSpace(*input.Password)
	}

	result, err := env.TelegramAccountCompleteAuth(ctx, phone, code, password)
	if err != nil {
		// Check if 2FA password is required
		if strings.Contains(err.Error(), "2FA password required") {
			hint := env.TelegramAccountGetPasswordHint(phone)
			return &model.ErrorPayload{
				Message: fmt.Sprintf("2FA password required. Hint: %s", hint),
			}, nil
		}
		return &model.ErrorPayload{Message: fmt.Sprintf("Authentication failed: %s", err.Error())}, nil
	}

	// Insert the account into database
	isPremium := int64(0)
	if result.IsPremium {
		isPremium = 1
	}

	account, err := env.InsertTelegramAccount(ctx, db.InsertTelegramAccountParams{
		Phone:       phone,
		SessionData: result.SessionData,
		DisplayName: result.DisplayName,
		IsPremium:   isPremium,
		ApiID:       int64(result.APIID),
		ApiHash:     result.APIHash,
		CreatedBy:   int64(token.ID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert telegram account: %w", err)
	}

	payload := model.AdminCompleteTelegramAccountAuthPayload{
		Account: &account,
	}

	return &payload, nil
}
