package insertnote_test

import (
	"context"
	"testing"
	"trip2g/internal/case/insertnote"
	"trip2g/internal/db"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

// TestUnhideCalledEvenWhenContentUnchanged verifies that UnhideNotePath is called
// even when the note content hasn't changed. This is a regression test for a bug
// where hidden notes stayed hidden after being pushed with the same content.
func TestUnhideCalledEvenWhenContentUnchanged(t *testing.T) {
	ctx := context.Background()

	unhideCalled := false
	unhidePath := ""

	env := &EnvMock{
		InsertNotePathFunc: func(ctx context.Context, arg db.InsertNotePathParams) (db.InsertNotePathRow, error) {
			// Return existing note with same content hash (simulating unchanged content)
			// The hash is calculated by Resolve and passed in arg.LatestContentHash
			return db.InsertNotePathRow{
				ID:                1,
				VersionCount:      1,                     // Already has a version
				LatestContentHash: arg.LatestContentHash, // Same hash = content unchanged
			}, nil
		},
		UnhideNotePathFunc: func(ctx context.Context, value string) error {
			unhideCalled = true
			unhidePath = value
			return nil
		},
		// These should NOT be called when content is unchanged
		IncrementNoteVersionCountFunc: func(ctx context.Context, arg db.IncrementNoteVersionCountParams) (int64, error) {
			t.Error("IncrementNoteVersionCount should not be called when content is unchanged")
			return 0, nil
		},
		InsertNoteVersionFunc: func(ctx context.Context, arg db.InsertNoteVersionParams) error {
			t.Error("InsertNoteVersion should not be called when content is unchanged")
			return nil
		},
	}

	note := model.RawNote{
		Path:    "test.md",
		Content: "test content",
	}

	pathID, err := insertnote.Resolve(ctx, env, note)
	require.NoError(t, err)
	require.Equal(t, int64(1), pathID)

	// UnhideNotePath MUST be called even when content hasn't changed
	require.True(t, unhideCalled, "UnhideNotePath should be called even when content is unchanged")
	require.Equal(t, "test.md", unhidePath)
}

// TestUnhideCalledWhenContentChanged verifies that UnhideNotePath is called
// when new content is pushed.
func TestUnhideCalledWhenContentChanged(t *testing.T) {
	ctx := context.Background()

	unhideCalled := false
	versionCreated := false

	env := &EnvMock{
		InsertNotePathFunc: func(ctx context.Context, arg db.InsertNotePathParams) (db.InsertNotePathRow, error) {
			// Return existing note with different content hash
			return db.InsertNotePathRow{
				ID:                1,
				VersionCount:      1,
				LatestContentHash: "different-hash",
			}, nil
		},
		UnhideNotePathFunc: func(ctx context.Context, value string) error {
			unhideCalled = true
			return nil
		},
		IncrementNoteVersionCountFunc: func(ctx context.Context, arg db.IncrementNoteVersionCountParams) (int64, error) {
			return 2, nil
		},
		InsertNoteVersionFunc: func(ctx context.Context, arg db.InsertNoteVersionParams) error {
			versionCreated = true
			return nil
		},
	}

	note := model.RawNote{
		Path:    "test.md",
		Content: "new content",
	}

	pathID, err := insertnote.Resolve(ctx, env, note)
	require.NoError(t, err)
	require.Equal(t, int64(1), pathID)

	require.True(t, unhideCalled, "UnhideNotePath should be called")
	require.True(t, versionCreated, "New version should be created when content changed")
}

// TestNewNoteUnhideAndVersionCreated verifies that both UnhideNotePath and
// version creation happen for a brand new note.
func TestNewNoteUnhideAndVersionCreated(t *testing.T) {
	ctx := context.Background()

	unhideCalled := false
	versionCreated := false

	env := &EnvMock{
		InsertNotePathFunc: func(ctx context.Context, arg db.InsertNotePathParams) (db.InsertNotePathRow, error) {
			// Return new note (version count = 0)
			return db.InsertNotePathRow{
				ID:                1,
				VersionCount:      0, // New note, no versions yet
				LatestContentHash: "",
			}, nil
		},
		UnhideNotePathFunc: func(ctx context.Context, value string) error {
			unhideCalled = true
			return nil
		},
		IncrementNoteVersionCountFunc: func(ctx context.Context, arg db.IncrementNoteVersionCountParams) (int64, error) {
			return 1, nil
		},
		InsertNoteVersionFunc: func(ctx context.Context, arg db.InsertNoteVersionParams) error {
			versionCreated = true
			return nil
		},
	}

	note := model.RawNote{
		Path:    "new.md",
		Content: "brand new content",
	}

	pathID, err := insertnote.Resolve(ctx, env, note)
	require.NoError(t, err)
	require.Equal(t, int64(1), pathID)

	require.True(t, unhideCalled, "UnhideNotePath should be called for new note")
	require.True(t, versionCreated, "Version should be created for new note")
}
