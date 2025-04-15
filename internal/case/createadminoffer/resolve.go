package createadminoffer

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"trip2g/internal/db"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

type Env interface {
	CreateOffer(ctx context.Context, arg db.CreateOfferParams) error
}

type Request struct {
	ID       string
	Names    string
	Lifetime string
	PriceUSD *float64
	PriceRUB *float64
	PriceBTC *float64
	StartsAt *time.Time
	EndsAt   *time.Time
}

type Response struct {
	Success bool
	Errors  []string
}

func Resolve(ctx context.Context, env Env, req Request) (*Response, error) {
	response := Response{
		Success: true,
		Errors:  make([]string, 0),
	}

	// Validate required fields
	if req.ID == "" {
		response.Success = false
		response.Errors = append(response.Errors, "id_required")
	}
	if req.Names == "" {
		response.Success = false
		response.Errors = append(response.Errors, "names_required")
	}
	if req.Lifetime == "" {
		response.Success = false
		response.Errors = append(response.Errors, "lifetime_required")
	}

	// If validation failed, return early
	if !response.Success {
		return &response, nil
	}

	// Convert nullable fields to sql.NullFloat64 and sql.NullTime
	var priceUsd, priceRub, priceBtc sql.NullFloat64
	var startsAt, endsAt sql.NullTime

	if req.PriceUSD != nil {
		priceUsd = sql.NullFloat64{
			Float64: *req.PriceUSD,
			Valid:   true,
		}
	}
	if req.PriceRUB != nil {
		priceRub = sql.NullFloat64{
			Float64: *req.PriceRUB,
			Valid:   true,
		}
	}
	if req.PriceBTC != nil {
		priceBtc = sql.NullFloat64{
			Float64: *req.PriceBTC,
			Valid:   true,
		}
	}
	if req.StartsAt != nil {
		startsAt = sql.NullTime{
			Time:  *req.StartsAt,
			Valid: true,
		}
	}
	if req.EndsAt != nil {
		endsAt = sql.NullTime{
			Time:  *req.EndsAt,
			Valid: true,
		}
	}

	// Create the offer
	err := env.CreateOffer(ctx, db.CreateOfferParams{
		ID:       req.ID,
		Names:    req.Names,
		Lifetime: req.Lifetime,
		PriceUsd: priceUsd,
		PriceRub: priceRub,
		PriceBtc: priceBtc,
		StartsAt: startsAt,
		EndsAt:   endsAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create offer: %w", err)
	}

	return &response, nil
}
