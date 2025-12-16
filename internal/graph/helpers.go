package graph

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/model"
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

func resolveOnePtr[T any, K any](
	ctx context.Context,
	id *K,
	fetch func(context.Context, K) (T, error),
) (*T, error) {
	if id == nil {
		return nil, nil
	}

	return resolveOne(ctx, *id, fetch)
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

func (r *queryResolver) convertSearchResultsToNotePath(ctx context.Context, nodes []model.SearchResult) ([]db.NotePath, error) {
	res := []db.NotePath{}

	for _, result := range nodes {
		if result.NoteView != nil {
			pathID := result.NoteView.PathID

			notePath, selectErr := r.env(ctx).NotePathByID(ctx, pathID)
			if selectErr != nil {
				return nil, fmt.Errorf("failed to get note path by ID %d: %w", pathID, selectErr)
			}

			res = append(res, notePath)
		}
	}

	return res, nil
}
