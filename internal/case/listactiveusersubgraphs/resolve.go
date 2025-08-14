package listactiveusersubgraphs

import (
	"context"
	"fmt"
	"trip2g/internal/db"
)

type Env interface {
	ListActiveSubgraphNamesByUserID(ctx context.Context, userID int64) ([]string, error)
	ListActiveTgChatSubgraphNamesByUserID(ctx context.Context, id int64) ([]string, error)
	ListActivePatreonSubgraphNamesByUserID(ctx context.Context, id int64) ([]string, error)
	ListActiveBoostySubgraphNamesByUserID(ctx context.Context, id int64) ([]string, error)

	AdminByUserID(ctx context.Context, userID int64) (db.Admin, error)
	ListAllSubgraphs(ctx context.Context) ([]db.Subgraph, error)
}

// TODO: maybe we need to add a cache for this function
// or store results of this function in the user table.

func Resolve(ctx context.Context, env Env, userID int64) ([]string, error) {
	// Check if the user is an admin
	_, err := env.AdminByUserID(ctx, userID)
	if err != nil {
		if !db.IsNoFound(err) {
			return nil, fmt.Errorf("failed to get admin by user ID: %w", err)
		}
	} else {
		allSubgraphs, allErr := env.ListAllSubgraphs(ctx)
		if allErr != nil {
			return nil, fmt.Errorf("failed to list all subgraphs: %w", allErr)
		}

		result := make([]string, 0, len(allSubgraphs))
		for _, subgraph := range allSubgraphs {
			result = append(result, subgraph.Name)
		}

		return result, nil
	}

	uniqMap := make(map[string]struct{})

	payedSubgraphs, err := env.ListActiveSubgraphNamesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active subgraph names: %w", err)
	}

	tgChatSubgraphs, err := env.ListActiveTgChatSubgraphNamesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active tg chat subgraph names: %w", err)
	}

	patreonSubgraphs, err := env.ListActivePatreonSubgraphNamesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active patreon subgraph names: %w", err)
	}

	boostySubgraphs, err := env.ListActiveBoostySubgraphNamesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active boosty subgraph names: %w", err)
	}

	for _, subgraph := range payedSubgraphs {
		uniqMap[subgraph] = struct{}{}
	}

	for _, subgraph := range tgChatSubgraphs {
		uniqMap[subgraph] = struct{}{}
	}

	for _, subgraph := range patreonSubgraphs {
		uniqMap[subgraph] = struct{}{}
	}

	for _, subgraph := range boostySubgraphs {
		uniqMap[subgraph] = struct{}{}
	}

	result := make([]string, 0, len(uniqMap))

	for subgraph := range uniqMap {
		result = append(result, subgraph)
	}

	return result, nil
}
