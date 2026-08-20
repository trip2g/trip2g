package updatenotes_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/appreq"
	"trip2g/internal/case/updatenotes"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

// mockEnv is a hand-written mock implementing updatenotes.Env.
type mockEnv struct {
	latestNoteViews            func() *appmodel.NoteViews
	insertNote                 func(ctx context.Context, note appmodel.RawNote) (appmodel.NoteSaveResult, error)
	hideNotePath               func(ctx context.Context, params db.HideNotePathParams) error
	prepareLatestNotes         func(ctx context.Context, partial bool) (*appmodel.NoteViews, error)
	handleLatestNotesAfterSave func(ctx context.Context, pathIDs []int64) error
}

func (m *mockEnv) LatestNoteViews() *appmodel.NoteViews {
	return m.latestNoteViews()
}

func (m *mockEnv) InsertNote(ctx context.Context, note appmodel.RawNote) (appmodel.NoteSaveResult, error) {
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
		Permalink: path, // must be non-empty so RegisterNote stores it in Map; PathMap[note.Path] is what the impl uses
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
		latestNoteViews: appmodel.NewNoteViews,
		insertNote: func(_ context.Context, note appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			insertedNote = note
			return appmodel.NoteSaveResult{PathID: 10, VersionID: 110}, nil
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

// TestResolve_UpsertReportsOwnVersion pins the race-free fix: updated[].versionId
// must be the version the WRITE itself inserted, not the latest version visible in
// the post-write reload. Under concurrent same-board editing the reload can already
// carry a peer's newer version for the same note; reporting that as "ours" would
// briefly suppress a genuine peer event on the saving client. Here the reload reports
// a peer's version (999) for the saved note, but the resolver must surface our own
// inserted version (42).
func TestResolve_UpsertReportsOwnVersion(t *testing.T) {
	ctx := context.Background()

	const pathID int64 = 7
	const ownVersion int64 = 42
	const peerVersion int64 = 999

	// The post-write reload sees a peer's newer version for the same note.
	reloaded := appmodel.NewNoteViews()
	reloaded.RegisterNote(&appmodel.NoteView{
		PathID:    pathID,
		Path:      "note.md",
		Permalink: "note.md",
		VersionID: peerVersion,
		Content:   []byte("peer content"),
	})

	env := &mockEnv{
		latestNoteViews: appmodel.NewNoteViews,
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			return appmodel.NoteSaveResult{PathID: pathID, VersionID: ownVersion}, nil
		},
		prepareLatestNotes: func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
			return reloaded, nil
		},
		handleLatestNotesAfterSave: noopHandle,
	}

	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Upsert: &model.NoteChangeUpsertInput{Path: "note.md", Content: "my content"}},
		},
	}

	result, err := updatenotes.Resolve(ctx, env, input)
	require.NoError(t, err)

	payload, ok := result.(model.UpdateNotesSuccessPayload)
	require.True(t, ok, "expected UpdateNotesSuccessPayload, got %T", result)
	require.Len(t, payload.Updated, 1)
	require.Equal(t, "note.md", payload.Updated[0].Path)
	require.Equal(t, ownVersion, payload.Updated[0].VersionID,
		"must report the write's OWN inserted version, not the peer's reload version")
}

