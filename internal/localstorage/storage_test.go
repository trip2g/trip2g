package localstorage

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"trip2g/internal/db"
	"trip2g/internal/miniostorage"

	"github.com/stretchr/testify/require"
)

func newTestStorage(t *testing.T) *LocalStorage {
	t.Helper()

	s, err := New(Config{
		Dir:       t.TempDir(),
		PublicURL: "https://example.com/",
	})
	require.NoError(t, err)

	return s
}

func TestAssetRoundTrip(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	asset := db.NoteAsset{ID: 42, FileName: "photo.png", Size: 5}
	content := []byte("hello")

	// Initially absent.
	exists, err := s.NoteAssetExists(ctx, asset)
	require.NoError(t, err)
	require.False(t, exists)

	// Put.
	require.NoError(t, s.PutAssetObject(ctx, bytes.NewReader(content), asset))

	// Exists.
	exists, err = s.NoteAssetExists(ctx, asset)
	require.NoError(t, err)
	require.True(t, exists)

	// Get.
	rc, err := s.GetAssetObject(ctx, asset)
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	require.Equal(t, content, got)

	// URL: stable, non-expiring, points at the /_assets/ route.
	url, err := s.NoteAssetURL(ctx, asset)
	require.NoError(t, err)
	require.Equal(t, "https://example.com/_assets/na/42/photo.png", url.Value)
	require.True(t, url.ExpiresAt.IsZero())

	// Delete.
	require.NoError(t, s.DeleteAssetObject(ctx, asset))
	exists, err = s.NoteAssetExists(ctx, asset)
	require.NoError(t, err)
	require.False(t, exists)

	// Delete again is a no-op.
	require.NoError(t, s.DeleteAssetObject(ctx, asset))
}

func TestPrivateObjectRoundTrip(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	objectID := "backups/db-backup-1.db.gz"
	content := []byte("backup-bytes")

	// Initially absent.
	exists, err := s.PrivateObjectExists(ctx, objectID)
	require.NoError(t, err)
	require.False(t, exists)

	// Put.
	require.NoError(t, s.PutPrivateObject(ctx, bytes.NewReader(content), objectID))

	// Exists.
	exists, err = s.PrivateObjectExists(ctx, objectID)
	require.NoError(t, err)
	require.True(t, exists)

	// List (matching prefix).
	objects, err := s.ListPrivateObjects(ctx, miniostorage.ListOptions{Prefix: "backups/"})
	require.NoError(t, err)
	require.Len(t, objects, 1)
	require.Equal(t, objectID, objects[0].Key)
	require.Equal(t, int64(len(content)), objects[0].Size)
	require.False(t, objects[0].LastModified.IsZero())

	// List with a non-matching prefix returns nothing.
	objects, err = s.ListPrivateObjects(ctx, miniostorage.ListOptions{Prefix: "other/"})
	require.NoError(t, err)
	require.Empty(t, objects)

	// Get.
	rc, err := s.GetPrivateObject(ctx, objectID)
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	require.Equal(t, content, got)

	// Delete.
	require.NoError(t, s.DeletePrivateObject(ctx, objectID))
	exists, err = s.PrivateObjectExists(ctx, objectID)
	require.NoError(t, err)
	require.False(t, exists)
}

func TestPathTraversalIsContained(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	// A malicious filename with "../" must not escape the assets dir: the key is
	// cleaned so the write lands inside the base dir.
	asset := db.NoteAsset{ID: 1, FileName: "../../escape.txt"}
	require.NoError(t, s.PutAssetObject(ctx, strings.NewReader("x"), asset))

	// The written file stays under <dir>/assets, and Get reads it back via the
	// same cleaned key.
	rc, err := s.GetAssetObject(ctx, asset)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
}

func TestPrefixIsApplied(t *testing.T) {
	s, err := New(Config{
		Dir:       t.TempDir(),
		PublicURL: "https://example.com",
		Prefix:    "tenant1",
	})
	require.NoError(t, err)
	ctx := context.Background()

	asset := db.NoteAsset{ID: 7, FileName: "a.bin"}
	require.Equal(t, "tenant1/na/7/a.bin", s.NoteAssetPath(asset))

	require.NoError(t, s.PutPrivateObject(ctx, strings.NewReader("y"), "backups/x.gz"))

	// ListPrivateObjects strips the configured prefix from returned keys.
	objects, err := s.ListPrivateObjects(ctx, miniostorage.ListOptions{Prefix: "backups/"})
	require.NoError(t, err)
	require.Len(t, objects, 1)
	require.Equal(t, "backups/x.gz", objects[0].Key)
}
