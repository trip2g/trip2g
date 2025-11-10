package uploadnoteasset_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"testing"
	"trip2g/internal/case/uploadnoteasset"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/require"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg uploadnoteasset_test . Env

type Env interface {
	Logger() logger.Logger
	PutAssetObject(ctx context.Context, reader io.Reader, info db.NoteAsset) error
	DeleteAssetObject(ctx context.Context, asset db.NoteAsset) error
	CreateNoteAsset(ctx context.Context, params db.CreateNoteAssetParams) error
	NoteAssetByPathAndHash(ctx context.Context, arg db.NoteAssetByPathAndHashParams) (db.NoteAsset, error)
	NoteVersionAssetPaths(ctx context.Context, id int64) (map[string]struct{}, error)
	PrepareLatestNotes(ctx context.Context) (*appmodel.NoteViews, error)
}

func calcHash(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

func TestResolve(t *testing.T) {
	ctx := context.Background()
	testContent := []byte("test file content")
	testHash := calcHash(testContent)

	tests := []struct {
		name     string
		input    model.UploadNoteAssetInput
		setupEnv func() *EnvMock
		wantErr  bool
		checkErr func(t *testing.T, err error)
		validate func(t *testing.T, payload model.UploadNoteAssetOrErrorPayload, env *EnvMock)
	}{
		{
			name: "success - new asset upload",
			input: model.UploadNoteAssetInput{
				NoteID:       123,
				Path:         "images/test.png",
				AbsolutePath: "/absolute/path/test.png",
				Sha256Hash:   testHash,
				File: graphql.Upload{
					File:     bytes.NewReader(testContent),
					Filename: "test.png",
					Size:     int64(len(testContent)),
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.TestLogger{}
					},
					NoteVersionAssetPathsFunc: func(ctx context.Context, id int64) (map[string]struct{}, error) {
						return map[string]struct{}{
							"images/test.png": {},
						}, nil
					},
					NoteAssetByPathAndHashFunc: func(ctx context.Context, arg db.NoteAssetByPathAndHashParams) (db.NoteAsset, error) {
						return db.NoteAsset{}, sql.ErrNoRows
					},
					CreateNoteAssetFunc: func(ctx context.Context, params db.CreateNoteAssetParams) (db.NoteAsset, error) {
						return db.NoteAsset{
							ID:           1,
							AbsolutePath: params.Asset.AbsolutePath,
							FileName:     params.Asset.FileName,
							Sha256Hash:   params.Asset.Sha256Hash,
							Size:         params.Asset.Size,
						}, nil
					},
					PutAssetObjectFunc: func(ctx context.Context, reader io.Reader, info db.NoteAsset) error {
						// Must consume the reader to simulate actual upload
						_, err := io.ReadAll(reader)
						return err
					},
					DeleteAssetObjectFunc: func(ctx context.Context, asset db.NoteAsset) error {
						return nil
					},
					DeleteNoteAssetFunc: func(ctx context.Context, id int64) error {
						return nil
					},
					PrepareLatestNotesFunc: func(ctx context.Context) (*appmodel.NoteViews, error) {
						return &appmodel.NoteViews{}, nil
					},
				}
			},
			wantErr: false,
			validate: func(t *testing.T, payload model.UploadNoteAssetOrErrorPayload, env *EnvMock) {
				require.IsType(t, &model.UploadNoteAssetPayload{}, payload)
				p := payload.(*model.UploadNoteAssetPayload)
				require.False(t, p.UploadSkipped)

				// Verify CreateNoteAsset was called
				require.Len(t, env.CreateNoteAssetCalls(), 1)
				// Verify PutAssetObject was called
				require.Len(t, env.PutAssetObjectCalls(), 1)
			},
		},
		{
			name: "failure - hash mismatch does NOT leave DB records",
			input: model.UploadNoteAssetInput{
				NoteID:       123,
				Path:         "images/test.png",
				AbsolutePath: "/absolute/path/test.png",
				Sha256Hash:   "wronghash",
				File: graphql.Upload{
					File:     bytes.NewReader(testContent),
					Filename: "test.png",
					Size:     int64(len(testContent)),
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.TestLogger{}
					},
					NoteVersionAssetPathsFunc: func(ctx context.Context, id int64) (map[string]struct{}, error) {
						return map[string]struct{}{
							"images/test.png": {},
						}, nil
					},
					NoteAssetByPathAndHashFunc: func(ctx context.Context, arg db.NoteAssetByPathAndHashParams) (db.NoteAsset, error) {
						return db.NoteAsset{}, sql.ErrNoRows
					},
					CreateNoteAssetFunc: func(ctx context.Context, params db.CreateNoteAssetParams) (db.NoteAsset, error) {
						return db.NoteAsset{
							ID:           1,
							AbsolutePath: params.Asset.AbsolutePath,
							FileName:     params.Asset.FileName,
							Sha256Hash:   params.Asset.Sha256Hash,
							Size:         params.Asset.Size,
						}, nil
					},
					PutAssetObjectFunc: func(ctx context.Context, reader io.Reader, info db.NoteAsset) error {
						// Must consume the reader to simulate actual upload
						_, err := io.ReadAll(reader)
						return err
					},
					DeleteAssetObjectFunc: func(ctx context.Context, asset db.NoteAsset) error {
						return nil
					},
					DeleteNoteAssetFunc: func(ctx context.Context, id int64) error {
						return nil
					},
				}
			},
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "hash mismatch")
			},
			validate: func(t *testing.T, payload model.UploadNoteAssetOrErrorPayload, env *EnvMock) {
				// CreateNoteAsset was called
				require.Len(t, env.CreateNoteAssetCalls(), 1)
				// PutAssetObject was called
				require.Len(t, env.PutAssetObjectCalls(), 1)
				// DeleteAssetObject was called to cleanup file
				require.Len(t, env.DeleteAssetObjectCalls(), 1)
				// DeleteNoteAsset was called to cleanup DB record
				require.Len(t, env.DeleteNoteAssetCalls(), 1)
			},
		},
		{
			name: "failure - upload fails does NOT leave DB records",
			input: model.UploadNoteAssetInput{
				NoteID:       123,
				Path:         "images/test.png",
				AbsolutePath: "/absolute/path/test.png",
				Sha256Hash:   testHash,
				File: graphql.Upload{
					File:     bytes.NewReader(testContent),
					Filename: "test.png",
					Size:     int64(len(testContent)),
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.TestLogger{}
					},
					NoteVersionAssetPathsFunc: func(ctx context.Context, id int64) (map[string]struct{}, error) {
						return map[string]struct{}{
							"images/test.png": {},
						}, nil
					},
					NoteAssetByPathAndHashFunc: func(ctx context.Context, arg db.NoteAssetByPathAndHashParams) (db.NoteAsset, error) {
						return db.NoteAsset{}, sql.ErrNoRows
					},
					CreateNoteAssetFunc: func(ctx context.Context, params db.CreateNoteAssetParams) (db.NoteAsset, error) {
						return db.NoteAsset{
							ID:           1,
							AbsolutePath: params.Asset.AbsolutePath,
							FileName:     params.Asset.FileName,
							Sha256Hash:   params.Asset.Sha256Hash,
							Size:         params.Asset.Size,
						}, nil
					},
					PutAssetObjectFunc: func(ctx context.Context, reader io.Reader, info db.NoteAsset) error {
						return errors.New("upload failed")
					},
					DeleteAssetObjectFunc: func(ctx context.Context, asset db.NoteAsset) error {
						return nil
					},
					DeleteNoteAssetFunc: func(ctx context.Context, id int64) error {
						return nil
					},
				}
			},
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "failed to upload asset")
			},
			validate: func(t *testing.T, payload model.UploadNoteAssetOrErrorPayload, env *EnvMock) {
				// CreateNoteAsset was called
				require.Len(t, env.CreateNoteAssetCalls(), 1)
				// DeleteNoteAsset was called for DB cleanup (file not uploaded yet)
				require.Len(t, env.DeleteNoteAssetCalls(), 1)
			},
		},
		{
			name: "success - asset reuse (same asset in different versions)",
			input: model.UploadNoteAssetInput{
				NoteID:       456, // Different note version
				Path:         "images/test.png",
				AbsolutePath: "/absolute/path/test.png",
				Sha256Hash:   testHash,
				File: graphql.Upload{
					File:     bytes.NewReader(testContent),
					Filename: "test.png",
					Size:     int64(len(testContent)),
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.TestLogger{}
					},
					NoteVersionAssetPathsFunc: func(ctx context.Context, id int64) (map[string]struct{}, error) {
						return map[string]struct{}{
							"images/test.png": {},
						}, nil
					},
					NoteAssetByPathAndHashFunc: func(ctx context.Context, arg db.NoteAssetByPathAndHashParams) (db.NoteAsset, error) {
						// Asset already exists from previous version
						return db.NoteAsset{
							ID:           99,
							AbsolutePath: arg.AbsolutePath,
							FileName:     "test.png",
							Sha256Hash:   arg.Sha256Hash,
							Size:         int64(len(testContent)),
						}, nil
					},
					UpsertNoteVersionAssetFunc: func(ctx context.Context, arg db.UpsertNoteVersionAssetParams) error {
						return nil
					},
					PrepareLatestNotesFunc: func(ctx context.Context) (*appmodel.NoteViews, error) {
						return &appmodel.NoteViews{}, nil
					},
				}
			},
			wantErr: false,
			validate: func(t *testing.T, payload model.UploadNoteAssetOrErrorPayload, env *EnvMock) {
				require.IsType(t, &model.UploadNoteAssetPayload{}, payload)
				p := payload.(*model.UploadNoteAssetPayload)
				require.True(t, p.UploadSkipped)

				// CreateNoteAsset should NOT be called (asset already exists)
				require.Empty(t, env.CreateNoteAssetCalls())
				// PutAssetObject should NOT be called (no upload needed)
				require.Empty(t, env.PutAssetObjectCalls())
				// UpsertNoteVersionAsset SHOULD be called to link existing asset
				require.Len(t, env.UpsertNoteVersionAssetCalls(), 1)
				// PrepareLatestNotes should be called to update views
				require.Len(t, env.PrepareLatestNotesCalls(), 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.setupEnv()

			payload, err := uploadnoteasset.Resolve(ctx, env, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
			}

			if tt.validate != nil {
				tt.validate(t, payload, env)
			}
		})
	}
}
