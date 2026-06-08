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

	setAdminToolsCalls []db.SetApiKeyMcpAdminToolsParams
}

func (m *mockEnv) GenerateAPIKey() string {
	return "test-api-key-12345"
}

func (m *mockEnv) InsertAPIKey(_ context.Context, _ db.InsertAPIKeyParams) (db.ApiKey, error) {
	return db.ApiKey{ID: 42}, nil
}

//nolint:staticcheck // method name must match the sqlc-generated interface
func (m *mockEnv) SetApiKeyMcpAdminTools(_ context.Context, arg db.SetApiKeyMcpAdminToolsParams) error {
	m.setAdminToolsCalls = append(m.setAdminToolsCalls, arg)
	return nil
}

func (m *mockEnv) LatestNoteViews() *model.NoteViews {
	return nil
}

func (m *mockEnv) PublicURL() string {
	return m.publicURL
}

func TestResolve_IndexMDContainsPublicURL(t *testing.T) {
	env := &mockEnv{publicURL: "https://example.com"}

	zipData, err := Resolve(context.Background(), env, 1, false)
	require.NoError(t, err)
	require.NotEmpty(t, zipData)

	// Read the zip and find _index.md.
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)

	var indexContent string
	for _, file := range reader.File {
		if file.Name == "example.com/_index.md" {
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

func TestResolve_EnableAdminGraphQL(t *testing.T) {
	env := &mockEnv{publicURL: "https://example.com"}

	_, err := Resolve(context.Background(), env, 7, true)
	require.NoError(t, err)

	require.Len(t, env.setAdminToolsCalls, 1)
	require.Equal(t, int64(42), env.setAdminToolsCalls[0].ID)
	require.NotNil(t, env.setAdminToolsCalls[0].Enabled)
	require.True(t, *env.setAdminToolsCalls[0].Enabled)
}

func TestResolve_AdminGraphQLDisabledByDefault(t *testing.T) {
	env := &mockEnv{publicURL: "https://example.com"}

	_, err := Resolve(context.Background(), env, 7, false)
	require.NoError(t, err)

	require.Empty(t, env.setAdminToolsCalls)
}

func TestResolve_FolderRenamedToDomain(t *testing.T) {
	env := &mockEnv{publicURL: "https://trip2g.com"}

	zipData, err := Resolve(context.Background(), env, 1, false)
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)

	for _, file := range reader.File {
		require.False(t, len(file.Name) >= len(oldPrefix) && file.Name[:len(oldPrefix)] == oldPrefix,
			"file %s should not have old prefix %s", file.Name, oldPrefix)
		require.True(t, len(file.Name) >= len("trip2g.com/") && file.Name[:len("trip2g.com/")] == "trip2g.com/",
			"file %s should start with trip2g.com/", file.Name)
	}
}
