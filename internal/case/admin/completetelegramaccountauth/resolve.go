package completetelegramaccountauth

import (
	"context"
	"fmt"
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/tgtd"
	"trip2g/internal/usertoken"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

type Env interface {
	TelegramAuthManager() *tgtd.AuthManager
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

	authManager := env.TelegramAuthManager()

	// Get API credentials from pending auth
	apiID, apiHash, ok := authManager.GetPendingAuthAPICredentials(input.Phone)
	if !ok {
		return &model.ErrorPayload{Message: "No pending authentication found for this phone"}, nil
	}

	// Complete auth
	password := ""
	if input.Password != nil {
		password = *input.Password
	}

	result, err := authManager.CompleteAuth(ctx, input.Phone, input.Code, password)
	if err != nil {
		// Check if 2FA password is required
		if strings.Contains(err.Error(), "2FA password required") {
			pending := authManager.GetPendingAuth(input.Phone)
			if pending != nil {
				var passwordHint *string
				if pending.PasswordHint != "" {
					passwordHint = &pending.PasswordHint
				}
				return &model.ErrorPayload{
					Message: fmt.Sprintf("2FA password required. Hint: %s", pointerOrEmpty(passwordHint)),
				}, nil
			}
		}
		return &model.ErrorPayload{Message: fmt.Sprintf("Authentication failed: %s", err.Error())}, nil
	}

	// Insert the account into database
	isPremium := int64(0)
	if result.IsPremium {
		isPremium = 1
	}

	account, err := env.InsertTelegramAccount(ctx, db.InsertTelegramAccountParams{
		Phone:       input.Phone,
		SessionData: result.SessionData,
		DisplayName: result.DisplayName,
		IsPremium:   isPremium,
		ApiID:       int64(apiID),
		ApiHash:     apiHash,
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

func pointerOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
