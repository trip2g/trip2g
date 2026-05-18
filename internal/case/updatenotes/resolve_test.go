package updatenotes_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/updatenotes"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

// mockEnv is a hand-written mock implementing updatenotes.Env.
type mockEnv struct {
	latestNoteViews          func() *appmodel.NoteViews
	insertNote               func(ctx context.Context, note appmodel.RawNote) (int64, error)
	hideNotePath             func(ctx context.Context, params db.HideNotePathParams) error
	prepareLatestNotes       func(ctx context.Context, partial bool) (*appmodel.NoteViews, error)
	handleLatestNotesAfterSave func(ctx context.Context, pathIDs []int64) error
}

func (m *mockEnv) LatestNoteViews() *appmodel.NoteViews {
	return m.latestNoteViews()
}

func (m *mockEnv) InsertNote(ctx context.Context, note appmodel.RawNote) (int64, error) {
	if m.insertNote == nil {
		panic("unexpected call to InsertNote")
	}
	return m.insertNote(ctx, note)
}

func (m *mockEnv) HideNotePath(ctx context.Context, params db.HideNotePathParams) error {
	if m.hideNotePath == nil {
		panic("unexpected call to HideNotePath")
	}
	return m.hideNotePath(ctx, params)
}

func (m *mockEnv) PrepareLatestNotes(ctx context.Context, partial bool) (*appmodel.NoteViews, error) {
	return m.prepareLatestNotes(ctx, partial)
}

func (m *mockEnv) HandleLatestNotesAfterSave(ctx context.Context, pathIDs []int64) error {
	return m.handleLatestNotesAfterSave(ctx, pathIDs)
}

func hashContent(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

func makeNVSWithNote(path, content string, pathID int64) *appmodel.NoteViews {
	nvs := appmodel.NewNoteViews()
	note := &appmodel.NoteView{
		PathID:    pathID,
		Path:      path,
		Permalink: path, // GetByPath uses nv.Map keyed by Permalink
		Content:   []byte(content),
	}
	nvs.RegisterNote(note)
	return nvs
}

func noopPrepare(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
	return appmodel.NewNoteViews(), nil
}

func noopHandle(_ context.Context, _ []int64) error { return nil }

func TestResolve_UpsertBasic(t *testing.T) {
	ctx := context.Background()

	var insertedNote appmodel.RawNote
	handleCalled := false
	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return appmodel.NewNoteViews() },
		insertNote: func(_ context.Context, note appmodel.RawNote) (int64, error) {
			insertedNote = note
			return 10, nil
		},
		prepareLatestNotes: noopPrepare,
		handleLatestNotesAfterSave: func(_ context.Context, ids []int64) error {
			handleCalled = true
			return nil
		},
	}

	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Upsert: &model.NoteChangeUpsertInput{Path: "note.md", Content: "hello"}},
		},
	}

	result, err := updatenotes.Resolve(ctx, env, input)
	require.NoError(t, err)

	payload, ok := result.(model.UpdateNotesSuccessPayload)
	require.True(t, ok, "expected UpdateNotesSuccessPayload, got %T", result)
	require.Equal(t, []string{"note.md"}, payload.Paths)
	require.Equal(t, "note.md", insertedNote.Path)
	require.Equal(t, "hello", insertedNote.Content)
	require.True(t, handleCalled, "HandleLatestNotesAfterSave must be called after successful writes")
}

func TestResolve_UpsertWithCorrectHash(t *testing.T) {
	ctx := context.Background()
	existingContent := "existing content"
	correctHash := hashContent(existingContent)

	nvs := makeNVSWithNote("note.md", existingContent, 1)

	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return nvs },
		insertNote: func(_ context.Context, _ appmodel.RawNote) (int64, error) {
			return 11, nil
		},
		prepareLatestNotes:       noopPrepare,
		handleLatestNotesAfterSave: noopHandle,
	}

	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Upsert: &model.NoteChangeUpsertInput{Path: "note.md", Content: "new content", ExpectedHash: &correctHash}},
		},
	}

	result, err := updatenotes.Resolve(ctx, env, input)
	require.NoError(t, err)

	payload, ok := result.(model.UpdateNotesSuccessPayload)
	require.True(t, ok, "expected UpdateNotesSuccessPayload, got %T", result)
	require.Equal(t, []string{"note.md"}, payload.Paths)
}

