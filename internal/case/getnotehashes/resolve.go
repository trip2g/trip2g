package getnotehashes

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

type Response struct {
	Map map[string]string
}

func Resolve(ctx context.Context, env Env, _ Request) (*Response, error) {
	response := Response{
		Map: make(map[string]string),
	}

	paths, err := env.AllNotePaths(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all note paths: %w", err)
	}

	for _, path := range paths {
		response.Map[path.Value] = path.LatestContentHash
	}

	return &response, nil
}
