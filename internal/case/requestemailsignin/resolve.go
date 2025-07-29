package requestemailsignin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type Env interface {
	QueueRequestSignInEmail(ctx context.Context, email string, code string) error
	UserByEmail(ctx context.Context, email string) (db.User, error)
	CountActiveSignInCodes(ctx context.Context, userID int64) (int64, error)
	CreateSignInCode(ctx context.Context, userID int64) (string, error)
	UserBanByUserID(ctx context.Context, userID int64) (*db.UserBan, error)

	// patreon, boosty, etc
	TryToAutoRegisterUser(ctx context.Context, email string) (*db.User, error)
}

func NormalizeInput(input *model.RequestEmailSignInCodeInput) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
}

func ValidateInput(req *model.RequestEmailSignInCodeInput) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(req,
		ozzo.Field(&req.Email, ozzo.Required, is.Email),
	))
}

type Input = model.RequestEmailSignInCodeInput

func Resolve(ctx context.Context, env Env, input Input) (model.RequestEmailSignInCodeOrErrorPayload, error) {
	NormalizeInput(&input)

	errPayload := ValidateInput(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	user, err := env.UserByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			autoUser, autoErr := env.TryToAutoRegisterUser(ctx, input.Email)
			if autoErr != nil {
				return nil, fmt.Errorf("failed to auto-register user: %w", autoErr)
			}

			if autoUser == nil {
				return model.NewFieldError("email", "not_found"), nil
			}

			user = *autoUser
		} else {
			return nil, fmt.Errorf("failed to get user by email: %w", err)
		}
	}

	ban, err := env.UserBanByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check user ban: %w", err)
	}

	if ban != nil {
		msg := "user_banned"
		if ban.Reason != "" {
			msg = "user_banned: " + ban.Reason
		}

		return &model.ErrorPayload{Message: msg}, nil
	}

	count, err := env.CountActiveSignInCodes(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to count active signin codes: %w", err)
	}

	if count > 3 {
		return &model.ErrorPayload{Message: "too_many_sign_in_codes"}, nil
	}

	code, err := env.CreateSignInCode(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create signin code: %w", err)
	}

	err = env.QueueRequestSignInEmail(ctx, input.Email, code)
	if err != nil {
		return nil, fmt.Errorf("failed to queue signin email: %w", err)
	}

	response := model.RequestEmailSignInCodePayload{Success: true}

	return &response, nil
}