func TestResolve_UpsertWithWrongHash(t *testing.T) {
	ctx := context.Background()
	existingContent := "existing content"
	actualHash := hashContent(existingContent)
	wrongHash := "wronghash=="

	nvs := makeNVSWithNote("note.md", existingContent, 1)

	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return nvs },
		insertNote: func(_ context.Context, _ appmodel.RawNote) (int64, error) {
			t.Fatal("InsertNote should not be called on hash mismatch")
			return 0, nil
		},
		prepareLatestNotes:       noopPrepare,
		handleLatestNotesAfterSave: noopHandle,
	}

	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Upsert: &model.NoteChangeUpsertInput{Path: "note.md", Content: "new content", ExpectedHash: &wrongHash}},
		},
	}

	result, err := updatenotes.Resolve(ctx, env, input)
	require.NoError(t, err)

	payload, ok := result.(model.UpdateNotesHashMismatchPayload)
	require.True(t, ok, "expected UpdateNotesHashMismatchPayload, got %T", result)
	require.Equal(t, "note.md", payload.Path)
	require.Equal(t, actualHash, payload.ActualHash)
}

func TestResolve_PatchFound(t *testing.T) {
	ctx := context.Background()
	existingContent := "hello world"
	nvs := makeNVSWithNote("note.md", existingContent, 5)

	var insertedNote appmodel.RawNote
	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return nvs },
		insertNote: func(_ context.Context, note appmodel.RawNote) (int64, error) {
			insertedNote = note
			return 12, nil
		},
		prepareLatestNotes:       noopPrepare,
		handleLatestNotesAfterSave: noopHandle,
	}

	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Patch: &model.NoteChangePatchInput{Path: "note.md", Find: "world", Replace: "Go"}},
		},
	}

	result, err := updatenotes.Resolve(ctx, env, input)
	require.NoError(t, err)

	payload, ok := result.(model.UpdateNotesSuccessPayload)
	require.True(t, ok, "expected UpdateNotesSuccessPayload, got %T", result)
	require.Equal(t, []string{"note.md"}, payload.Paths)
	require.Equal(t, "hello Go", insertedNote.Content)
}

func TestResolve_PatchNotFound(t *testing.T) {
	ctx := context.Background()
	nvs := makeNVSWithNote("note.md", "hello world", 5)

	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return nvs },
		insertNote: func(_ context.Context, _ appmodel.RawNote) (int64, error) {
			t.Fatal("InsertNote should not be called when find string is absent")
			return 0, nil
		},
		prepareLatestNotes:       noopPrepare,
		handleLatestNotesAfterSave: noopHandle,
	}

	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Patch: &model.NoteChangePatchInput{Path: "note.md", Find: "missing", Replace: "X"}},
		},
	}

	result, err := updatenotes.Resolve(ctx, env, input)
	require.NoError(t, err)

	payload, ok := result.(model.UpdateNotesPatchNotFoundPayload)
	require.True(t, ok, "expected UpdateNotesPatchNotFoundPayload, got %T", result)
	require.Equal(t, "note.md", payload.Path)
	require.Equal(t, "missing", payload.Find)
}

func TestResolve_PatchMultipleOccurrences(t *testing.T) {
	ctx := context.Background()
	nvs := makeNVSWithNote("note.md", "foo bar foo", 5)

	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return nvs },
		insertNote: func(_ context.Context, _ appmodel.RawNote) (int64, error) {
			t.Fatal("InsertNote should not be called on ambiguous patch")
			return 0, nil
		},
		prepareLatestNotes:       noopPrepare,
		handleLatestNotesAfterSave: noopHandle,
	}

	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Patch: &model.NoteChangePatchInput{Path: "note.md", Find: "foo", Replace: "baz"}},
		},
	}

	result, err := updatenotes.Resolve(ctx, env, input)
	require.NoError(t, err)

	// Multiple occurrences are treated as "not found" to avoid ambiguous patching
	payload, ok := result.(model.UpdateNotesPatchNotFoundPayload)
	require.True(t, ok, "expected UpdateNotesPatchNotFoundPayload for multiple occurrences, got %T", result)
	require.Equal(t, "note.md", payload.Path)
	require.Equal(t, "foo", payload.Find)
}

func TestResolve_PatchNoteMissing(t *testing.T) {
	ctx := context.Background()

	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return appmodel.NewNoteViews() },
		insertNote: func(_ context.Context, _ appmodel.RawNote) (int64, error) {
			t.Fatal("InsertNote should not be called when note is missing")
			return 0, nil
		},
		prepareLatestNotes:       noopPrepare,
		handleLatestNotesAfterSave: noopHandle,
	}

	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Patch: &model.NoteChangePatchInput{Path: "missing.md", Find: "x", Replace: "y"}},
		},
	}

	result, err := updatenotes.Resolve(ctx, env, input)
	require.NoError(t, err)

	_, ok := result.(model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload when note not found, got %T", result)
}

