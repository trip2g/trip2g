package addfederationsecretsubgraph

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg addfederationsecretsubgraph_test . Env

import (
	"context"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	InsertFederationSecretSubgraph(ctx context.Context, arg db.InsertFederationSecretSubgraphParams) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Input = model.AddFederationSecretSubgraphInput
type Payload = model.AddFederationSecretSubgraphOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.Kid, ozzo.Required),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	params := db.InsertFederationSecretSubgraphParams{
		Kid:        input.Kid,
		SubgraphID: input.SubgraphID,
		CreatedBy:  int64(token.ID),
	}

	err = env.InsertFederationSecretSubgraph(ctx, params)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return &model.ErrorPayload{Message: "This subgraph is already assigned to this federation secret"}, nil
		}
		return nil, fmt.Errorf("failed to add federation secret subgraph: %w", err)
	}

	return &model.AddFederationSecretSubgraphPayload{Success: true}, nil
}
