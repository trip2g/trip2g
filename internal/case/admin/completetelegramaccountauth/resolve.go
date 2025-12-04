package completetelegramaccountauth

import (
	"context"
	"database/sql"
	"errors"
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
	GetTelegramAccountByPhone(ctx context.Context, phone string) (db.TelegramAccount, error)
	InsertTelegramAccount(ctx context.Context, arg db.InsertTelegramAccountParams) (db.TelegramAccount, error)
	UpdateTelegramAccount(ctx context.Context, arg db.UpdateTelegramAccountParams) error
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

	isPremium := int64(0)
	if result.IsPremium {
		isPremium = 1
	}

	// Check if account with this phone already exists
	existingAccount, err := env.GetTelegramAccountByPhone(ctx, phone)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check existing account: %w", err)
	}

	var account db.TelegramAccount

	if err == nil {
		// Account exists - update session and enable it
		enabled := int64(1)
		err = env.UpdateTelegramAccount(ctx, db.UpdateTelegramAccountParams{
			ID:          existingAccount.ID,
			SessionData: result.SessionData,
			DisplayName: sql.NullString{String: result.DisplayName, Valid: true},
			IsPremium:   sql.NullInt64{Int64: isPremium, Valid: true},
			ApiID:       sql.NullInt64{Int64: int64(result.APIID), Valid: true},
			ApiHash:     sql.NullString{String: result.APIHash, Valid: true},
			Enabled:     sql.NullInt64{Int64: enabled, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update telegram account: %w", err)
		}

		// Fetch updated account
		account, err = env.GetTelegramAccountByPhone(ctx, phone)
		if err != nil {
			return nil, fmt.Errorf("failed to get updated telegram account: %w", err)
		}
	} else {
		// Account doesn't exist - insert new
		account, err = env.InsertTelegramAccount(ctx, db.InsertTelegramAccountParams{
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
	}

	payload := model.AdminCompleteTelegramAccountAuthPayload{
		Account: &account,
	}

	return &payload, nil
}