func TestResolve_PatchWithWrongHash(t *testing.T) {
	ctx := context.Background()
	existingContent := "hello world"
	actualHash := hashContent(existingContent)
	wrongHash := "badhash=="

	nvs := makeNVSWithNote("note.md", existingContent, 5)

	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return nvs },
		insertNote: func(_ context.Context, _ appmodel.RawNote) (int64, error) {
			t.Fatal("InsertNote should not be called on hash mismatch")
			return 0, nil
		},
		prepareLatestNotes:       noopPrepare,
		handleLatestNotesAfterSave: noopHandle,
	}

	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Patch: &model.NoteChangePatchInput{Path: "note.md", Find: "world", Replace: "Go", ExpectedHash: &wrongHash}},
		},
	}

	result, err := updatenotes.Resolve(ctx, env, input)
	require.NoError(t, err)

	payload, ok := result.(model.UpdateNotesHashMismatchPayload)
	require.True(t, ok, "expected UpdateNotesHashMismatchPayload, got %T", result)
	require.Equal(t, "note.md", payload.Path)
	require.Equal(t, actualHash, payload.ActualHash)
}

func TestResolve_Hide(t *testing.T) {
	ctx := context.Background()

	var hiddenPath string
	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return appmodel.NewNoteViews() },
		insertNote: func(_ context.Context, _ appmodel.RawNote) (int64, error) {
			t.Fatal("InsertNote should not be called for hide")
			return 0, nil
		},
		hideNotePath: func(_ context.Context, params db.HideNotePathParams) error {
			hiddenPath = params.Value
			return nil
		},
		prepareLatestNotes:       noopPrepare,
		handleLatestNotesAfterSave: noopHandle,
	}

	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Hide: &model.NoteChangeHideInput{Path: "gone.md"}},
		},
	}

	result, err := updatenotes.Resolve(ctx, env, input)
	require.NoError(t, err)

	payload, ok := result.(model.UpdateNotesSuccessPayload)
	require.True(t, ok, "expected UpdateNotesSuccessPayload, got %T", result)
	require.Equal(t, []string{"gone.md"}, payload.Paths)
	require.Equal(t, "gone.md", hiddenPath)
}

func TestResolve_EmptyChangeSkipped(t *testing.T) {
	// malformed input: all change types nil → skip silently, do not error
	ctx := context.Background()

	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return appmodel.NewNoteViews() },
		insertNote: func(_ context.Context, _ appmodel.RawNote) (int64, error) {
			t.Fatal("InsertNote should not be called for empty change")
			return 0, nil
		},
		hideNotePath: func(_ context.Context, _ db.HideNotePathParams) error {
			t.Fatal("HideNotePath should not be called for empty change")
			return nil
		},
		prepareLatestNotes:       noopPrepare,
		handleLatestNotesAfterSave: noopHandle,
	}

	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{}, // all nil
		},
	}

	result, err := updatenotes.Resolve(ctx, env, input)
	require.NoError(t, err)

	payload, ok := result.(model.UpdateNotesSuccessPayload)
	require.True(t, ok, "expected UpdateNotesSuccessPayload, got %T", result)
	require.Empty(t, payload.Paths)
}

func TestResolve_MixedBatch(t *testing.T) {
	ctx := context.Background()
	existingContent := "original text"
	nvs := makeNVSWithNote("patch-me.md", existingContent, 20)

	insertCallCount := 0
	hideCallCount := 0

	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return nvs },
		insertNote: func(_ context.Context, _ appmodel.RawNote) (int64, error) {
			insertCallCount++
			return int64(100 + insertCallCount), nil
		},
		hideNotePath: func(_ context.Context, _ db.HideNotePathParams) error {
			hideCallCount++
			return nil
		},
		prepareLatestNotes:       noopPrepare,
		handleLatestNotesAfterSave: noopHandle,
	}

	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Patch: &model.NoteChangePatchInput{Path: "patch-me.md", Find: "original", Replace: "updated"}},
			{Upsert: &model.NoteChangeUpsertInput{Path: "new.md", Content: "brand new"}},
			{Hide: &model.NoteChangeHideInput{Path: "old.md"}},
		},
	}

	result, err := updatenotes.Resolve(ctx, env, input)
	require.NoError(t, err)

	payload, ok := result.(model.UpdateNotesSuccessPayload)
	require.True(t, ok, "expected UpdateNotesSuccessPayload, got %T", result)
	require.Len(t, payload.Paths, 3)
	require.Contains(t, payload.Paths, "patch-me.md")
	require.Contains(t, payload.Paths, "new.md")
	require.Contains(t, payload.Paths, "old.md")
	require.Equal(t, 2, insertCallCount)
	require.Equal(t, 1, hideCallCount)
}
