package canceltelegramaccountauth

import (
	"context"
	"fmt"
	"strings"

	"trip2g/internal/graph/model"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

type Env interface {
	TelegramAccountCancelAuth(phone string) error
}

type Input = model.AdminCancelTelegramAccountAuthInput
type Payload = model.AdminCancelTelegramAccountAuthOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	err := ozzo.ValidateStruct(&input,
		ozzo.Field(&input.Phone, ozzo.Required),
	)
	if err != nil {
		return model.NewOzzoError(err), nil
	}

	phone := strings.TrimSpace(input.Phone)

	err = env.TelegramAccountCancelAuth(phone)
	if err != nil {
		return &model.ErrorPayload{Message: fmt.Sprintf("Failed to cancel auth: %s", err.Error())}, nil
	}

	payload := model.AdminCancelTelegramAccountAuthPayload{
		Success: true,
	}

	return &payload, nil
}
