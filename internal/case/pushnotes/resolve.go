package pushnotes

import (
	"context"
	"fmt"
	"trip2g/internal/db"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

// Env describes all IO deps.
type Env interface {
	InsertNote(ctx context.Context, update db.Note) error
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

type Request struct {
	Updates []Update
}

type Response struct {
	Assets []Asset
}

func Resolve(ctx context.Context, env Env, request Request) (*Response, error) {
	for _, update := range request.Updates {
		note := db.Note{
			Path:    update.Path,
			Content: update.Content,
		}

		insertErr := env.InsertNote(ctx, note)
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
