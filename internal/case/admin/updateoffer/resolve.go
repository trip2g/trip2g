package updateoffer

import (
	"context"
	"fmt"
	"time"
	"trip2g/internal/appresp"
	"trip2g/internal/db"
	"trip2g/internal/validator"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

type Env interface {
	UpdateOffer(ctx context.Context, arg db.UpdateOfferParams) (db.Offer, error)
}

type Request struct {
	ID       string
	Names    string
	Lifetime *string
	PriceUSD *float64
	PriceRUB *float64
	PriceBTC *float64
	StartsAt *time.Time
	EndsAt   *time.Time
}

type Response struct {
	appresp.Response

	Row *db.Offer
}

func Resolve(ctx context.Context, env Env, req Request) (*Response, error) {
	response := Response{}
	response.Success = true
	response.Errors = make([]string, 0)

	names, err := validator.NormalizeSubgraphNames(req.Names)
	response.AddErrorIf(err != nil, "invalid_names")

	response.AddErrorIf(req.PriceUSD == nil && req.PriceRUB == nil && req.PriceBTC == nil, "invalid_price")
	response.AddErrorIf(req.PriceUSD != nil && *req.PriceUSD < 0, "invalid_price_usd")
	response.AddErrorIf(req.PriceRUB != nil && *req.PriceRUB < 0, "invalid_price_rub")
	response.AddErrorIf(req.PriceBTC != nil && *req.PriceBTC < 0, "invalid_price_btc")

	if !response.Success {
		return &response, nil
	}

	params := db.UpdateOfferParams{
		ID:       req.ID,
		Names:    names,
		Lifetime: db.ToNullableString(req.Lifetime),
		PriceUsd: db.ToNullableFloat64(req.PriceUSD),
		PriceRub: db.ToNullableFloat64(req.PriceRUB),
		PriceBtc: db.ToNullableFloat64(req.PriceBTC),
		StartsAt: db.ToNullableTime(req.StartsAt),
		EndsAt:   db.ToNullableTime(req.EndsAt),
	}

	offer, err := env.UpdateOffer(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update offer: %w", err)
	}

	response.Row = &offer

	return &response, nil
}
