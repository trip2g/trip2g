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
	"trip2g/internal/appreq"
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
	DeleteNoteAsset(ctx context.Context, id int64) error
	CreateNoteAsset(ctx context.Context, params db.CreateNoteAssetParams) (db.NoteAsset, error)
	UpsertNoteVersionAsset(ctx context.Context, arg db.UpsertNoteVersionAssetParams) error
	NoteAssetByPathAndHash(ctx context.Context, arg db.NoteAssetByPathAndHashParams) (db.NoteAsset, error)
	NoteAssetExists(ctx context.Context, asset db.NoteAsset) (bool, error)
	NoteVersionAssetPaths(ctx context.Context, id int64) (map[string]struct{}, error)
	NoteVersionByID(ctx context.Context, id int64) (db.NoteVersionByIDRow, error)
	PrepareLatestNotes(ctx context.Context, partial bool) (*appmodel.NoteViews, error)
	CheckStorageLimits(ctx context.Context, additionalAssetBytes int64) (string, error)
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
					PrepareLatestNotesFunc: func(ctx context.Context, partial bool) (*appmodel.NoteViews, error) {
						return &appmodel.NoteViews{}, nil
					},
					CheckStorageLimitsFunc: func(_ context.Context, _ int64) (string, error) { return "", nil },
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
					CheckStorageLimitsFunc: func(_ context.Context, _ int64) (string, error) { return "", nil },
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
					CheckStorageLimitsFunc: func(_ context.Context, _ int64) (string, error) { return "", nil },
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
					NoteAssetExistsFunc: func(ctx context.Context, asset db.NoteAsset) (bool, error) {
						// Object is still present in storage - no heal needed
						return true, nil
					},
					UpsertNoteVersionAssetFunc: func(ctx context.Context, arg db.UpsertNoteVersionAssetParams) error {
						return nil
					},
					PrepareLatestNotesFunc: func(ctx context.Context, partial bool) (*appmodel.NoteViews, error) {
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
				// PutAssetObject should NOT be called (object still present, no heal needed)
				require.Empty(t, env.PutAssetObjectCalls())
				// UpsertNoteVersionAsset SHOULD be called to link existing asset
				require.Len(t, env.UpsertNoteVersionAssetCalls(), 1)
				// PrepareLatestNotes should be called to update views
				require.Len(t, env.PrepareLatestNotesCalls(), 1)
			},
		},
		{
			name: "success - asset row exists but object missing from storage (self-heal)",
			input: model.UploadNoteAssetInput{
				NoteID:       456,
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
						// DB row exists, but the underlying object was lost (e.g. bucket wipe)
						return db.NoteAsset{
							ID:           99,
							AbsolutePath: arg.AbsolutePath,
							FileName:     "test.png",
							Sha256Hash:   arg.Sha256Hash,
							Size:         int64(len(testContent)),
						}, nil
					},
					NoteAssetExistsFunc: func(ctx context.Context, asset db.NoteAsset) (bool, error) {
						return false, nil
					},
					PutAssetObjectFunc: func(ctx context.Context, reader io.Reader, info db.NoteAsset) error {
						_, err := io.ReadAll(reader)
						return err
					},
					UpsertNoteVersionAssetFunc: func(ctx context.Context, arg db.UpsertNoteVersionAssetParams) error {
						return nil
					},
					PrepareLatestNotesFunc: func(ctx context.Context, partial bool) (*appmodel.NoteViews, error) {
						return &appmodel.NoteViews{}, nil
					},
				}
			},
			wantErr: false,
			validate: func(t *testing.T, payload model.UploadNoteAssetOrErrorPayload, env *EnvMock) {
				require.IsType(t, &model.UploadNoteAssetPayload{}, payload)
				p := payload.(*model.UploadNoteAssetPayload)
				require.True(t, p.UploadSkipped)

				// CreateNoteAsset should NOT be called (DB row already exists)
				require.Empty(t, env.CreateNoteAssetCalls())
				// PutAssetObject SHOULD be called to re-upload the missing object
				require.Len(t, env.PutAssetObjectCalls(), 1)
				// UpsertNoteVersionAsset SHOULD be called to link existing asset
				require.Len(t, env.UpsertNoteVersionAssetCalls(), 1)
				// PrepareLatestNotes should be called to update views
				require.Len(t, env.PrepareLatestNotesCalls(), 1)
			},
		},
		{
			name: "failure - storage limit exceeded",
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
					CheckStorageLimitsFunc: func(_ context.Context, _ int64) (string, error) {
						return "assets storage limit exceeded", nil
					},
				}
			},
			wantErr: false,
			validate: func(t *testing.T, payload model.UploadNoteAssetOrErrorPayload, env *EnvMock) {
				require.IsType(t, &model.ErrorPayload{}, payload)
				p := payload.(*model.ErrorPayload)
				require.Equal(t, "assets storage limit exceeded", p.Message)
				require.Empty(t, env.CreateNoteAssetCalls())
				require.Empty(t, env.PutAssetObjectCalls())
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

// TestResolve_ScopedToken pins write-scope enforcement for uploadNoteAsset:
// a scoped shortapitoken may attach assets only to notes whose path matches
// its write_patterns (same matcher as updateNotes); unscoped requests skip
// the note-path lookup entirely.
func TestResolve_ScopedToken(t *testing.T) {
	testContent := []byte("asset bytes")
	testHash := calcHash(testContent)

	makeInput := func() model.UploadNoteAssetInput {
		return model.UploadNoteAssetInput{
			NoteID:       42,
			Path:         "images/pic.png",
			AbsolutePath: "/abs/images/pic.png",
			Sha256Hash:   testHash,
			File: graphql.Upload{
				File:     bytes.NewReader(testContent),
				Filename: "pic.png",
				Size:     int64(len(testContent)),
			},
		}
	}

	// makeEnv returns mocks for the asset-reuse happy path (existing asset,
	// object present, link upserted). notePath is what NoteVersionByID reports
	// for version 42; leave NoteVersionByIDFunc nil to assert it is not called.
	makeEnv := func(notePath string) *EnvMock {
		env := &EnvMock{
			LoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
			NoteVersionAssetPathsFunc: func(_ context.Context, _ int64) (map[string]struct{}, error) {
				return map[string]struct{}{"images/pic.png": {}}, nil
			},
			NoteAssetByPathAndHashFunc: func(_ context.Context, _ db.NoteAssetByPathAndHashParams) (db.NoteAsset, error) {
				return db.NoteAsset{ID: 1}, nil
			},
			NoteAssetExistsFunc: func(_ context.Context, _ db.NoteAsset) (bool, error) { return true, nil },
			UpsertNoteVersionAssetFunc: func(_ context.Context, _ db.UpsertNoteVersionAssetParams) error {
				return nil
			},
			PrepareLatestNotesFunc: func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
				return &appmodel.NoteViews{}, nil
			},
		}
		if notePath != "" {
			env.NoteVersionByIDFunc = func(_ context.Context, id int64) (db.NoteVersionByIDRow, error) {
				require.Equal(t, int64(42), id)
				return db.NoteVersionByIDRow{VersionID: id, Path: notePath}, nil
			}
		}
		return env
	}

	scopedCtx := func(patterns []string) context.Context {
		return appreq.NewContext(context.Background(), &appreq.Request{
			WebhookScoped:        true,
			WebhookDeliveryKind:  "change",
			WebhookWritePatterns: patterns,
		})
	}

	tests := []struct {
		name       string
		ctx        context.Context
		notePath   string // "" = NoteVersionByID must not be called
		wantDenied bool
	}{
		{
			name:     "scoped token with matching pattern allowed",
			ctx:      scopedCtx([]string{"notes/**"}),
			notePath: "notes/todo.md",
		},
		{
			name:       "scoped token with non-matching pattern denied",
			ctx:        scopedCtx([]string{"notes/**"}),
			notePath:   "secrets/private.md",
			wantDenied: true,
		},
		{
			name:       "scoped token with empty patterns denied",
			ctx:        scopedCtx([]string{}),
			notePath:   "notes/todo.md",
			wantDenied: true,
		},
		{
			name:     "unscoped request allowed without path lookup",
			ctx:      context.Background(),
			notePath: "", // NoteVersionByIDFunc stays nil — a call would panic
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := makeEnv(tt.notePath)

			result, err := uploadnoteasset.Resolve(tt.ctx, env, makeInput())
			require.NoError(t, err)

			if tt.wantDenied {
				ep, ok := result.(*model.ErrorPayload)
				require.True(t, ok, "expected *ErrorPayload (write denied), got %T", result)
				require.Contains(t, ep.Message, "write denied for path: "+tt.notePath)
				require.Empty(t, env.UpsertNoteVersionAssetCalls(), "denied upload must not link the asset")
				require.Empty(t, env.NoteVersionAssetPathsCalls(), "denied upload must not proceed to validation")
				return
			}
			require.IsType(t, &model.UploadNoteAssetPayload{}, result)
			require.Len(t, env.UpsertNoteVersionAssetCalls(), 1)
		})
	}
}
