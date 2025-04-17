package pushnotes

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

// Env describes all IO deps.
type Env interface {
	Logger() logger.Logger
	InsertNote(ctx context.Context, update db.Note) error
	InsertSubgraph(ctx context.Context, name string) error
	PrepareNotes(ctx context.Context) (map[string]*mdloader.Page, error)
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

var ErrNoUpdates = fmt.Errorf("no updates provided")
var ErrInvalidUpdate = fmt.Errorf("invalid update provided")

func (r *Request) Validate() error {
	if len(r.Updates) == 0 {
		return ErrNoUpdates
	}

	for _, update := range r.Updates {
		if update.Path == "" || update.Content == "" {
			return ErrInvalidUpdate
		}
	}

	return nil
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

	pages, err := env.PrepareNotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare notes: %w", err)
	}

	subgraphs := make(map[string]struct{})

	for _, page := range pages {
		sbI, ok := page.RawMeta["subgraphs"]
		if !ok {
			continue
		}

		switch sbI := sbI.(type) {
		case string:
			subgraphs[sbI] = struct{}{}
		case []interface{}:
			for _, sb := range sbI {
				if sbStr, ok := sb.(string); ok {
					subgraphs[sbStr] = struct{}{}
				}
			}
		default:
			return nil, fmt.Errorf("invalid subgraph type: %T", sbI)
		}
	}

	env.Logger().Info("insert subgraphs", "subgraphs", subgraphs)

	for subgraph := range subgraphs {
		insertErr := env.InsertSubgraph(ctx, subgraph)
		if insertErr != nil {
			return nil, fmt.Errorf("failed to insert subgraph: %w", insertErr)
		}
	}

	// TODO: PrepareNotes should return the list of assets
	response := Response{}

	return &response, nil
}
