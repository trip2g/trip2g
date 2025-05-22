package graph

import (
	"context"
	"database/sql"
	"errors"
	"trip2g/internal/appreq"
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

var errUnauthorized = errors.New("unauthorized")

func checkAdmin(ctx context.Context) error {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return err
	}

	token, err := req.UserToken()
	if err != nil {
		return err
	}

	if !token.IsAdmin() {
		return errUnauthorized
	}

	return nil
}
