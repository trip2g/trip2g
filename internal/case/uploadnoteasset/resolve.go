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
	DeleteNoteAsset(ctx context.Context, id int64) error
	CreateNoteAsset(ctx context.Context, params db.CreateNoteAssetParams) (db.NoteAsset, error)
	NoteAssetByPathAndHash(ctx context.Context, arg db.NoteAssetByPathAndHashParams) (db.NoteAsset, error)
	NoteVersionAssetPaths(ctx context.Context, id int64) (map[string]struct{}, error)
	PrepareLatestNotes(ctx context.Context) (*appmodel.NoteViews, error)
}

// for sanitize file names.
var reUnsafeChars = regexp.MustCompile(`[^a-zA-Z0-9_.-]`)

type Input = model.UploadNoteAssetInput
type Payload = model.UploadNoteAssetOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	// Step 1: Validation
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

	// Step 2: Check if asset already exists
	_, err = env.NoteAssetByPathAndHash(ctx, findAssetParams)
	if err != nil && !db.IsNoFound(err) {
		return nil, fmt.Errorf("failed to find note asset: %w", err)
	}

	alreadyUploaded := !db.IsNoFound(err)

	if !alreadyUploaded {
		err = uploadAndCreateAsset(ctx, env, input, fileName)
		if err != nil {
			return nil, err
		}

		// Prepare latest notes only when new asset was uploaded (transactional)
		_, err = env.PrepareLatestNotes(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare notes: %w", err)
		}
	}

	response := model.UploadNoteAssetPayload{
		UploadSkipped: alreadyUploaded,
	}

	return &response, nil
}

func uploadAndCreateAsset(ctx context.Context, env Env, input Input, fileName string) error {
	// Step 3: Create asset in database first to get ID (transactional)
	// IMPORTANT: Client MUST provide correct hash. We cannot read entire file
	// before upload to verify hash (memory inefficient for multi-GB files).
	// Hash is verified AFTER upload using TeeReader. If mismatch occurs,
	// both file and DB record are deleted (client error).
	createParams := db.CreateNoteAssetParams{
		Asset: db.InsertNoteAssetParams{
			AbsolutePath: input.AbsolutePath,
			FileName:     fileName,
			Sha256Hash:   input.Sha256Hash,
			Size:         input.File.Size,
		},
		VersionID: input.NoteID,
		Path:      input.Path,
	}

	asset, err := env.CreateNoteAsset(ctx, createParams)
	if err != nil {
		return fmt.Errorf("failed to create note asset: %w", err)
	}

	// Step 4: Upload file and calculate hash simultaneously using TeeReader
	hasher := sha256.New()
	teeReader := io.TeeReader(input.File.File, hasher)

	err = env.PutAssetObject(ctx, teeReader, asset)
	if err != nil {
		// Cleanup: delete DB record (best effort, log if fails)
		deleteErr := env.DeleteNoteAsset(ctx, asset.ID)
		if deleteErr != nil {
			env.Logger().Error("failed to delete DB record after upload failure", "assetID", asset.ID, "error", deleteErr)
		}
		return fmt.Errorf("failed to upload asset: %w", err)
	}

	// Step 5: Validate hash after upload
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != input.Sha256Hash {
		// Cleanup: delete file from MinIO (best effort, log if fails)
		deleteFileErr := env.DeleteAssetObject(ctx, asset)
		if deleteFileErr != nil {
			env.Logger().Error("failed to delete file after hash mismatch", "asset", asset, "error", deleteFileErr)
		}
		// Cleanup: delete DB record (best effort, log if fails)
		deleteDBErr := env.DeleteNoteAsset(ctx, asset.ID)
		if deleteDBErr != nil {
			env.Logger().Error("failed to delete DB record after hash mismatch", "assetID", asset.ID, "error", deleteDBErr)
		}
		return fmt.Errorf("hash mismatch: expected %s, got %s", input.Sha256Hash, actualHash)
	}

	return nil
}
