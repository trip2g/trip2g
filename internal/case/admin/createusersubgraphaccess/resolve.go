package createusersubgraphaccess

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg createusersubgraphaccess_test . Env

import (
	"context"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	UserByID(ctx context.Context, id int64) (db.User, error)
	SubgraphByID(ctx context.Context, id int64) (db.Subgraph, error)
	AdminCreateUserSubgraphAccess(ctx context.Context, arg db.AdminCreateUserSubgraphAccessParams) (db.UserSubgraphAccess, error)
}

type Input = model.CreateUserSubgraphAccessInput
type Payload = model.CreateUserSubgraphAccessOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.SubgraphIds, ozzo.Required, ozzo.Length(1, 100)),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	// Check if user exists
	_, err = env.UserByID(ctx, input.UserID)
	if err != nil {
		if db.IsNoFound(err) {
			return &model.ErrorPayload{Message: "User not found"}, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Validate all subgraphs exist
	for _, subgraphID := range input.SubgraphIds {
		_, err = env.SubgraphByID(ctx, subgraphID)
		if err != nil {
			if db.IsNoFound(err) {
				return &model.ErrorPayload{Message: fmt.Sprintf("Subgraph %d not found", subgraphID)}, nil
			}
			return nil, fmt.Errorf("failed to get subgraph: %w", err)
		}
	}

	// Create accesses for each subgraph
	accesses := make([]db.UserSubgraphAccess, 0, len(input.SubgraphIds))
	for _, subgraphID := range input.SubgraphIds {
		params := db.AdminCreateUserSubgraphAccessParams{
			UserID:     input.UserID,
			SubgraphID: subgraphID,
			ExpiresAt:  input.ExpiresAt,
			CreatedBy:  ptr.To(int64(token.ID)),
		}

		access, createErr := env.AdminCreateUserSubgraphAccess(ctx, params)
		if createErr != nil {
			return nil, fmt.Errorf("failed to create user subgraph access: %w", createErr)
		}
		accesses = append(accesses, access)
	}

	payload := model.CreateUserSubgraphAccessPayload{
		Accesses: accesses,
	}

	return &payload, nil
}
