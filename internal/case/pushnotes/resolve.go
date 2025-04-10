package pushnotes

import (
	"context"
	"fmt"
)

//go:generate easyjson -snake_case -no_std_marshalers ./resolve.go

// Env describes all IO deps.
type Env interface {
	InsertNote(ctx context.Context, update Update) error
	PrepareNotes(ctx context.Context) error
}

type Asset struct {
	Path string

	// S3 presigned URL for PUT
	PutPresignedURL string
}

type Update struct {
	Path    string
	Content string
}

//easyjson:json
type Request struct {
	Updates []Update
}

//easyjson:json
type Response struct {
	Assets []Asset
}

func Resolve(ctx context.Context, env Env, request Request) (*Response, error) {
	for _, update := range request.Updates {
		insertErr := env.InsertNote(ctx, update)
		if insertErr != nil {
			return nil, fmt.Errorf("failed to insert note: %w", insertErr)
		}
	}

	err := env.PrepareNotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare notes: %w", err)
	}

	// TODO: PrepareNotes should return the list of assets
	response := Response{}

	return &response, nil
}
