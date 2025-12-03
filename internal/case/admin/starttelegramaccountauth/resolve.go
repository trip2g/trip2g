package starttelegramaccountauth

import (
	"context"
	"fmt"

	"trip2g/internal/graph/model"
	"trip2g/internal/tgtd"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

type Env interface {
	TelegramAuthManager() *tgtd.AuthManager
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

	authManager := env.TelegramAuthManager()

	pendingAuth, err := authManager.StartAuth(ctx, input.Phone, int(input.APIID), input.APIHash)
	if err != nil {
		return &model.ErrorPayload{Message: fmt.Sprintf("Failed to start auth: %s", err.Error())}, nil
	}

	state := mapAuthState(pendingAuth.State)
	var passwordHint *string
	if pendingAuth.PasswordHint != "" {
		passwordHint = &pendingAuth.PasswordHint
	}

	payload := model.AdminStartTelegramAccountAuthPayload{
		AuthState: &model.AdminTelegramAccountAuthState{
			Phone:        pendingAuth.Phone,
			State:        state,
			PasswordHint: passwordHint,
		},
	}

	return &payload, nil
}

func mapAuthState(state tgtd.AuthState) model.AdminTelegramAccountAuthStateEnum {
	switch state {
	case tgtd.AuthStateWaitingForCode:
		return model.AdminTelegramAccountAuthStateEnumWaitingForCode
	case tgtd.AuthStateWaitingForPassword:
		return model.AdminTelegramAccountAuthStateEnumWaitingForPassword
	case tgtd.AuthStateAuthorized:
		return model.AdminTelegramAccountAuthStateEnumAuthorized
	case tgtd.AuthStateError:
		return model.AdminTelegramAccountAuthStateEnumError
	}
	return model.AdminTelegramAccountAuthStateEnumError
}
