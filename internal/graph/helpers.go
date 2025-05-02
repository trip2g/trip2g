package graph

import (
	"context"
	"database/sql"
	"errors"
)

func resolveOne[T any, K any](
	ctx context.Context,
	id K,
	fetch func(context.Context, K) (T, error),
) (*T, error) {
	row, err := fetch(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &row, nil
}
