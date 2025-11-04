package main

import (
	"context"
	"trip2g/internal/db"
)

func (a *app) CreateNoteAsset(ctx context.Context, params db.CreateNoteAssetParams) error {
	return a.WithTransaction(ctx, func(txCtx context.Context, env *app) (bool, error) {
		err := env.WriteQueries.CreateNoteAsset(txCtx, params)
		return err == nil, err
	})
}
