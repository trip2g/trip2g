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
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

type Env interface {
	TelegramAccountCompleteAuth(ctx context.Context, phone, code, password string) (*appmodel.TelegramCompleteAuthResult, error)
	TelegramAccountGetPasswordHint(phone string) string
	TelegramAccountGetAppConfig(ctx context.Context, accountID int64) (string, error)
	GetTelegramAccountByPhone(ctx context.Context, phone string) (db.TelegramAccount, error)
	InsertTelegramAccount(ctx context.Context, arg db.InsertTelegramAccountParams) (db.TelegramAccount, error)
	UpdateTelegramAccount(ctx context.Context, arg db.UpdateTelegramAccountParams) error
	UpdateTelegramAccountAppConfig(ctx context.Context, arg db.UpdateTelegramAccountAppConfigParams) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	EncryptData(plaintext []byte) ([]byte, error)
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

	// Upsert account (update existing or insert new)
	account, err := upsertAccount(ctx, env, phone, result, token.ID)
	if err != nil {
		return nil, err
	}

	// Fetch and save app config
	appConfig, configErr := env.TelegramAccountGetAppConfig(ctx, account.ID)
	if configErr == nil && appConfig != "" {
		err = env.UpdateTelegramAccountAppConfig(ctx, db.UpdateTelegramAccountAppConfigParams{
			AppConfig: appConfig,
			ID:        account.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update app config: %w", err)
		}
	}

	payload := model.AdminCompleteTelegramAccountAuthPayload{
		Account: &account,
	}

	return &payload, nil
}

func upsertAccount(ctx context.Context, env Env, phone string, result *appmodel.TelegramCompleteAuthResult, createdBy int) (db.TelegramAccount, error) {
	isPremium := int64(0)
	if result.IsPremium {
		isPremium = 1
	}

	// Encrypt session data before storing
	encryptedSession, err := env.EncryptData(result.SessionData)
	if err != nil {
		return db.TelegramAccount{}, fmt.Errorf("failed to encrypt session data: %w", err)
	}

	// Check if account with this phone already exists
	existingAccount, err := env.GetTelegramAccountByPhone(ctx, phone)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return db.TelegramAccount{}, fmt.Errorf("failed to check existing account: %w", err)
	}

	// Account exists - update session and enable it
	if err == nil {
		enabled := int64(1)
		updateErr := env.UpdateTelegramAccount(ctx, db.UpdateTelegramAccountParams{
			ID:          existingAccount.ID,
			SessionData: encryptedSession,
			DisplayName: &result.DisplayName,
			IsPremium:   &isPremium,
			ApiID:       ptr.To(int64(result.APIID)),
			ApiHash:     &result.APIHash,
			Enabled:     &enabled,
		})
		if updateErr != nil {
			return db.TelegramAccount{}, fmt.Errorf("failed to update telegram account: %w", updateErr)
		}

		// Fetch updated account
		account, getErr := env.GetTelegramAccountByPhone(ctx, phone)
		if getErr != nil {
			return db.TelegramAccount{}, fmt.Errorf("failed to get updated telegram account: %w", getErr)
		}
		return account, nil
	}

	// Account doesn't exist - insert new
	account, insertErr := env.InsertTelegramAccount(ctx, db.InsertTelegramAccountParams{
		Phone:       phone,
		SessionData: encryptedSession,
		DisplayName: result.DisplayName,
		IsPremium:   isPremium,
		ApiID:       int64(result.APIID),
		ApiHash:     result.APIHash,
		CreatedBy:   int64(createdBy),
	})
	if insertErr != nil {
		return db.TelegramAccount{}, fmt.Errorf("failed to insert telegram account: %w", insertErr)
	}

	return account, nil
}
