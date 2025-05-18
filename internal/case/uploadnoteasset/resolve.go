package uploadnoteasset

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"path"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

type Env interface {
	PutAssetObject(ctx context.Context, reader io.Reader, info appmodel.FileInfo) error
	InsertNoteAsset(ctx context.Context, arg db.InsertNoteAssetParams) (int64, error)
	UpsertNoteVersionAsset(ctx context.Context, arg db.UpsertNoteVersionAssetParams) error
	NoteAssetByPathAndHash(ctx context.Context, arg db.NoteAssetByPathAndHashParams) (db.NoteAsset, error)
	NoteVersionAssetPaths(ctx context.Context, id int64) (map[string]struct{}, error)
}

func Resolve(ctx context.Context, env Env, input model.UploadNoteAssetInput) (model.UploadNoteAssetOrErrorPayload, error) {
	assetPaths, err := env.NoteVersionAssetPaths(ctx, int64(input.NoteID))
	if err != nil {
		return nil, fmt.Errorf("failed to get note version asset paths: %w", err)
	}

	_, exists := assetPaths[input.Path]
	if !exists {
		return &model.ErrorPayload{Message: "unknown asset path"}, nil
	}

	findAssetParams := db.NoteAssetByPathAndHashParams{
		AbsolutePath: input.AbsolutePath,
		Sha256Hash:   input.Sha256Hash,
	}

	alreadyUploaded := false
	assetID := int64(0)

	asset, err := env.NoteAssetByPathAndHash(ctx, findAssetParams)
	if err != nil {
		if db.IsNoFound(err) {
			noteAssetParams := db.InsertNoteAssetParams{
				AbsolutePath: input.AbsolutePath,
				Sha256Hash:   input.Sha256Hash,
				ContentType:  input.File.ContentType,
				Size:         input.File.Size,
			}

			assetID, err = env.InsertNoteAsset(ctx, noteAssetParams)
			if err != nil {
				return nil, fmt.Errorf("failed to upsert note asset: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to find note asset: %w", err)
		}
	} else {
		alreadyUploaded = true
		assetID = asset.ID
	}

	noteVersionAssetParams := db.UpsertNoteVersionAssetParams{
		AssetID:   assetID,
		VersionID: int64(input.NoteID),
		Path:      input.Path,
	}

	err = env.UpsertNoteVersionAsset(ctx, noteVersionAssetParams)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert note version asset: %w", err)
	}

	if !alreadyUploaded {
		ext := path.Ext(input.Path)
		name := fmt.Sprintf("na/%d%s", assetID, ext)

		hasher := sha256.New()
		teeReader := io.TeeReader(input.File.File, hasher)

		info := appmodel.FileInfo{
			Path: name,
			Size: input.File.Size,

			ContentType: input.File.ContentType,
		}

		err = env.PutAssetObject(ctx, teeReader, info)
		if err != nil {
			return nil, fmt.Errorf("failed to upload asset: %w", err)
		}

		actualHash := fmt.Sprintf("%x", hasher.Sum(nil))
		if actualHash != input.Sha256Hash {
			// will rollback the transaction
			return nil, fmt.Errorf("hash mismatch: expected %s, got %s", input.Sha256Hash, actualHash)
		}
	}

	response := model.UploadNoteAssetPayload{
		UploadSkipped: alreadyUploaded,
	}

	return &response, nil
}
