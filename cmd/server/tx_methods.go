package main

import (
	"context"
	"fmt"
	"trip2g/internal/db"
)

func (a *app) CreateNoteAsset(ctx context.Context, params db.CreateNoteAssetParams) error {
	asset, err := a.InsertNoteAsset(ctx, params.Asset)
	if err != nil {
		return fmt.Errorf("failed to InsertNoteAsset: %w", err)
	}

	noteVersionAssetParams := db.UpsertNoteVersionAssetParams{
		AssetID:   asset.ID,
		VersionID: params.VersionID,
		Path:      params.Path,
	}

	err = a.UpsertNoteVersionAsset(ctx, noteVersionAssetParams)
	if err != nil {
		return fmt.Errorf("failed to UpsertNoteVersionAsset: %w", err)
	}

	return nil
}
