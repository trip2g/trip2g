package createusertoken

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	usercase "trip2g/internal/case/createusertoken"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/usertoken"
)

// maxExpiryDays caps what an admin may mint for someone else. A token handed to
// a colleague or an agent outlives the conversation that produced it, so the
// ceiling is the thing that eventually revokes it when nobody remembers to.
const maxExpiryDays = 365

// Env embeds the minting case's Env: this case decides who gets a token and
// records that decision, and the token itself is still minted in one place.
type Env interface {
	usercase.Env

	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	UserByID(ctx context.Context, id int64) (db.User, error)
	AuditLogger() logger.Logger
}

type Input = model.AdminCreateUserTokenInput
type Payload = model.CreateUserTokenOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	errPayload := model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.Name, ozzo.Required, ozzo.Length(1, 100)),
	))
	if errPayload != nil {
		return errPayload
	}

	if r.ExpiresInDays != nil && (*r.ExpiresInDays < 1 || *r.ExpiresInDays > maxExpiryDays) {
		return model.NewFieldError("expiresInDays", fmt.Sprintf("must be between 1 and %d", maxExpiryDays))
	}

	return nil
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	actor, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	// The token is useless without a user to carry the subgraph grants, and a
	// typo in the id would otherwise mint a credential nobody can revoke by
	// looking at the user page. A missing row is the admin's mistake; anything
	// else is the database failing and must not read as "no such user".
	_, err = env.UserByID(ctx, input.UserID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return model.NewFieldError("userId", "user not found"), nil
	case err != nil:
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	payload, err := usercase.Resolve(ctx, env, input.UserID, model.CreateUserTokenInput{
		Name:          input.Name,
		ExpiresInDays: input.ExpiresInDays,
	})
	if err != nil {
		return nil, err
	}

	// Audit who minted it and for whom, never the token itself — it is a
	// usable credential and the log outlives the person reading it.
	env.AuditLogger().Info("admin create user token",
		"actorUserID", actor.ID,
		"targetUserID", input.UserID,
		"name", input.Name,
	)

	return payload, nil
}
