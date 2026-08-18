package revokeusertoken

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	"trip2g/internal/usertoken"
)

// Env deliberately has no "by this user" filter: an admin revoking a token
// works from the token id alone, and requiring the owner as well would mean
// the page that lists every token cannot revoke from that list.
type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	AdminRevokeUserToken(ctx context.Context, id string) (db.UserToken, error)
	AuditLogger() logger.Logger
}

type Input = model.AdminRevokeUserTokenInput
type Payload = model.AdminRevokeUserTokenOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	actor, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	// A token already revoked matches no row, and so does an id that never
	// existed. Both are the same answer to the caller: there is nothing left to
	// revoke, and saying so is not a failure.
	token, err := env.AdminRevokeUserToken(ctx, input.ID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return model.NewFieldError("id", "token not found or already revoked"), nil
	case err != nil:
		return nil, fmt.Errorf("failed to revoke user token: %w", err)
	}

	env.AuditLogger().Info("admin revoke user token",
		"actorUserID", actor.ID,
		"targetUserID", token.UserID,
		"tokenID", token.ID,
	)

	return &model.AdminRevokeUserTokenPayload{Token: &token}, nil
}