// TestResolve_UpsertUnchangedIsInert pins the unchanged-content path: the write
// stored nothing and unhid nothing, so there is no reload and no change event —
// otherwise an idempotent agent write re-triggers the webhooks that caused it. The
// path is still reported, with the version it already has.
func TestResolve_UpsertUnchangedIsInert(t *testing.T) {
	ctx := context.Background()

	const pathID int64 = 8
	const currentVersion int64 = 500

	current := appmodel.NewNoteViews()
	current.RegisterNote(&appmodel.NoteView{
		PathID:    pathID,
		Path:      "note.md",
		Permalink: "note.md",
		VersionID: currentVersion,
		Content:   []byte("same content"),
	})

	reloads := 0
	handles := 0
	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return current },
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			return appmodel.NoteSaveResult{PathID: pathID, VersionID: 0}, nil // content unchanged → no new version
		},
		prepareLatestNotes: func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
			reloads++
			return current, nil
		},
		handleLatestNotesAfterSave: func(_ context.Context, _ []int64) error {
			handles++
			return nil
		},
	}

	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Upsert: &model.NoteChangeUpsertInput{Path: "note.md", Content: "same content"}},
		},
	}

	result, err := updatenotes.Resolve(ctx, env, input)
	require.NoError(t, err)

	payload, ok := result.(model.UpdateNotesSuccessPayload)
	require.True(t, ok, "expected UpdateNotesSuccessPayload, got %T", result)
	require.Len(t, payload.Updated, 1)
	require.Equal(t, currentVersion, payload.Updated[0].VersionID,
		"version 0 from the write means unchanged content → report the version it already has")
	require.Zero(t, reloads, "nothing changed, so nothing to reload for")
	require.Zero(t, handles, "a write that stored nothing must raise no change events")
}

func TestResolve_UpsertWithCorrectHash(t *testing.T) {
	ctx := context.Background()
	existingContent := "existing content"
	correctHash := hashContent(existingContent)

	nvs := makeNVSWithNote("note.md", existingContent, 1)

	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return nvs },
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			return appmodel.NoteSaveResult{PathID: 11, VersionID: 111}, nil
		},
		prepareLatestNotes:         noopPrepare,
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
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			t.Fatal("InsertNote should not be called on hash mismatch")
			return appmodel.NoteSaveResult{PathID: 0, VersionID: 0}, nil
		},
		prepareLatestNotes:         noopPrepare,
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

// TestResolve_UpsertCreateOnly pins the create-only sentinel: expectedHash == ""
// means "expect the note absent". An absent note hashes to "" → match → create;
// any existing note hashes non-empty → HashMismatch (never overwritten), even when
// its content is empty.
func TestResolve_UpsertCreateOnly(t *testing.T) {
	emptyHash := ""

	tests := []struct {
		name string
		// existing is the note already present at "note.md"; nil means absent.
		existing *struct {
			content string
			pathID  int64
		}
		wantCreated bool
	}{
		{
			name:        "absent note is created",
			existing:    nil,
			wantCreated: true,
		},
		{
			name: "existing note is not overwritten",
			existing: &struct {
				content string
				pathID  int64
			}{content: "existing content", pathID: 1},
			wantCreated: false,
		},
		{
			name: "existing note with empty content still mismatches",
			existing: &struct {
				content string
				pathID  int64
			}{content: "", pathID: 2},
			wantCreated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			var nvs *appmodel.NoteViews
			if tt.existing != nil {
				nvs = makeNVSWithNote("note.md", tt.existing.content, tt.existing.pathID)
			} else {
				nvs = appmodel.NewNoteViews()
			}

			var insertedNote appmodel.RawNote
			env := &mockEnv{
				latestNoteViews: func() *appmodel.NoteViews { return nvs },
				insertNote: func(_ context.Context, note appmodel.RawNote) (appmodel.NoteSaveResult, error) {
					if !tt.wantCreated {
						t.Fatal("InsertNote must not be called when an existing note blocks the create")
					}
					insertedNote = note
					return appmodel.NoteSaveResult{PathID: 100, VersionID: 200}, nil
				},
				prepareLatestNotes:         noopPrepare,
				handleLatestNotesAfterSave: noopHandle,
			}

			input := model.UpdateNotesInput{
				Changes: []model.NoteChangeInput{
					{Upsert: &model.NoteChangeUpsertInput{Path: "note.md", Content: "# New\n", ExpectedHash: &emptyHash}},
				},
			}

			result, err := updatenotes.Resolve(ctx, env, input)
			require.NoError(t, err)

			if tt.wantCreated {
				payload, ok := result.(model.UpdateNotesSuccessPayload)
				require.True(t, ok, "expected UpdateNotesSuccessPayload, got %T", result)
				require.Equal(t, []string{"note.md"}, payload.Paths)
				require.Equal(t, "note.md", insertedNote.Path)
				require.Equal(t, "# New\n", insertedNote.Content)
				return
			}

			payload, ok := result.(model.UpdateNotesHashMismatchPayload)
			require.True(t, ok, "expected UpdateNotesHashMismatchPayload, got %T", result)
			require.Equal(t, "note.md", payload.Path)
			// An existing note always hashes non-empty, so the actualHash carried back
			// is never "" — proving the create was correctly rejected.
			require.NotEmpty(t, payload.ActualHash)
			require.Equal(t, hashContent(tt.existing.content), payload.ActualHash)
		})
	}
}

