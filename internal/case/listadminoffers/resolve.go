package listadminoffers

import (
	"context"
	"database/sql"
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
	Lifetime  string
	PriceUSD  *float64
	PriceRUB  *float64
	PriceBTC  *float64
	StartsAt  *time.Time
	EndsAt    *time.Time
}

type Response struct {
	Rows []Offer `json:"rows"`
}

func nullFloat64(v sql.NullFloat64) *float64 {
	if v.Valid {
		return &v.Float64
	}

	return nil
}

func nullTime(v sql.NullTime) *time.Time {
	if v.Valid {
		return &v.Time
	}

	return nil
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
			Lifetime:  offer.Lifetime,

			PriceUSD: nullFloat64(offer.PriceUsd),
			PriceRUB: nullFloat64(offer.PriceRub),
			PriceBTC: nullFloat64(offer.PriceBtc),

			StartsAt: nullTime(offer.StartsAt),
			EndsAt:   nullTime(offer.EndsAt),
		}

		response.Rows = append(response.Rows, offerData)
	}

	return &response, nil
}
