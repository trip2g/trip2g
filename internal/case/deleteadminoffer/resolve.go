package deleteadminoffer

import (
	"context"
	"fmt"
	"trip2g/internal/appresp"
	"trip2g/internal/db"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

type Env interface {
	DeleteOffer(ctx context.Context, id int64) (db.Offer, error)
}

type Request struct {
	ID int64
}

type Response struct {
	appresp.Response

	Row *db.Offer
}

func Resolve(ctx context.Context, env Env, req Request) (*Response, error) {
	response := Response{}
	response.Success = true
	response.Errors = make([]string, 0)

	// Validate ID
	if req.ID <= 0 {
		response.Success = false
		response.Errors = append(response.Errors, "invalid_id")
		return &response, nil
	}

	// Delete the offer (set ends_at to now)
	offer, err := env.DeleteOffer(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete offer: %w", err)
	}

	response.Row = &offer

	return &response, nil
}