func TestResolve_PatchFound(t *testing.T) {
	ctx := context.Background()
	existingContent := "hello world"
	nvs := makeNVSWithNote("note.md", existingContent, 5)

	var insertedNote appmodel.RawNote
	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return nvs },
		insertNote: func(_ context.Context, note appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			insertedNote = note
			return appmodel.NoteSaveResult{PathID: 12, VersionID: 112}, nil
		},
		prepareLatestNotes:         noopPrepare,
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
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			t.Fatal("InsertNote should not be called when find string is absent")
			return appmodel.NoteSaveResult{PathID: 0, VersionID: 0}, nil
		},
		prepareLatestNotes:         noopPrepare,
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
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			t.Fatal("InsertNote should not be called on ambiguous patch")
			return appmodel.NoteSaveResult{PathID: 0, VersionID: 0}, nil
		},
		prepareLatestNotes:         noopPrepare,
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
		latestNoteViews: appmodel.NewNoteViews,
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			t.Fatal("InsertNote should not be called when note is missing")
			return appmodel.NoteSaveResult{PathID: 0, VersionID: 0}, nil
		},
		prepareLatestNotes:         noopPrepare,
		handleLatestNotesAfterSave: noopHandle,
	}

	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Patch: &model.NoteChangePatchInput{Path: "missing.md", Find: "x", Replace: "y"}},
		},
	}

	result, err := updatenotes.Resolve(ctx, env, input)
	require.NoError(t, err)

	_, ok := result.(*model.ErrorPayload)
	require.True(t, ok, "expected *ErrorPayload when note not found, got %T", result)
}

func TestResolve_PatchWithWrongHash(t *testing.T) {
	ctx := context.Background()
	existingContent := "hello world"
	actualHash := hashContent(existingContent)
	wrongHash := "badhash=="

	nvs := makeNVSWithNote("note.md", existingContent, 5)

	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return nvs },
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			t.Fatal("InsertNote should not be called on hash mismatch")
			return appmodel.NoteSaveResult{PathID: 0, VersionID: 0}, nil
		},
		prepareLatestNotes:         noopPrepare,
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
	prepareCount := 0
	env := &mockEnv{
		latestNoteViews: appmodel.NewNoteViews,
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			t.Fatal("InsertNote should not be called for hide")
			return appmodel.NoteSaveResult{PathID: 0, VersionID: 0}, nil
		},
		hideNotePath: func(_ context.Context, params db.HideNotePathParams) error {
			hiddenPath = params.Value
			return nil
		},
		prepareLatestNotes: func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
			prepareCount++
			return appmodel.NewNoteViews(), nil
		},
		handleLatestNotesAfterSave: func(_ context.Context, _ []int64) error {
			t.Fatal("HandleLatestNotesAfterSave should not be called for hide-only batch")
			return nil
		},
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
	// Hide must reload NoteViews so the hidden path stops resolving on the site.
	require.Equal(t, 1, prepareCount)
}

