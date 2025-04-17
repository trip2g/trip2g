package listadminoffers

import (
	"context"
	"fmt"
	"time"
	"trip2g/internal/db"
)

//go:generate easyjson -all -snake_case -no_std_marshalers ./resolve.go

type Env interface {
	AllOffers(ctx context.Context) ([]db.Offer, error)
}

type Request struct {
}

type Offer struct {
	ID        string
	CreatedAt time.Time
	Names     string
	Lifetime  *string
	PriceUSD  *float64
	PriceRUB  *float64
	PriceBTC  *float64
	StartsAt  *time.Time
	EndsAt    *time.Time
}

type Response struct {
	Rows []Offer
}

func Resolve(ctx context.Context, env Env, _ Request) (*Response, error) {
	response := Response{
		Rows: make([]Offer, 0),
	}

	offers, err := env.AllOffers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all offers: %w", err)
	}

	for _, offer := range offers {
		offerData := Offer{
			ID:        offer.ID,
			CreatedAt: offer.CreatedAt,
			Names:     offer.Names,
			Lifetime:  db.ToStringPtr(offer.Lifetime),

			PriceUSD: db.ToFloat64Ptr(offer.PriceUsd),
			PriceRUB: db.ToFloat64Ptr(offer.PriceRub),
			PriceBTC: db.ToFloat64Ptr(offer.PriceBtc),

			StartsAt: db.ToTimePtr(offer.StartsAt),
			EndsAt:   db.ToTimePtr(offer.EndsAt),
		}

		response.Rows = append(response.Rows, offerData)
	}

	return &response, nil
}
