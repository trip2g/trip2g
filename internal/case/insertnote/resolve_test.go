package insertnote_test

import (
	"context"
	"testing"
	"trip2g/internal/case/insertnote"
	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/ptr"

	"github.com/stretchr/testify/require"
)

// TestUnhideCalledEvenWhenContentUnchanged verifies that a hidden path comes back
// even when the note content hasn't changed. Writing is the only way to unhide, so
// a restored file with identical bytes must still reappear — and the caller is told
// about it, since the served snapshot has to be reloaded for it.
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
				HiddenBy:          ptr.To(int64(7)),      // and the path is hidden
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
		InsertNoteVersionFunc: func(ctx context.Context, arg db.InsertNoteVersionParams) (int64, error) {
			t.Error("InsertNoteVersion should not be called when content is unchanged")
			return 0, nil
		},
	}

	note := model.RawNote{
		Path:    "test.md",
		Content: "test content",
	}

	saved, err := insertnote.Resolve(ctx, env, note)
	require.NoError(t, err)
	require.Equal(t, int64(1), saved.PathID)
	// No new version is created when content is unchanged.
	require.Equal(t, int64(0), saved.VersionID)

	// UnhideNotePath MUST be called even when content hasn't changed
	require.True(t, unhideCalled, "UnhideNotePath should be called even when content is unchanged")
	require.Equal(t, "test.md", unhidePath)
	require.True(t, saved.Unhidden, "the caller must learn the write brought the path back")
	require.True(t, saved.AffectsSnapshot(), "an unhidden path has to reach the served snapshot")
	require.False(t, saved.Versioned(), "no content change, so no change event")
}

// TestUnhideSkippedWhenNotHidden pins the other half: a write to a visible path
// does not touch the hidden flag at all. A full-vault resync pushes thousands of
// notes, and an unconditional update per note is pure write amplification.
func TestUnhideSkippedWhenNotHidden(t *testing.T) {
	ctx := context.Background()

	env := &EnvMock{
		InsertNotePathFunc: func(_ context.Context, arg db.InsertNotePathParams) (db.InsertNotePathRow, error) {
			return db.InsertNotePathRow{
				ID:                1,
				VersionCount:      1,
				LatestContentHash: arg.LatestContentHash,
			}, nil
		},
		UnhideNotePathFunc: func(_ context.Context, _ string) error {
			t.Error("UnhideNotePath must not be called for a visible path")
			return nil
		},
	}

	saved, err := insertnote.Resolve(ctx, env, model.RawNote{Path: "test.md", Content: "test content"})
	require.NoError(t, err)
	require.False(t, saved.Unhidden)
	require.False(t, saved.AffectsSnapshot(), "nothing changed, so nothing to reload for")
}

// TestUnhideCalledWhenContentChanged verifies that UnhideNotePath is called
// when new content is pushed.
func TestUnhideCalledWhenContentChanged(t *testing.T) {
	ctx := context.Background()

	unhideCalled := false
	versionCreated := false

	env := &EnvMock{
		InsertNotePathFunc: func(ctx context.Context, arg db.InsertNotePathParams) (db.InsertNotePathRow, error) {
			// Return an existing, hidden note with a different content hash
			return db.InsertNotePathRow{
				ID:                1,
				VersionCount:      1,
				LatestContentHash: "different-hash",
				HiddenBy:          ptr.To(int64(7)),
			}, nil
		},
		UnhideNotePathFunc: func(ctx context.Context, value string) error {
			unhideCalled = true
			return nil
		},
		IncrementNoteVersionCountFunc: func(ctx context.Context, arg db.IncrementNoteVersionCountParams) (int64, error) {
			return 2, nil
		},
		InsertNoteVersionFunc: func(ctx context.Context, arg db.InsertNoteVersionParams) (int64, error) {
			versionCreated = true
			return 42, nil
		},
		NoteVersionActorFunc: func(ctx context.Context) model.NoteActor {
			return model.NoteActor{}
		},
	}

	note := model.RawNote{
		Path:    "test.md",
		Content: "new content",
	}

	saved, err := insertnote.Resolve(ctx, env, note)
	require.NoError(t, err)
	require.Equal(t, int64(1), saved.PathID)
	// Resolve returns the id of the note_versions row it inserted.
	require.Equal(t, int64(42), saved.VersionID)

	require.True(t, unhideCalled, "UnhideNotePath should be called")
	require.True(t, versionCreated, "New version should be created when content changed")
}