func TestResolve_EmptyChangeSkipped(t *testing.T) {
	// malformed input: all change types nil → skip silently, do not error
	ctx := context.Background()

	env := &mockEnv{
		latestNoteViews: appmodel.NewNoteViews,
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			t.Fatal("InsertNote should not be called for empty change")
			return appmodel.NoteSaveResult{PathID: 0, VersionID: 0}, nil
		},
		hideNotePath: func(_ context.Context, _ db.HideNotePathParams) error {
			t.Fatal("HideNotePath should not be called for empty change")
			return nil
		},
		prepareLatestNotes:         noopPrepare,
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
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			insertCallCount++
			return appmodel.NoteSaveResult{PathID: int64(100 + insertCallCount), VersionID: int64(200 + insertCallCount)}, nil
		},
		hideNotePath: func(_ context.Context, _ db.HideNotePathParams) error {
			hideCallCount++
			return nil
		},
		prepareLatestNotes:         noopPrepare,
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

func TestResolve_WriteDeniedOutOfPattern(t *testing.T) {
	req := &appreq.Request{WebhookWritePatterns: []string{"boards/**"}}
	ctx := appreq.NewContext(context.Background(), req)

	env := &mockEnv{
		latestNoteViews: appmodel.NewNoteViews,
		// insertNote intentionally nil — must NOT be reached.
	}

	out, err := updatenotes.Resolve(ctx, env, model.UpdateNotesInput{
		ApiKey: db.ApiKey{},
		Changes: []model.NoteChangeInput{
			{Upsert: &model.NoteChangeUpsertInput{Path: "secrets/private.md", Content: "x"}},
		},
	})
	require.NoError(t, err)
	ep, ok := out.(*model.ErrorPayload)
	require.True(t, ok)
	require.Contains(t, ep.Message, "write denied for path: secrets/private.md")
}

func TestResolve_WriteAllowedInPattern(t *testing.T) {
	req := &appreq.Request{WebhookWritePatterns: []string{"boards/**"}}
	ctx := appreq.NewContext(context.Background(), req)

	called := false
	env := &mockEnv{
		latestNoteViews: appmodel.NewNoteViews,
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			called = true
			return appmodel.NoteSaveResult{PathID: 1, VersionID: 0}, nil
		},
		prepareLatestNotes:         noopPrepare,
		handleLatestNotesAfterSave: noopHandle,
	}

	out, err := updatenotes.Resolve(ctx, env, model.UpdateNotesInput{
		ApiKey: db.ApiKey{},
		Changes: []model.NoteChangeInput{
			{Upsert: &model.NoteChangeUpsertInput{Path: "boards/sprint.md", Content: "x"}},
		},
	})
	require.NoError(t, err)
	require.IsType(t, model.UpdateNotesSuccessPayload{}, out)
	require.True(t, called)
}

