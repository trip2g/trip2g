package revokeusersubgraphaccess

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg revokeusersubgraphaccess_test . Env

import (
	"context"
	"fmt"
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

const TargetType = "user_subgraph_access"

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	UserSubgraphAccessByID(ctx context.Context, id int64) (db.UserSubgraphAccess, error)
	CreateRevoke(ctx context.Context, arg db.CreateRevokeParams) (db.Revoke, error)
	RevokeUserSubgraphAccess(ctx context.Context, arg db.RevokeUserSubgraphAccessParams) (db.UserSubgraphAccess, error)
}

type Input = model.RevokeUserSubgraphAccessInput
type Payload = model.RevokeUserSubgraphAccessOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return &model.ErrorPayload{Message: "Reason is required"}, nil
	}

	actor, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	access, err := env.UserSubgraphAccessByID(ctx, input.ID)
	if err != nil {
		if db.IsNoFound(err) {
			return &model.ErrorPayload{Message: "Access not found"}, nil
		}

		return nil, fmt.Errorf("failed to get user subgraph access: %w", err)
	}

	if access.RevokeID != nil {
		return &model.ErrorPayload{Message: "Access already revoked"}, nil
	}

	revokeParams := db.CreateRevokeParams{
		TargetType: TargetType,
		TargetID:   access.ID,
		ByID:       int64(actor.ID),
		Reason:     &reason,
	}

	revoke, err := env.CreateRevoke(ctx, revokeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create revoke: %w", err)
	}

	accessParams := db.RevokeUserSubgraphAccessParams{
		RevokeID: &revoke.ID,
		ID:       access.ID,
	}

	revoked, err := env.RevokeUserSubgraphAccess(ctx, accessParams)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke user subgraph access: %w", err)
	}

	return &model.RevokeUserSubgraphAccessPayload{Access: &revoked}, nil
}
