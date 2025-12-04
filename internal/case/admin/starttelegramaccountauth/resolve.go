package starttelegramaccountauth

import (
	"context"
	"fmt"
	"strings"

	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

type Env interface {
	TelegramAccountStartAuth(ctx context.Context, phone string, apiID int, apiHash string) (*appmodel.TelegramStartAuthResult, error)
}

type Input = model.AdminStartTelegramAccountAuthInput
type Payload = model.AdminStartTelegramAccountAuthOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	err := ozzo.ValidateStruct(&input,
		ozzo.Field(&input.Phone, ozzo.Required),
		ozzo.Field(&input.APIID, ozzo.Required),
		ozzo.Field(&input.APIHash, ozzo.Required),
	)
	if err != nil {
		return model.NewOzzoError(err), nil
	}

	phone := strings.TrimSpace(input.Phone)
	apiHash := strings.TrimSpace(input.APIHash)

	// Start auth - if phone already exists, completeTelegramAccountAuth will update the existing account
	result, err := env.TelegramAccountStartAuth(ctx, phone, int(input.APIID), apiHash)
	if err != nil {
		return &model.ErrorPayload{Message: fmt.Sprintf("Failed to start auth: %s", err.Error())}, nil
	}

	state := mapAuthState(result.State)
	var passwordHint *string
	if result.PasswordHint != "" {
		passwordHint = &result.PasswordHint
	}

	payload := model.AdminStartTelegramAccountAuthPayload{
		AuthState: &model.AdminTelegramAccountAuthState{
			Phone:        result.Phone,
			State:        state,
			PasswordHint: passwordHint,
		},
	}

	return &payload, nil
}

func mapAuthState(state appmodel.TelegramAuthState) model.AdminTelegramAccountAuthStateEnum {
	switch state {
	case appmodel.TelegramAuthStateWaitingForCode:
		return model.AdminTelegramAccountAuthStateEnumWaitingForCode
	case appmodel.TelegramAuthStateWaitingForPassword:
		return model.AdminTelegramAccountAuthStateEnumWaitingForPassword
	case appmodel.TelegramAuthStateAuthorized:
		return model.AdminTelegramAccountAuthStateEnumAuthorized
	case appmodel.TelegramAuthStateError:
		return model.AdminTelegramAccountAuthStateEnumError
	}
	return model.AdminTelegramAccountAuthStateEnumError
}
