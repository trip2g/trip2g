package disablegittoken

import (
	"context"
	"database/sql"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	DisableGitToken(ctx context.Context, params db.DisableGitTokenParams) (db.GitToken, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Input = model.DisableGitTokenInput
type Payload = model.DisableGitTokenOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.ID, validation.Required),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	params := db.DisableGitTokenParams{
		ID:         input.ID,
		DisabledBy: sql.NullInt64{Valid: true, Int64: int64(token.ID)},
	}

	gitToken, err := env.DisableGitToken(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to disable git token: %w", err)
	}

	payload := model.DisableGitTokenPayload{
		GitToken: &gitToken,
	}

	return &payload, nil
}