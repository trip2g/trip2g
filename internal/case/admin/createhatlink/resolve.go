package createhatlink

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
)

// The minted URL is a bearer credential that signs its holder in without a
// mail round-trip, so its lifetime is kept short: defaultExpiryMinutes mirrors
// loginLinkExpiry in cmd/server/loginlink.go, maxExpiryMinutes is the hard cap
// an admin cannot raise.
const (
	defaultExpiryMinutes = 5
	maxExpiryMinutes     = 60
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	GenerateHotAuthTokenWithTTL(ctx context.Context, data appmodel.HotAuthToken, ttl time.Duration) (string, error)
	PublicURL() string
	AuditLogger() logger.Logger
}

type Input = model.CreateHatLinkInput
type Payload = model.CreateHatLinkOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	errPayload := model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.Email, ozzo.Required, is.Email),
	))
	if errPayload != nil {
		return errPayload
	}

	if r.ExpiresInMinutes != nil && (*r.ExpiresInMinutes < 1 || *r.ExpiresInMinutes > maxExpiryMinutes) {
		return model.NewFieldError("expiresInMinutes", fmt.Sprintf("must be between 1 and %d", maxExpiryMinutes))
	}

	return nil
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	publicURL := strings.TrimRight(env.PublicURL(), "/")
	if publicURL == "" {
		return nil, errors.New("public URL is not configured")
	}

	minutes := defaultExpiryMinutes
	if input.ExpiresInMinutes != nil {
		minutes = int(*input.ExpiresInMinutes)
	}
	expiresIn := time.Duration(minutes) * time.Minute

	adminEnter := input.AdminEnter != nil && *input.AdminEnter

	hatToken, err := env.GenerateHotAuthTokenWithTTL(ctx, appmodel.HotAuthToken{
		Email:      input.Email,
		AdminEnter: adminEnter,
	}, expiresIn)
	if err != nil {
		return nil, fmt.Errorf("failed to mint hot auth token: %w", err)
	}

	// Audit the request, never the token or the URL — they are usable credentials.
	env.AuditLogger().Info("create hat link",
		"actorUserID", token.ID,
		"email", input.Email,
		"adminEnter", adminEnter,
		"expiresInMinutes", minutes,
	)

	payload := model.CreateHatLinkPayload{
		URL:       publicURL + "/_system/hat?token=" + hatToken,
		ExpiresAt: time.Now().Add(expiresIn),
	}

	return &payload, nil
}
