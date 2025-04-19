package createusersubgraphaccess

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"trip2g/internal/appresp"
	"trip2g/internal/db"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

type Env interface {
	CreateUserSubgraphAccess(ctx context.Context, arg db.CreateUserSubgraphAccessParams) (db.UserSubgraphAccess, error)
}

type Request struct {
	UserID     int64
	SubgraphID int64
	PurchaseID *int64
	ExpiresAt  *string
}

func (r *Request) Validate() error {
	return nil
}

type Response struct {
	appresp.Response
	Access *db.UserSubgraphAccess
}

func Resolve(ctx context.Context, env Env, request Request) (Response, error) {
	var response Response
	response.Success = true

	arg := db.CreateUserSubgraphAccessParams{
		UserID:     request.UserID,
		SubgraphID: request.SubgraphID,
		PurchaseID: db.ToNullableInt64(request.PurchaseID),
	}

	if request.ExpiresAt != nil {
		expiresAt, err := time.Parse(time.RFC3339, *request.ExpiresAt)
		if err != nil {
			return response, fmt.Errorf("invalid expires_at: %w", err)
		}
		arg.ExpiresAt = sql.NullTime{
			Time:  expiresAt,
			Valid: true,
		}
	}

	access, err := env.CreateUserSubgraphAccess(ctx, arg)
	if err != nil {
		return response, fmt.Errorf("failed to create user subgraph access: %w", err)
	}

	response.Access = &access
	return response, nil
}
