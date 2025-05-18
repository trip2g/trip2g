package uploadnoteasset

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"
	"trip2g/internal/translit"
)

type Env interface {
	Logger() logger.Logger
	PutAssetObject(ctx context.Context, reader io.Reader, info db.NoteAsset) error
	DeleteAssetObject(ctx context.Context, asset db.NoteAsset) error
	InsertNoteAsset(ctx context.Context, arg db.InsertNoteAssetParams) (db.NoteAsset, error)
	UpsertNoteVersionAsset(ctx context.Context, arg db.UpsertNoteVersionAssetParams) error
	NoteAssetByPathAndHash(ctx context.Context, arg db.NoteAssetByPathAndHashParams) (db.NoteAsset, error)
	NoteVersionAssetPaths(ctx context.Context, id int64) (map[string]struct{}, error)
	PrepareNotes(ctx context.Context) (*appmodel.NoteViews, error)
}

// for sanitize file names
var reUnsafeChars = regexp.MustCompile(`[^a-zA-Z0-9_.-]`)

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

	fileName := translit.ToASCII(filepath.Base(input.Path))
	fileName = reUnsafeChars.ReplaceAllString(fileName, "_")

	asset, err := env.NoteAssetByPathAndHash(ctx, findAssetParams)
	if err != nil {
		if db.IsNoFound(err) {
			noteAssetParams := db.InsertNoteAssetParams{
				AbsolutePath: input.AbsolutePath,
				FileName:     fileName,
				Sha256Hash:   input.Sha256Hash,
				ContentType:  input.File.ContentType,
				Size:         input.File.Size,
			}

			asset, err = env.InsertNoteAsset(ctx, noteAssetParams)
			if err != nil {
				return nil, fmt.Errorf("failed to insert note asset: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to find note asset: %w", err)
		}
	} else {
		alreadyUploaded = true
	}

	noteVersionAssetParams := db.UpsertNoteVersionAssetParams{
		AssetID:   asset.ID,
		VersionID: int64(input.NoteID),
		Path:      input.Path,
	}

	err = env.UpsertNoteVersionAsset(ctx, noteVersionAssetParams)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert note version asset: %w", err)
	}

	if !alreadyUploaded {
		hasher := sha256.New()
		teeReader := io.TeeReader(input.File.File, hasher)

		// TODO: this code must works without transaction!
		// because the uploading process can be long
		err = env.PutAssetObject(ctx, teeReader, asset)
		if err != nil {
			return nil, fmt.Errorf("failed to upload asset: %w", err)
		}

		actualHash := fmt.Sprintf("%x", hasher.Sum(nil))
		if actualHash != input.Sha256Hash {
			// delete the asset from storage
			deleteErr := env.DeleteAssetObject(ctx, asset)
			if deleteErr != nil {
				env.Logger().Error("failed to delete asset object", "asset", asset, "error", deleteErr)
			}

			// will rollback the transaction
			return nil, fmt.Errorf("hash mismatch: expected %s, got %s", input.Sha256Hash, actualHash)
		}
	}

	// isn't optimal. TODO: fix it
	_, err = env.PrepareNotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare notes: %w", err)
	}

	response := model.UploadNoteAssetPayload{
		UploadSkipped: alreadyUploaded,
	}

	return &response, nil
}