// F2: scoped token (WebhookDeliveryKind set) + empty write_patterns → deny-all.
func TestResolve_ScopedToken_EmptyWritePatterns_DenyAll(t *testing.T) {
	tests := []struct {
		name   string
		change model.NoteChangeInput
	}{
		{
			name:   "upsert denied",
			change: model.NoteChangeInput{Upsert: &model.NoteChangeUpsertInput{Path: "any.md", Content: "x"}},
		},
		{
			name:   "patch denied",
			change: model.NoteChangeInput{Patch: &model.NoteChangePatchInput{Path: "any.md", Find: "x", Replace: "y"}},
		},
		{
			name:   "hide denied",
			change: model.NoteChangeInput{Hide: &model.NoteChangeHideInput{Path: "any.md"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Scoped token: DeliveryKind set, WritePatterns deliberately empty (CRUD default).
			req := &appreq.Request{
				WebhookDeliveryKind:  "change",
				WebhookWritePatterns: []string{}, // empty = should be deny-all when scoped
			}
			ctx := appreq.NewContext(context.Background(), req)

			nvs := makeNVSWithNote("any.md", "existing", 1)
			env := &mockEnv{
				latestNoteViews: func() *appmodel.NoteViews { return nvs },
				// insertNote and hideNotePath intentionally nil — must NOT be reached.
			}

			out, err := updatenotes.Resolve(ctx, env, model.UpdateNotesInput{
				ApiKey:  db.ApiKey{},
				Changes: []model.NoteChangeInput{tc.change},
			})
			require.NoError(t, err)
			ep, ok := out.(*model.ErrorPayload)
			require.True(t, ok, "expected *ErrorPayload (write denied), got %T", out)
			require.Contains(t, ep.Message, "write denied")
		})
	}
}

// F2: scoped token + non-empty matching write_patterns → allowed.
func TestResolve_ScopedToken_MatchingPattern_Allowed(t *testing.T) {
	req := &appreq.Request{
		WebhookDeliveryKind:  "cron",
		WebhookWritePatterns: []string{"notes/**"},
	}
	ctx := appreq.NewContext(context.Background(), req)

	called := false
	env := &mockEnv{
		latestNoteViews: appmodel.NewNoteViews,
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			called = true
			return appmodel.NoteSaveResult{PathID: 1, VersionID: 0}, nil
		},
		prepareLatestNotes:         noopPrepare,
		handleLatestNotesAfterSave: noopHandle,
	}

	out, err := updatenotes.Resolve(ctx, env, model.UpdateNotesInput{
		ApiKey: db.ApiKey{},
		Changes: []model.NoteChangeInput{
			{Upsert: &model.NoteChangeUpsertInput{Path: "notes/todo.md", Content: "x"}},
		},
	})
	require.NoError(t, err)
	require.IsType(t, model.UpdateNotesSuccessPayload{}, out)
	require.True(t, called)
}

// Scoped token + leading-slash upsert path: small models sometimes prepend "/"
// to the path. ScopedKB normalizes candidates for its own glob check, but the
// raw path reaches updatenotes, where webhookWriteDenied matches it against
// write_patterns with no normalization — "/concepts/x.md" vs "concepts/**" is
// wrongly DENIED. Desired: normalize before matching, so the write is allowed.
func TestResolve_ScopedToken_LeadingSlashPathNotDenied(t *testing.T) {
	req := &appreq.Request{
		WebhookDeliveryKind:  "change",
		WebhookWritePatterns: []string{"concepts/**"},
	}
	ctx := appreq.NewContext(context.Background(), req)

	called := false
	env := &mockEnv{
		latestNoteViews: appmodel.NewNoteViews,
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			called = true
			return appmodel.NoteSaveResult{PathID: 1, VersionID: 0}, nil
		},
		prepareLatestNotes:         noopPrepare,
		handleLatestNotesAfterSave: noopHandle,
	}

	out, err := updatenotes.Resolve(ctx, env, model.UpdateNotesInput{
		ApiKey: db.ApiKey{},
		Changes: []model.NoteChangeInput{
			{Upsert: &model.NoteChangeUpsertInput{Path: "/concepts/x.md", Content: "x"}},
		},
	})
	require.NoError(t, err)
	require.IsType(t, model.UpdateNotesSuccessPayload{}, out,
		"leading-slash path inside write scope must not be denied")
	require.True(t, called)
}

