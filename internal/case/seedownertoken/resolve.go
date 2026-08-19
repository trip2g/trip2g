// Package seedownertoken reconciles the one personal token the instance owner
// hands to a fleet host, from OWNER_PERSONAL_TOKEN_VALUE. It runs on every boot
// and is idempotent: an unchanged value writes nothing.
package seedownertoken

import (
	"context"
	"errors"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/personaltoken"
)

type Env interface {
	UserByEmail(ctx context.Context, email string) (db.User, error)
	UserTokenByHashAny(ctx context.Context, hash string) (db.UserToken, error)
	InsertUserToken(ctx context.Context, arg db.InsertUserTokenParams) (db.UserToken, error)
	RevokeSupersededOwnerTokens(ctx context.Context, arg db.RevokeSupersededOwnerTokensParams) (int64, error)
	GenerateUniqID() string
}

type Outcome string

const (
	OutcomeSkipped     Outcome = "skipped"
	OutcomeDisabled    Outcome = "disabled"
	OutcomeSeeded      Outcome = "seeded"
	OutcomeFound       Outcome = "found"
	OutcomeLeftRevoked Outcome = "left revoked"
)

// Result is what the caller logs: one line per boot.
type Result struct {
	Outcome           Outcome
	RevokedSuperseded int64
}

// Resolve reconciles the reserved owner-token row against value. Insertion goes
// straight to the write query rather than through createusertoken, so an owner
// who already holds a full token list can never fail a boot.
func Resolve(ctx context.Context, env Env, ownerEmail, value string) (Result, error) {
	if ownerEmail == "" {
		if value != "" {
			return Result{}, errors.New("OWNER_PERSONAL_TOKEN_VALUE is set but there is no owner (OWNER_EMAIL) to own the token")
		}
		return Result{Outcome: OutcomeSkipped}, nil
	}

	owner, err := env.UserByEmail(ctx, ownerEmail)
	if err != nil {
		return Result{}, fmt.Errorf("failed to get owner by email: %w", err)
	}

	if value == "" {
		revoked, revErr := env.RevokeSupersededOwnerTokens(ctx, revokeParams(owner.ID, ""))
		if revErr != nil {
			return Result{}, fmt.Errorf("failed to revoke seeded owner tokens: %w", revErr)
		}
		return Result{Outcome: OutcomeDisabled, RevokedSuperseded: revoked}, nil
	}

	hash := personaltoken.Hash(value)

	outcome, err := reconcileRow(ctx, env, owner.ID, value, hash)
	if err != nil {
		return Result{}, err
	}

	revoked, err := env.RevokeSupersededOwnerTokens(ctx, revokeParams(owner.ID, hash))
	if err != nil {
		return Result{}, fmt.Errorf("failed to revoke superseded owner tokens: %w", err)
	}

	return Result{Outcome: outcome, RevokedSuperseded: revoked}, nil
}

// reconcileRow inserts the row for value unless one already carries its hash. A
// revoked row is left alone: revoking in the admin UI is the only way to kill
// the credential without restarting the instance, and a restart that undid it
// would make that kill switch worthless.
func reconcileRow(ctx context.Context, env Env, ownerID int64, value, hash string) (Outcome, error) {
	existing, err := env.UserTokenByHashAny(ctx, hash)
	switch {
	case err == nil && existing.RevokedAt != nil:
		return OutcomeLeftRevoked, nil
	case err == nil:
		return OutcomeFound, nil
	case !db.IsNoFound(err):
		return "", fmt.Errorf("failed to look up the seeded owner token: %w", err)
	}

	_, err = env.InsertUserToken(ctx, db.InsertUserTokenParams{
		ID:          env.GenerateUniqID(),
		UserID:      ownerID,
		Name:        personaltoken.ReservedOwnerTokenName,
		TokenHash:   hash,
		TokenPrefix: personaltoken.DisplayPrefix(value),
		Scope:       "all",
	})
	if err != nil {
		return "", fmt.Errorf("failed to insert the seeded owner token: %w", err)
	}

	return OutcomeSeeded, nil
}

func revokeParams(ownerID int64, keepHash string) db.RevokeSupersededOwnerTokensParams {
	return db.RevokeSupersededOwnerTokensParams{
		UserID:   ownerID,
		Name:     personaltoken.ReservedOwnerTokenName,
		KeepHash: keepHash,
	}
}
