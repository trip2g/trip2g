package revokeusersubgraphaccess

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"trip2g/internal/appresp"
	"trip2g/internal/db"
	"trip2g/internal/usertoken"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

type Env interface {
	CreateRevoke(ctx context.Context, arg db.CreateRevokeParams) (int64, error)
	RevokeUserSubgraphAccess(ctx context.Context, arg db.RevokeUserSubgraphAccessParams) error
}

var (
	ErrEmptyReason = errors.New("reason is required")
	ErrNoAuth      = errors.New("user token is required")
)

type Request struct {
	SubgraphAccessID int64
	Reason           string
	UserToken        *usertoken.Data
}

func (r *Request) Validate() error {
	if r.Reason == "" {
		return ErrEmptyReason
	}
	if r.UserToken == nil {
		return ErrNoAuth
	}
	return nil
}

type Response struct {
	appresp.Response
}

func Resolve(ctx context.Context, env Env, request Request) (Response, error) {
	var response Response
	response.Success = true

	createRevokeParams := db.CreateRevokeParams{
		TargetType: "user_subgraph_access",
		TargetID:   request.SubgraphAccessID,
		By:         request.UserToken.ID,
		Reason:     sql.NullString{String: request.Reason, Valid: true},
	}

	// Create revoke record
	revokeID, err := env.CreateRevoke(ctx, createRevokeParams)
	if err != nil {
		return response, fmt.Errorf("failed to create revoke: %w", err)
	}

	revokeUSAParams := db.RevokeUserSubgraphAccessParams{
		RevokeID: sql.NullInt64{Int64: revokeID, Valid: true},
		ID:       request.SubgraphAccessID,
	}

	// Revoke the access
	err = env.RevokeUserSubgraphAccess(ctx, revokeUSAParams)
	if err != nil {
		return response, fmt.Errorf("failed to revoke access: %w", err)
	}

	return response, nil
}
