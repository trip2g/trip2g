package graph

import (
	"context"
	"database/sql"
	"errors"
	"trip2g/internal/appreq"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
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

func prepareTOC(note *appmodel.NoteView) []model.NoteTocItem {
	toc := make([]model.NoteTocItem, 0, len(note.TOC()))
	for _, heading := range note.TOC() {
		level := heading.Level
		if level > 2147483647 {
			level = 2147483647 // Cap at max int32 value
		}
		toc = append(toc, model.NoteTocItem{
			ID:    heading.ID,
			Title: heading.Text,
			Level: int32(level), //nolint:gosec // heading level is always a small positive number
		})
	}

	return toc
}