// The mirror-image defect: the STORED PATTERN carries the leading slash. Paths
// are normalized before matching, but patterns were passed raw to MatchesAny,
// so write_patterns ["/concepts/**"] never matched the canonical slash-less
// path and every write was wrongly denied. Desired: patterns are normalized
// too, so the write is allowed.
func TestResolve_ScopedToken_SlashPrefixedPatternNotDenied(t *testing.T) {
	req := &appreq.Request{
		WebhookDeliveryKind:  "change",
		WebhookWritePatterns: []string{"/concepts/**"},
	}
	ctx := appreq.NewContext(context.Background(), req)

	called := false
	env := &mockEnv{
		latestNoteViews: appmodel.NewNoteViews,
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			called = true
			return appmodel.NoteSaveResult{PathID: 1, VersionID: 0}, nil
		},
		prepareLatestNotes:         noopPrepare,
		handleLatestNotesAfterSave: noopHandle,
	}

	out, err := updatenotes.Resolve(ctx, env, model.UpdateNotesInput{
		ApiKey: db.ApiKey{},
		Changes: []model.NoteChangeInput{
			{Upsert: &model.NoteChangeUpsertInput{Path: "concepts/x.md", Content: "x"}},
		},
	})
	require.NoError(t, err)
	require.IsType(t, model.UpdateNotesSuccessPayload{}, out,
		"slash-prefixed write pattern must still allow its in-scope paths")
	require.True(t, called)
}

// Same leading-slash defect for the Hide branch: the raw path reached
// webhookWriteDenied and HideNotePath un-normalized, so a scoped token hiding
// "/concepts/x.md" was wrongly denied against "concepts/**".
func TestResolve_ScopedToken_LeadingSlashHideNotDenied(t *testing.T) {
	req := &appreq.Request{
		WebhookDeliveryKind:  "change",
		WebhookWritePatterns: []string{"concepts/**"},
	}
	ctx := appreq.NewContext(context.Background(), req)

	called := false
	env := &mockEnv{
		latestNoteViews: appmodel.NewNoteViews,
		hideNotePath: func(_ context.Context, params db.HideNotePathParams) error {
			called = true
			require.Equal(t, "concepts/x.md", params.Value, "HideNotePath must receive the normalized path")
			return nil
		},
		prepareLatestNotes: noopPrepare,
	}

	out, err := updatenotes.Resolve(ctx, env, model.UpdateNotesInput{
		ApiKey: db.ApiKey{},
		Changes: []model.NoteChangeInput{
			{Hide: &model.NoteChangeHideInput{Path: "/concepts/x.md"}},
		},
	})
	require.NoError(t, err)
	require.IsType(t, model.UpdateNotesSuccessPayload{}, out,
		"leading-slash hide path inside write scope must not be denied")
	require.True(t, called)
}

// A scoped credential (WebhookScoped=true) minted WITHOUT a DeliveryKind —
// e.g. a future visitor-facing shortapitoken, since the dk claim is omitempty —
// must still be deny-all on empty write_patterns. The deny-all branch used to
// key off DeliveryKind, so such a token fell into the legacy empty=allow-all
// path and could write ANY note.
func TestResolve_ScopedNoDeliveryKind_EmptyWritePatterns_DenyAll(t *testing.T) {
	tests := []struct {
		name   string
		change model.NoteChangeInput
	}{
		{
			name:   "upsert outside any scope denied",
			change: model.NoteChangeInput{Upsert: &model.NoteChangeUpsertInput{Path: "secrets/private.md", Content: "pwned"}},
		},
		{
			name:   "patch denied",
			change: model.NoteChangeInput{Patch: &model.NoteChangePatchInput{Path: "secrets/private.md", Find: "x", Replace: "y"}},
		},
		{
			name:   "hide denied",
			change: model.NoteChangeInput{Hide: &model.NoteChangeHideInput{Path: "secrets/private.md"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Scoped shortapitoken with NO DeliveryKind and NO write scope.
			req := &appreq.Request{
				WebhookScoped:        true,
				WebhookWritePatterns: []string{},
			}
			ctx := appreq.NewContext(context.Background(), req)

			nvs := makeNVSWithNote("secrets/private.md", "x", 1)
			env := &mockEnv{
				latestNoteViews: func() *appmodel.NoteViews { return nvs },
				// insertNote and hideNotePath intentionally nil — must NOT be reached.
			}

			out, err := updatenotes.Resolve(ctx, env, model.UpdateNotesInput{
				ApiKey:  db.ApiKey{},
				Changes: []model.NoteChangeInput{tc.change},
			})
			require.NoError(t, err)
			ep, ok := out.(*model.ErrorPayload)
			require.True(t, ok, "expected *ErrorPayload (write denied), got %T", out)
			require.Contains(t, ep.Message, "write denied")
		})
	}
}

// Scoped, no DeliveryKind, non-empty matching write_patterns → allowed
// (the fix must not over-deny in-scope writes).
func TestResolve_ScopedNoDeliveryKind_MatchingPattern_Allowed(t *testing.T) {
	req := &appreq.Request{
		WebhookScoped:        true,
		WebhookWritePatterns: []string{"notes/**"},
	}
	ctx := appreq.NewContext(context.Background(), req)

	called := false
	env := &mockEnv{
		latestNoteViews: appmodel.NewNoteViews,
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			called = true
			return appmodel.NoteSaveResult{PathID: 1, VersionID: 0}, nil
		},
		prepareLatestNotes:         noopPrepare,
		handleLatestNotesAfterSave: noopHandle,
	}

	out, err := updatenotes.Resolve(ctx, env, model.UpdateNotesInput{
		ApiKey: db.ApiKey{},
		Changes: []model.NoteChangeInput{
			{Upsert: &model.NoteChangeUpsertInput{Path: "notes/todo.md", Content: "x"}},
		},
	})
	require.NoError(t, err)
	require.IsType(t, model.UpdateNotesSuccessPayload{}, out)
	require.True(t, called)
}

