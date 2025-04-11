package listadminnotepaths

import (
	"context"
	"fmt"
	"trip2g/internal/db"
)

//go:generate easyjson -all -snake_case -no_std_marshalers ./resolve.go

type Env interface {
	AllNotePaths(ctx context.Context) ([]db.NotePath, error)
}

type Request struct {
}

type NotePath struct {
	ID        int64
	Value     string
	ValueHash string

	VersionCount      int64
	LatestContentHash string
}

type Response struct {
	Rows []NotePath
}

func Resolve(ctx context.Context, env Env, _ Request) (*Response, error) {
	response := Response{
		Rows: make([]NotePath, 0),
	}

	paths, err := env.AllNotePaths(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all note paths: %w", err)
	}

	for _, path := range paths {
		response.Rows = append(response.Rows, NotePath{
			ID:        path.ID,
			Value:     path.Value,
			ValueHash: path.ValueHash,

			VersionCount:      path.VersionCount,
			LatestContentHash: path.LatestContentHash,
		})
	}

	return &response, nil
}
