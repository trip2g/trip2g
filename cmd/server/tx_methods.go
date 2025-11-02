package main

import (
	"context"
	"trip2g/internal/db"
)

func (a *app) CreateNoteAsset(ctx context.Context, params db.CreateNoteAssetParams) error {
	return a.WithTransaction(ctx, func(env *app) (bool, error) {
		err := env.WriteQueries.CreateNoteAsset(ctx, params)
		if err != nil {
			return false, err
		}
		return true, nil
	})
}
