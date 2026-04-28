package removefederationsecretsubgraph

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg removefederationsecretsubgraph_test . Env

import (
	"context"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	DeleteFederationSecretSubgraph(ctx context.Context, arg db.DeleteFederationSecretSubgraphParams) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type Input = model.RemoveFederationSecretSubgraphInput
type Payload = model.RemoveFederationSecretSubgraphOrErrorPayload

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

	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	params := db.DeleteFederationSecretSubgraphParams{
		Kid:        input.Kid,
		SubgraphID: input.SubgraphID,
	}

	err = env.DeleteFederationSecretSubgraph(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to remove federation secret subgraph: %w", err)
	}

	return &model.RemoveFederationSecretSubgraphPayload{Success: true}, nil
}