// F2: unscoped/admin request (no DeliveryKind) + empty write_patterns → allowed (unchanged behavior).
func TestResolve_Unscoped_EmptyWritePatterns_Allowed(t *testing.T) {
	// No appreq in context — simulates admin/unscoped request.
	ctx := context.Background()

	called := false
	env := &mockEnv{
		latestNoteViews: appmodel.NewNoteViews,
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			called = true
			return appmodel.NoteSaveResult{PathID: 1, VersionID: 0}, nil
		},
		prepareLatestNotes:         noopPrepare,
		handleLatestNotesAfterSave: noopHandle,
	}

	out, err := updatenotes.Resolve(ctx, env, model.UpdateNotesInput{
		ApiKey: db.ApiKey{},
		Changes: []model.NoteChangeInput{
			{Upsert: &model.NoteChangeUpsertInput{Path: "anything.md", Content: "x"}},
		},
	})
	require.NoError(t, err)
	require.IsType(t, model.UpdateNotesSuccessPayload{}, out)
	require.True(t, called)
}

// TestResolve_UnhideReloadsWithoutEvents pins the other half of the no-op rule: a
// write that stored nothing but brought a hidden path back must reload the served
// snapshot — the note is absent from it while hidden — yet still raise no change
// event, since its content did not change.
func TestResolve_UnhideReloadsWithoutEvents(t *testing.T) {
	ctx := context.Background()

	const pathID int64 = 9

	views := appmodel.NewNoteViews()
	views.RegisterNote(&appmodel.NoteView{
		PathID: pathID, Path: "note.md", Permalink: "note.md", VersionID: 42,
	})

	reloads := 0
	handles := 0
	env := &mockEnv{
		latestNoteViews: appmodel.NewNoteViews,
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			return appmodel.NoteSaveResult{PathID: pathID, Unhidden: true}, nil
		},
		prepareLatestNotes: func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
			reloads++
			return views, nil
		},
		handleLatestNotesAfterSave: func(_ context.Context, _ []int64) error {
			handles++
			return nil
		},
	}

	result, err := updatenotes.Resolve(ctx, env, model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Upsert: &model.NoteChangeUpsertInput{Path: "note.md", Content: "same content"}},
		},
	})
	require.NoError(t, err)

	payload, ok := result.(model.UpdateNotesSuccessPayload)
	require.True(t, ok, "expected UpdateNotesSuccessPayload, got %T", result)
	require.Equal(t, 1, reloads, "an unhidden note must reach the served snapshot")
	require.Zero(t, handles, "unhiding is not a content change")
	require.Len(t, payload.Updated, 1)
}

