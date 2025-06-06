package createoffer

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
	InsertOffer(ctx context.Context, arg db.InsertOfferParams) (db.Offer, error)
	InsertOfferSubgraph(ctx context.Context, arg db.InsertOfferSubgraphParams) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	SubgraphByID(ctx context.Context, id int64) (db.Subgraph, error)
	GenerateUniqID() string
}

func normalizeInput(i *model.CreateOfferInput) {
	if i.Lifetime != nil {
		*i.Lifetime = strings.TrimSpace(*i.Lifetime)
	}
}

func validateInput(i *model.CreateOfferInput) *model.ErrorPayload {
	err := ozzo.ValidateStruct(i,
		ozzo.Field(&i.PriceUsd, ozzo.Min(0.0)),
		ozzo.Field(&i.SubgraphIds, ozzo.Required),
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

type Input = model.CreateOfferInput
type Payload = model.CreateOfferOrErrorPayload

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

	publicID := env.GenerateUniqID()

	for _, subgraphID := range input.SubgraphIds {
		_, subgraphErr := env.SubgraphByID(ctx, subgraphID)
		if subgraphErr != nil {
			if db.IsNoFound(subgraphErr) {
				return &model.ErrorPayload{Message: fmt.Sprintf("subgraph with ID %d does not exist", subgraphID)}, nil
			}
			return nil, fmt.Errorf("failed to get subgraph %d: %w", subgraphID, subgraphErr)
		}
	}

	offerParams := db.InsertOfferParams{
		PublicID: publicID,
		PriceUsd: sql.NullFloat64{Float64: input.PriceUsd, Valid: true},
		StartsAt: db.ToNullableTime(input.StartsAt),
		EndsAt:   db.ToNullableTime(input.EndsAt),
	}

	if input.Lifetime != nil {
		lifetime := appmodel.Lifetime(*input.Lifetime)
		offerParams.Lifetime = &lifetime
	}

	offer, err := env.InsertOffer(ctx, offerParams)
	if err != nil {
		return nil, fmt.Errorf("failed to insert offer: %w", err)
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

	payload := model.CreateOfferPayload{
		Offer: &offer,
	}

	return &payload, nil
}
