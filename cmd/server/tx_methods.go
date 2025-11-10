package main

import (
	"context"
	"trip2g/internal/db"
)

func (a *app) CreateNoteAsset(ctx context.Context, params db.CreateNoteAssetParams) (db.NoteAsset, error) {
	var asset db.NoteAsset
	err := a.WithTransaction(ctx, func(txCtx context.Context, env *app) (bool, error) {
		var txErr error
		asset, txErr = env.WriteQueries.CreateNoteAsset(txCtx, params)
		return txErr == nil, txErr
	})
	return asset, err
}
