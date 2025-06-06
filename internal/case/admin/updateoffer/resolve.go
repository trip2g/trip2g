package updateoffer

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

type Env interface {
	UpdateOffer(ctx context.Context, arg db.UpdateOfferParams) (db.Offer, error)
	DeleteOfferSubgraphs(ctx context.Context, offerID int64) error
	InsertOfferSubgraph(ctx context.Context, arg db.InsertOfferSubgraphParams) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	OfferByID(ctx context.Context, id int64) (db.Offer, error)
	SubgraphByID(ctx context.Context, id int64) (db.Subgraph, error)
}

func normalizeInput(i *model.UpdateOfferInput) {
	if i.Lifetime != nil {
		*i.Lifetime = strings.TrimSpace(*i.Lifetime)
	}
}

func validateInput(i *model.UpdateOfferInput) *model.ErrorPayload {
	err := ozzo.ValidateStruct(i,
		ozzo.Field(&i.ID, ozzo.Required),
		ozzo.Field(&i.PriceUsd, ozzo.When(i.PriceUsd != nil, ozzo.Min(0.0))),
		ozzo.Field(&i.SubgraphIds, ozzo.When(i.SubgraphIds != nil, ozzo.Required)),
	)
	if err != nil {
		return model.NewOzzoError(err)
	}

	// Custom validation: startsAt must be before endsAt
	if i.StartsAt != nil && i.EndsAt != nil {
		if !i.StartsAt.Before(*i.EndsAt) {
			return &model.ErrorPayload{
				ByFields: []model.FieldMessage{
					{Name: "startsAt", Value: "must be before ends at"},
				},
			}
		}
	}

	return nil
}

type Input = model.UpdateOfferInput
type Payload = model.UpdateOfferOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	_, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	normalizeInput(&input)

	errorPayload := validateInput(&input)
	if errorPayload != nil {
		return errorPayload, nil
	}

	_, err = env.OfferByID(ctx, input.ID)
	if err != nil {
		return &model.ErrorPayload{Message: "offer not found"}, nil
	}

	if input.SubgraphIds != nil {
		for _, subgraphID := range input.SubgraphIds {
			_, subgraphErr := env.SubgraphByID(ctx, subgraphID)
			if subgraphErr != nil {
				return &model.ErrorPayload{Message: fmt.Sprintf("subgraph with ID %d does not exist", subgraphID)}, nil
			}
		}
	}

	updateParams := db.UpdateOfferParams{
		ID: input.ID,
	}

	if input.Lifetime != nil {
		lifetime := appmodel.Lifetime(*input.Lifetime)
		updateParams.Lifetime = &lifetime
	}

	if input.PriceUsd != nil {
		updateParams.PriceUsd = sql.NullFloat64{Float64: *input.PriceUsd, Valid: true}
	}

	if input.StartsAt != nil {
		updateParams.StartsAt = sql.NullTime{Time: *input.StartsAt, Valid: true}
	}

	if input.EndsAt != nil {
		updateParams.EndsAt = sql.NullTime{Time: *input.EndsAt, Valid: true}
	}

	offer, err := env.UpdateOffer(ctx, updateParams)
	if err != nil {
		return nil, fmt.Errorf("failed to update offer: %w", err)
	}

	if input.SubgraphIds != nil {
		err = env.DeleteOfferSubgraphs(ctx, offer.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to delete offer subgraphs: %w", err)
		}

		for _, subgraphID := range input.SubgraphIds {
			osParams := db.InsertOfferSubgraphParams{
				OfferID:    offer.ID,
				SubgraphID: subgraphID,
			}

			insertErr := env.InsertOfferSubgraph(ctx, osParams)
			if insertErr != nil {
				return nil, fmt.Errorf("failed to insert offer subgraph: %w", insertErr)
			}
		}
	}

	payload := model.UpdateOfferPayload{
		Offer: &offer,
	}

	return &payload, nil
}
