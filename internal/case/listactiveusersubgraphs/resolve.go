package listactiveusersubgraphs

import (
	"context"
	"fmt"
)

type Env interface {
	ListActiveSubgraphNamesByUserID(ctx context.Context, userID int64) ([]string, error)
	ListActiveTgChatSubgraphNamesByUserID(ctx context.Context, id int64) ([]string, error)
}

func Resolve(ctx context.Context, env Env, userID int64) ([]string, error) {
	uniqMap := make(map[string]struct{})

	payedSubgraphs, err := env.ListActiveSubgraphNamesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active subgraph names: %w", err)
	}

	tgChatSubgraphs, err := env.ListActiveTgChatSubgraphNamesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active tg chat subgraph names: %w", err)
	}

	for _, subgraph := range payedSubgraphs {
		uniqMap[subgraph] = struct{}{}
	}

	for _, subgraph := range tgChatSubgraphs {
		uniqMap[subgraph] = struct{}{}
	}

	result := make([]string, 0, len(uniqMap))

	for subgraph := range uniqMap {
		result = append(result, subgraph)
	}

	return result, nil
}
