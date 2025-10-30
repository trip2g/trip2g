package uploadnoteasset

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
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
	PrepareLatestNotes(ctx context.Context) (*appmodel.NoteViews, error)
	AcquireTxEnvInRequest(ctx context.Context, label string) error
	ReleaseTxEnvInRequest(ctx context.Context, commit bool) error
}

// for sanitize file names.
var reUnsafeChars = regexp.MustCompile(`[^a-zA-Z0-9_.-]`)

type Input = model.UploadNoteAssetInput
type Payload = model.UploadNoteAssetOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	// Step 1: Validation (no transaction needed)
	assetPaths, err := env.NoteVersionAssetPaths(ctx, input.NoteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get note version asset paths: %w", err)
	}

	_, exists := assetPaths[input.Path]
	if !exists {
		names := []string{}

		for name := range assetPaths {
			names = append(names, name)
		}

		assets := strings.Join(names, ", ")

		return &model.ErrorPayload{Message: "unknown asset path. Assets: " + assets}, nil
	}

	findAssetParams := db.NoteAssetByPathAndHashParams{
		AbsolutePath: input.AbsolutePath,
		Sha256Hash:   input.Sha256Hash,
	}

	fileName := translit.ToASCII(filepath.Base(input.Path))
	fileName = reUnsafeChars.ReplaceAllString(fileName, "_")

	// Step 2: Check if asset already exists (no transaction needed)
	asset, err := env.NoteAssetByPathAndHash(ctx, findAssetParams)
	if err != nil && !db.IsNoFound(err) {
		return nil, fmt.Errorf("failed to find note asset: %w", err)
	}

	alreadyUploaded := !db.IsNoFound(err)
	txCommitted := false

	if !alreadyUploaded {
		// Step 3: Upload file and validate hash (no transaction - can take long time)
		hasher := sha256.New()
		teeReader := io.TeeReader(input.File.File, hasher)

		// Create temporary asset info for upload path (ID will be 0)
		tempAsset := db.NoteAsset{
			AbsolutePath: input.AbsolutePath,
			FileName:     fileName,
			Sha256Hash:   input.Sha256Hash,
			Size:         input.File.Size,
		}

		err = env.PutAssetObject(ctx, teeReader, tempAsset)
		if err != nil {
			return nil, fmt.Errorf("failed to upload asset: %w", err)
		}

		// Validate hash
		actualHash := hex.EncodeToString(hasher.Sum(nil))
		if actualHash != input.Sha256Hash {
			// Delete from storage - no DB cleanup needed since we haven't inserted yet
			deleteErr := env.DeleteAssetObject(ctx, tempAsset)
			if deleteErr != nil {
				env.Logger().Error("failed to delete asset object after hash mismatch", "asset", tempAsset, "error", deleteErr)
			}
			return nil, fmt.Errorf("hash mismatch: expected %s, got %s", input.Sha256Hash, actualHash)
		}

		// Step 4: Start transaction for DB operations
		err = env.AcquireTxEnvInRequest(ctx, "upload_asset")
		if err != nil {
			// Cleanup uploaded file since we can't start transaction
			deleteErr := env.DeleteAssetObject(ctx, tempAsset)
			if deleteErr != nil {
				env.Logger().Error("failed to delete asset object after tx acquire failure", "asset", tempAsset, "error", deleteErr)
			}
			return nil, fmt.Errorf("failed to acquire transaction: %w", err)
		}

		// Insert asset record
		noteAssetParams := db.InsertNoteAssetParams{
			AbsolutePath: input.AbsolutePath,
			FileName:     fileName,
			Sha256Hash:   input.Sha256Hash,
			Size:         input.File.Size,
		}

		asset, err = env.InsertNoteAsset(ctx, noteAssetParams)
		if err != nil {
			// Cleanup uploaded file since DB insert failed
			deleteErr := env.DeleteAssetObject(ctx, tempAsset)
			if deleteErr != nil {
				env.Logger().Error("failed to delete asset object after DB insert failure", "asset", tempAsset, "error", deleteErr)
			}
			return nil, fmt.Errorf("failed to insert note asset: %w", err)
		}
	} else {
		// Asset already exists - start transaction for linking only
		err = env.AcquireTxEnvInRequest(ctx, "link_asset")
		if err != nil {
			return nil, fmt.Errorf("failed to acquire transaction: %w", err)
		}
	}

	// Ensure transaction is released if not committed
	defer func() {
		if !txCommitted {
			releaseErr := env.ReleaseTxEnvInRequest(ctx, false)
			if releaseErr != nil {
				env.Logger().Error("failed to rollback transaction", "error", releaseErr)
			}
		}
	}()

	// Link asset to note version (inside transaction)
	noteVersionAssetParams := db.UpsertNoteVersionAssetParams{
		AssetID:   asset.ID,
		VersionID: input.NoteID,
		Path:      input.Path,
	}

	err = env.UpsertNoteVersionAsset(ctx, noteVersionAssetParams)
	if err != nil {
		if !alreadyUploaded {
			// Cleanup uploaded file
			tempAsset := db.NoteAsset{
				AbsolutePath: input.AbsolutePath,
				FileName:     fileName,
				Sha256Hash:   input.Sha256Hash,
			}
			deleteErr := env.DeleteAssetObject(ctx, tempAsset)
			if deleteErr != nil {
				env.Logger().Error("failed to delete asset object after version link failure", "asset", tempAsset, "error", deleteErr)
			}
		}
		return nil, fmt.Errorf("failed to upsert note version asset: %w", err)
	}

	// Prepare latest notes (inside transaction)
	_, err = env.PrepareLatestNotes(ctx)
	if err != nil {
		if !alreadyUploaded {
			// Cleanup uploaded file
			tempAsset := db.NoteAsset{
				AbsolutePath: input.AbsolutePath,
				FileName:     fileName,
				Sha256Hash:   input.Sha256Hash,
			}
			deleteErr := env.DeleteAssetObject(ctx, tempAsset)
			if deleteErr != nil {
				env.Logger().Error("failed to delete asset object after prepare notes failure", "asset", tempAsset, "error", deleteErr)
			}
		}
		return nil, fmt.Errorf("failed to prepare notes: %w", err)
	}

	// Commit transaction
	err = env.ReleaseTxEnvInRequest(ctx, true)
	if err != nil {
		if !alreadyUploaded {
			// Cleanup uploaded file
			tempAsset := db.NoteAsset{
				AbsolutePath: input.AbsolutePath,
				FileName:     fileName,
				Sha256Hash:   input.Sha256Hash,
			}
			deleteErr := env.DeleteAssetObject(ctx, tempAsset)
			if deleteErr != nil {
				env.Logger().Error("failed to delete asset object after tx commit failure", "asset", tempAsset, "error", deleteErr)
			}
		}
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	txCommitted = true

	response := model.UploadNoteAssetPayload{
		UploadSkipped: alreadyUploaded,
	}

	return &response, nil
}
