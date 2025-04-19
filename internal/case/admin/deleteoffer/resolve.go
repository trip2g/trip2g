package deleteoffer

import (
	"context"
	"fmt"
	"trip2g/internal/appresp"
	"trip2g/internal/db"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

type Env interface {
	DeleteOffer(ctx context.Context, id string) (db.Offer, error)
}

type Request struct {
	ID string
}

type Response struct {
	appresp.Response

	Row *db.Offer
}

func Resolve(ctx context.Context, env Env, req Request) (*Response, error) {
	response := Response{}
	response.Success = true
	response.Errors = make([]string, 0)

	offer, err := env.DeleteOffer(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete offer: %w", err)
	}

	response.Row = &offer

	return &response, nil
}
