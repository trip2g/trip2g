package downloadonboardingvault

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"testing"

	"trip2g/internal/db"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

type mockEnv struct {
	publicURL string
}

func (m *mockEnv) GenerateAPIKey() string {
	return "test-api-key-12345"
}

func (m *mockEnv) InsertAPIKey(_ context.Context, _ db.InsertAPIKeyParams) (db.ApiKey, error) {
	return db.ApiKey{}, nil
}

func (m *mockEnv) LatestNoteViews() *model.NoteViews {
	return nil
}

func (m *mockEnv) PublicURL() string {
	return m.publicURL
}

func TestResolve_IndexMDContainsPublicURL(t *testing.T) {
	env := &mockEnv{publicURL: "https://example.com"}

	zipData, err := Resolve(context.Background(), env, 1)
	require.NoError(t, err)
	require.NotEmpty(t, zipData)

	// Read the zip and find _index.md.
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)

	var indexContent string
	for _, file := range reader.File {
		if file.Name == indexMDPath {
			rc, openErr := file.Open()
			require.NoError(t, openErr)

			content, readErr := io.ReadAll(rc)
			rc.Close()
			require.NoError(t, readErr)

			indexContent = string(content)
			break
		}
	}

	require.NotEmpty(t, indexContent, "_index.md should exist in zip")
	require.Contains(t, indexContent, "https://example.com", "_index.md should contain publicURL")
	require.NotContains(t, indexContent, "{{publicUrl}}", "_index.md should not contain placeholder")
}