// TestResolve_RecordsDeliveryAttribution verifies that delivery attribution
// (kind + id) from NoteActor is written to the separate
// note_version_delivery_attribution table via InsertNoteVersionDeliveryAttribution,
// and that InsertNoteVersionParams no longer carries those fields.
func TestResolve_RecordsDeliveryAttribution(t *testing.T) {
	ctx := context.Background()

	var gotAttrib db.InsertNoteVersionDeliveryAttributionParams
	var gotVersion db.InsertNoteVersionParams
	kind := "change"
	var deliveryID int64 = 77

	env := &EnvMock{
		InsertNotePathFunc: func(_ context.Context, arg db.InsertNotePathParams) (db.InsertNotePathRow, error) {
			return db.InsertNotePathRow{ID: 1, VersionCount: 0}, nil
		},
		UnhideNotePathFunc: func(_ context.Context, _ string) error {
			return nil
		},
		IncrementNoteVersionCountFunc: func(_ context.Context, _ db.IncrementNoteVersionCountParams) (int64, error) {
			return 1, nil
		},
		InsertNoteVersionFunc: func(_ context.Context, arg db.InsertNoteVersionParams) (int64, error) {
			gotVersion = arg
			return 42, nil
		},
		InsertNoteVersionDeliveryAttributionFunc: func(_ context.Context, arg db.InsertNoteVersionDeliveryAttributionParams) error {
			gotAttrib = arg
			return nil
		},
		NoteVersionActorFunc: func(_ context.Context) model.NoteActor {
			return model.NoteActor{DeliveryKind: &kind, DeliveryID: &deliveryID}
		},
	}

	_, err := insertnote.Resolve(ctx, env, model.RawNote{Path: "boards/sprint.md", Content: "x"})
	require.NoError(t, err)

	// Delivery fields must NOT appear on the note_versions insert.
	// (InsertNoteVersionParams no longer has those fields at all — this
	// compile-time assertion is implicit; we just confirm the user fields are intact.)
	require.Nil(t, gotVersion.CreatedByUserID)
	require.Nil(t, gotVersion.CreatedByApiKeyID)

	// Attribution must be written to the separate table with the version id
	// returned by InsertNoteVersion (42) and the exact kind/id from the actor.
	require.Equal(t, int64(42), gotAttrib.NoteVersionID)
	require.Equal(t, "change", gotAttrib.DeliveryKind)
	require.EqualValues(t, 77, gotAttrib.DeliveryID)
}

// TestNewNoteCreatesVersionWithoutUnhide verifies that a brand new note gets its
// first version and never touches the hidden flag — there is nothing to unhide.
func TestNewNoteCreatesVersionWithoutUnhide(t *testing.T) {
	ctx := context.Background()

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
			t.Error("a brand new path is not hidden, so there is nothing to unhide")
			return nil
		},
		IncrementNoteVersionCountFunc: func(ctx context.Context, arg db.IncrementNoteVersionCountParams) (int64, error) {
			return 1, nil
		},
		InsertNoteVersionFunc: func(ctx context.Context, arg db.InsertNoteVersionParams) (int64, error) {
			versionCreated = true
			return 7, nil
		},
		NoteVersionActorFunc: func(ctx context.Context) model.NoteActor {
			return model.NoteActor{}
		},
	}

	note := model.RawNote{
		Path:    "new.md",
		Content: "brand new content",
	}

	saved, err := insertnote.Resolve(ctx, env, note)
	require.NoError(t, err)
	require.Equal(t, int64(1), saved.PathID)
	require.Equal(t, int64(7), saved.VersionID)

	require.True(t, versionCreated, "Version should be created for new note")
	require.False(t, saved.Unhidden, "a new path was never hidden")
	require.True(t, saved.Versioned())
}
