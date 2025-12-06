package importtelegramaccountchannel

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	GetTelegramAccountByID(ctx context.Context, id int64) (db.TelegramAccount, error)
	EnqueueImportTelegramChannel(ctx context.Context, params appmodel.ImportTelegramChannelParams) error
}

type Input = model.AdminImportTelegramAccountChannelInput
type Payload = model.AdminImportTelegramAccountChannelOrErrorPayload

func validateRequest(input *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(input,
		ozzo.Field(&input.AccountID, ozzo.Required),
		ozzo.Field(&input.ChannelID, ozzo.Required),
		ozzo.Field(&input.BasePath, ozzo.Required),
	))
}

// sanitizeBasePath cleans and validates the base path to prevent path traversal.
func sanitizeBasePath(basePath string) (string, error) {
	// Clean the path
	cleaned := filepath.Clean(basePath)

	// Check for path traversal attempts
	if strings.Contains(cleaned, "..") {
		return "", errors.New("invalid path: contains '..'")
	}

	// Remove leading slash for consistency
	cleaned = strings.TrimPrefix(cleaned, "/")

	// Don't allow empty path after cleaning
	if cleaned == "" || cleaned == "." {
		return "", errors.New("invalid path: empty after cleaning")
	}

	return cleaned, nil
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	// Validate admin access
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	// Validate input
	if errPayload := validateRequest(&input); errPayload != nil {
		return errPayload, nil
	}

	// Sanitize basePath to prevent path traversal
	sanitizedPath, err := sanitizeBasePath(input.BasePath)
	if err != nil {
		return &model.ErrorPayload{Message: err.Error()}, nil //nolint:nilerr // intentional: return user-visible error
	}

	// Verify account exists
	_, err = env.GetTelegramAccountByID(ctx, input.AccountID)
	if err != nil {
		return &model.ErrorPayload{Message: "Telegram account not found"}, nil //nolint:nilerr // intentional: return user-visible error
	}

	// Apply defaults: withMedia: false, skipExists: true
	withMedia := false
	if input.WithMedia != nil {
		withMedia = *input.WithMedia
	}
	skipExists := true
	if input.SkipExists != nil {
		skipExists = *input.SkipExists
	}

	// Enqueue background job with sanitized path
	params := appmodel.ImportTelegramChannelParams{
		AccountID:  input.AccountID,
		ChannelID:  input.ChannelID,
		BasePath:   sanitizedPath,
		WithMedia:  withMedia,
		SkipExists: skipExists,
	}

	err = env.EnqueueImportTelegramChannel(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue import job: %w", err)
	}

	payload := model.AdminImportTelegramAccountChannelPayload{
		Success: true,
	}

	return &payload, nil
}
