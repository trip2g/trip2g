package createadminoffer

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
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
	Lifetime *string
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

var ErrNameOnlyValid = errors.New("name must be only [a-zA-Z0-9_]")

func normalizeNames(input string) (string, error) {
	// split by |
	parts := strings.Split(input, "|")

	var regex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

	for i, part := range parts {
		part = strings.TrimSpace(part)

		// must be only [a-zA-Z0-9_]
		if !regex.MatchString(part) {
			return "", ErrNameOnlyValid
		}

		parts[i] = part
	}

	sort.StringSlice(parts).Sort()

	return strings.Join(parts, "|"), nil
}

func Resolve(ctx context.Context, env Env, req Request) (*Response, error) {
	response := Response{
		Success: true,
		Errors:  make([]string, 0),
	}

	names, err := normalizeNames(req.Names)
	if err != nil || names == "" {
		response.Success = false
		response.Errors = append(response.Errors, "invalid_names")
	}

	// If validation failed, return early
	if !response.Success {
		return &response, nil
	}

	// Create the offer
	err = env.CreateOffer(ctx, db.CreateOfferParams{
		Names:    req.Names,
		Lifetime: db.ToNullableString(req.Lifetime),
		PriceUsd: db.ToNullableFloat64(req.PriceUSD),
		PriceRub: db.ToNullableFloat64(req.PriceRUB),
		PriceBtc: db.ToNullableFloat64(req.PriceBTC),
		StartsAt: db.ToNullableTime(req.StartsAt),
		EndsAt:   db.ToNullableTime(req.EndsAt),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create offer: %w", err)
	}

	return &response, nil
}
