package creategittoken

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	GenerateGitToken() string
	InsertGitToken(ctx context.Context, params db.InsertGitTokenParams) (db.GitToken, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Input = model.CreateGitTokenInput
type Payload = model.CreateGitTokenOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.Description, validation.Required),
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

	gitToken := env.GenerateGitToken()
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(gitToken)))

	params := db.InsertGitTokenParams{
		ValueSha256: tokenHash,
		AdminID:     sql.NullInt64{Valid: true, Int64: int64(token.ID)},
		Description: input.Description,
		CanPull:     sql.NullBool{Valid: true, Bool: input.CanPull},
		CanPush:     sql.NullBool{Valid: true, Bool: input.CanPush},
	}

	createdToken, err := env.InsertGitToken(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to insert git token: %w", err)
	}

	payload := model.CreateGitTokenPayload{
		Value:    gitToken,
		GitToken: &createdToken,
	}

	return &payload, nil
}
