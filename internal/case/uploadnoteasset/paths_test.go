package uploadnoteasset_test

import (
	"bytes"
	"context"
	"database/sql"
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

func newPathsEnv(stored *string) *EnvMock {
	return &EnvMock{
		LoggerFunc: func() logger.Logger { return &logger.TestLogger{} },
		NoteVersionAssetPathsFunc: func(_ context.Context, _ int64) (map[string]struct{}, error) {
			return map[string]struct{}{"images/test.png": {}}, nil
		},
		NoteAssetByPathAndHashFunc: func(_ context.Context, _ db.NoteAssetByPathAndHashParams) (db.NoteAsset, error) {
			return db.NoteAsset{}, sql.ErrNoRows
		},
		CreateNoteAssetFunc: func(_ context.Context, params db.CreateNoteAssetParams) (db.NoteAsset, error) {
			*stored = params.Asset.AbsolutePath
			return db.NoteAsset{ID: 1, AbsolutePath: params.Asset.AbsolutePath}, nil
		},
		PutAssetObjectFunc: func(_ context.Context, reader io.Reader, _ db.NoteAsset) error {
			_, err := io.ReadAll(reader)
			return err
		},
		DeleteAssetObjectFunc: func(_ context.Context, _ db.NoteAsset) error { return nil },
		DeleteNoteAssetFunc:   func(_ context.Context, _ int64) error { return nil },
		PrepareLatestNotesFunc: func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
			return &appmodel.NoteViews{}, nil
		},
		CheckStorageLimitsFunc: func(_ context.Context, _ int64) (string, error) { return "", nil },
	}
}

func pathsInput(absolutePath string) model.UploadNoteAssetInput {
	content := []byte("test file content")

	return model.UploadNoteAssetInput{
		NoteID:       123,
		Path:         "images/test.png",
		AbsolutePath: absolutePath,
		Sha256Hash:   calcHash(content),
		File: graphql.Upload{
			File:     bytes.NewReader(content),
			Filename: "test.png",
			Size:     int64(len(content)),
		},
	}
}

func TestResolve_AbsolutePathStaysInsideVault(t *testing.T) {
	rejected := []string{"../y.bin", "/../y.bin", "a/../../y.bin", "/a/../../y.bin", "..", ""}

	for _, absolutePath := range rejected {
		t.Run("rejects "+absolutePath, func(t *testing.T) {
			var stored string

			env := newPathsEnv(&stored)

			payload, err := uploadnoteasset.Resolve(context.Background(), env, pathsInput(absolutePath))
			require.NoError(t, err)
			require.IsType(t, &model.ErrorPayload{}, payload)
			require.Empty(t, env.CreateNoteAssetCalls())
		})
	}
}

func TestResolve_AbsolutePathIsNormalized(t *testing.T) {
	cases := map[string]string{
		"/a/./b/../c.png": "/a/c.png",
		"a/./c.png":       "a/c.png",
	}

	for raw, want := range cases {
		t.Run(raw, func(t *testing.T) {
			var stored string

			env := newPathsEnv(&stored)

			payload, err := uploadnoteasset.Resolve(context.Background(), env, pathsInput(raw))
			require.NoError(t, err)
			require.IsType(t, &model.UploadNoteAssetPayload{}, payload)
			require.Equal(t, want, stored)
		})
	}
}
