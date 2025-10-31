package db

import (
	"context"
	"fmt"
)

type CreateNoteAssetParams struct {
	Asset InsertNoteAssetParams

	// UpsertNoteVersionAssetParams without AssetID
	VersionID int64
	Path      string
}

func (a *WriteQueries) CreateNoteAsset(ctx context.Context, params CreateNoteAssetParams) error {
	asset, err := a.InsertNoteAsset(ctx, params.Asset)
	if err != nil {
		return fmt.Errorf("failed to InsertNoteAsset: %w", err)
	}

	noteVersionAssetParams := UpsertNoteVersionAssetParams{
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
