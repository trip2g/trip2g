package graph

import (
	"context"
	"database/sql"
	"errors"
)

func resolveOne[T any](
	ctx context.Context,
	id int64,
	fetch func(context.Context, int64) (T, error),
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