// TestResolve_MixedBatchHandlesOnlyChangedPaths pins the filter: in a batch where
// one write changed content and one did not, only the changed path reaches the
// post-save handler, while both are reported to the caller.
func TestResolve_MixedBatchHandlesOnlyChangedPaths(t *testing.T) {
	ctx := context.Background()

	const changedPathID int64 = 11
	const unchangedPathID int64 = 12

	views := appmodel.NewNoteViews()
	views.RegisterNote(&appmodel.NoteView{
		PathID: changedPathID, Path: "changed.md", Permalink: "changed.md", VersionID: 21,
	})
	views.RegisterNote(&appmodel.NoteView{
		PathID: unchangedPathID, Path: "unchanged.md", Permalink: "unchanged.md", VersionID: 22,
	})

	var handled []int64
	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return views },
		insertNote: func(_ context.Context, note appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			if note.Path == "changed.md" {
				return appmodel.NoteSaveResult{PathID: changedPathID, VersionID: 21}, nil
			}
			return appmodel.NoteSaveResult{PathID: unchangedPathID}, nil
		},
		prepareLatestNotes: func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
			return views, nil
		},
		handleLatestNotesAfterSave: func(_ context.Context, pathIDs []int64) error {
			handled = pathIDs
			return nil
		},
	}

	result, err := updatenotes.Resolve(ctx, env, model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Upsert: &model.NoteChangeUpsertInput{Path: "changed.md", Content: "new"}},
			{Upsert: &model.NoteChangeUpsertInput{Path: "unchanged.md", Content: "same"}},
		},
	})
	require.NoError(t, err)

	payload, ok := result.(model.UpdateNotesSuccessPayload)
	require.True(t, ok, "expected UpdateNotesSuccessPayload, got %T", result)
	require.Len(t, payload.Updated, 2, "both paths are reported to the caller")
	require.Equal(t, []int64{changedPathID}, handled, "only the changed path raises events")
}

// TestResolve_HideWithUnchangedUpsertStillReports pins a batch that mixes a hide
// with a write that stored nothing: the hide drives the reload, and the no-op path
// must still come back in the response.
func TestResolve_HideWithUnchangedUpsertStillReports(t *testing.T) {
	ctx := context.Background()

	const pathID int64 = 13

	views := appmodel.NewNoteViews()
	views.RegisterNote(&appmodel.NoteView{
		PathID: pathID, Path: "kept.md", Permalink: "kept.md", VersionID: 31,
	})

	handles := 0
	env := &mockEnv{
		latestNoteViews: func() *appmodel.NoteViews { return views },
		insertNote: func(_ context.Context, _ appmodel.RawNote) (appmodel.NoteSaveResult, error) {
			return appmodel.NoteSaveResult{PathID: pathID}, nil
		},
		hideNotePath: func(_ context.Context, _ db.HideNotePathParams) error { return nil },
		prepareLatestNotes: func(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
			return views, nil
		},
		handleLatestNotesAfterSave: func(_ context.Context, _ []int64) error {
			handles++
			return nil
		},
	}

	result, err := updatenotes.Resolve(ctx, env, model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Hide: &model.NoteChangeHideInput{Path: "gone.md"}},
			{Upsert: &model.NoteChangeUpsertInput{Path: "kept.md", Content: "same"}},
		},
	})
	require.NoError(t, err)

	payload, ok := result.(model.UpdateNotesSuccessPayload)
	require.True(t, ok, "expected UpdateNotesSuccessPayload, got %T", result)
	require.Len(t, payload.Updated, 1, "the unchanged path must not vanish from the response")
	require.Zero(t, handles, "nothing was written, so no change events")
}
