package setpatreontiersubgraphs

import (
	"context"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

type Env interface {
	DeletePatreonTierSubgraphsByTierID(ctx context.Context, tierID int64) error
	InsertPatreonTierSubgraph(ctx context.Context, arg db.InsertPatreonTierSubgraphParams) error
	PatreonTierByID(ctx context.Context, id int64) (db.PatreonTier, error)
}

// Type aliases for cleaner code.
type Input = model.SetPatreonTierSubgraphsInput
type Payload = model.SetPatreonTierSubgraphsOrErrorPayload

// validateRequest validates input and returns ErrorPayload if invalid.
func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.TierID, validation.Required, validation.Min(int64(1))),
		// SubgraphIds can be empty array (to clear all), so just check it's not nil
		validation.Field(&r.SubgraphIds, validation.NotNil),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	// Always validate input first
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil // User-visible errors go in ErrorPayload
	}

	// First, delete all existing subgraphs for this tier
	err := env.DeletePatreonTierSubgraphsByTierID(ctx, input.TierID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete existing tier subgraphs: %w", err)
	}

	// Then insert all the new subgraphs
	for _, subgraphID := range input.SubgraphIds {
		params := db.InsertPatreonTierSubgraphParams{
			TierID:     input.TierID,
			SubgraphID: subgraphID,
			CreatedBy:  1, // TODO: Get actual admin user ID from context
		}

		insertErr := env.InsertPatreonTierSubgraph(ctx, params)
		if insertErr != nil {
			// If we fail partway through, the transaction should rollback
			// but since we don't have transaction support in this interface,
			// we'll just return the error
			return nil, fmt.Errorf("failed to insert tier subgraph %d: %w", subgraphID, insertErr)
		}
	}

	// Fetch the updated tier
	tier, tierErr := env.PatreonTierByID(ctx, input.TierID)
	if tierErr != nil {
		return nil, fmt.Errorf("failed to fetch updated tier: %w", tierErr)
	}

	// Define payload as separate variable
	payload := model.SetPatreonTierSubgraphsPayload{
		Tier: &tier,
	}

	return &payload, nil
}
