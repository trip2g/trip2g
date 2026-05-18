# updateNotes Mutation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `updateNotes` GraphQL mutation that atomically applies a batch of create/update (upsert), find-replace (patch), and hide operations without a separate `commitNotes` call.

**Architecture:** New use case `internal/case/updatenotes/` follows the standard Env-interface pattern. Schema types added to `schema.graphqls`, code generated with `make gqlgen`, resolver wired in `schema.resolvers.go`. Patch operation reads current content from in-memory `NoteViews`, performs string find/replace, then calls `InsertNote`. Upsert delegates directly to `InsertNote`. Hide calls `HideNotePath`. All three operations in one batch share a single `HandleLatestNotesAfterSave` call at the end.

**Tech Stack:** Go, gqlgen, SQLite, sha256 (crypto/sha256 + encoding/base64), moq mocks.

---

## File Map

| Action | File |
|--------|------|
| Modify | `internal/graph/schema.graphqls` |
| Generated (do not edit) | `internal/graph/generated.go`, `internal/graph/model/models_gen.go` |
| Create | `internal/case/updatenotes/resolve.go` |
| Create | `internal/case/updatenotes/resolve_test.go` |
| Modify | `internal/graph/schema.resolvers.go` |

---

## Task 1: Add GraphQL types and mutation to schema

**Files:**
- Modify: `internal/graph/schema.graphqls`

Find the `# hideNotes` section (around line 1633) and add the new block **before** it. Also add the mutation to the `Mutation` type near `pushNotes` and `hideNotes`.

- [ ] **Step 1: Add input and payload types**

Add after the `# hideNotes` block (find `union HideNotesOrErrorPayload` and insert after it):

```graphql
#
# updateNotes
#

input NoteChangeUpsertInput {
  path: String!
  content: String!
  expectedHash: String
}

input NoteChangePatchInput {
  path: String!
  find: String!
  replace: String!
  expectedHash: String
}

input NoteChangeHideInput {
  path: String!
}

input NoteChangeInput {
  upsert: NoteChangeUpsertInput
  patch: NoteChangePatchInput
  hide: NoteChangeHideInput
}

input UpdateNotesInput @goExtraField(name: "ApiKey", type: "trip2g/internal/db.ApiKey") {
  changes: [NoteChangeInput!]!
}

type UpdateNotesSuccessPayload {
  paths: [String!]!
}

type UpdateNotesHashMismatchPayload {
  path: String!
  actualHash: String!
}

type UpdateNotesPatchNotFoundPayload {
  path: String!
  find: String!
}

union UpdateNotesOrErrorPayload =
    UpdateNotesSuccessPayload
  | UpdateNotesHashMismatchPayload
  | UpdateNotesPatchNotFoundPayload
  | ErrorPayload
```

- [ ] **Step 2: Add mutation to Mutation type**

Find `pushNotes(input: PushNotesInput!): PushNotesOrErrorPayload!` in the `type Mutation` block and add directly after it:

```graphql
  updateNotes(input: UpdateNotesInput!): UpdateNotesOrErrorPayload!
```

- [ ] **Step 3: Run gqlgen**

```bash
make gqlgen
```

Expected: exits 0, regenerates `internal/graph/generated.go` and `internal/graph/model/models_gen.go`. New types `UpdateNotesInput`, `NoteChangeInput`, `UpdateNotesSuccessPayload`, `UpdateNotesHashMismatchPayload`, `UpdateNotesPatchNotFoundPayload` appear in `models_gen.go`. Interface `MutationResolver` in `generated.go` now includes `UpdateNotes(ctx context.Context, input model.UpdateNotesInput) (model.UpdateNotesOrErrorPayload, error)`.

- [ ] **Step 4: Verify build fails with missing resolver**

```bash
go build ./internal/graph/...
```

Expected: compile error — `*mutationResolver` does not implement `MutationResolver` (missing `UpdateNotes` method). This confirms gqlgen wired the mutation.

- [ ] **Step 5: Commit**

```bash
git add internal/graph/schema.graphqls internal/graph/generated.go internal/graph/model/models_gen.go
git commit -m "feat(graphql): add updateNotes mutation schema types"
```

---

## Task 2: Write failing tests for the use case

**Files:**
- Create: `internal/case/updatenotes/resolve_test.go`

The test uses a hand-written mock (moq will be added later; write it manually here so the test compiles before implementation exists).

- [ ] **Step 1: Create test file**

```go
package updatenotes_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"trip2g/internal/case/updatenotes"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

// --- minimal mock ---

type mockEnv struct {
	notes          *appmodel.NoteViews
	insertedNotes  []appmodel.RawNote
	hiddenPaths    []string
	insertErr      error
	hideErr        error
	afterSaveErr   error
	prepareErr     error
}

func (m *mockEnv) LatestNoteViews() *appmodel.NoteViews { return m.notes }

func (m *mockEnv) InsertNote(_ context.Context, note appmodel.RawNote) (int64, error) {
	if m.insertErr != nil {
		return 0, m.insertErr
	}
	m.insertedNotes = append(m.insertedNotes, note)
	return int64(len(m.insertedNotes)), nil
}

func (m *mockEnv) HideNotePath(_ context.Context, params db.HideNotePathParams) error {
	if m.hideErr != nil {
		return m.hideErr
	}
	m.hiddenPaths = append(m.hiddenPaths, params.Value)
	return nil
}

func (m *mockEnv) HandleLatestNotesAfterSave(_ context.Context, _ []int64) error {
	return m.afterSaveErr
}

func (m *mockEnv) PrepareLatestNotes(_ context.Context, _ bool) (*appmodel.NoteViews, error) {
	if m.prepareErr != nil {
		return nil, m.prepareErr
	}
	return m.notes, nil
}

// --- helpers ---

func noteViews(path, content string) *appmodel.NoteViews {
	nv := &appmodel.NoteView{
		Path:    path,
		Content: []byte(content),
		PathID:  1,
	}
	nvs := &appmodel.NoteViews{}
	nvs.List = []*appmodel.NoteView{nv}
	return nvs
}

func ptr(s string) *string { return &s }

// --- tests ---

func TestResolve_Upsert(t *testing.T) {
	env := &mockEnv{notes: noteViews("inbox.md", "old content")}
	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Upsert: &model.NoteChangeUpsertInput{Path: "inbox.md", Content: "new content"}},
		},
	}
	result, err := updatenotes.Resolve(context.Background(), env, input)
	require.NoError(t, err)
	payload, ok := result.(*model.UpdateNotesSuccessPayload)
	require.True(t, ok, "expected UpdateNotesSuccessPayload, got %T", result)
	require.Equal(t, []string{"inbox.md"}, payload.Paths)
	require.Len(t, env.insertedNotes, 1)
	require.Equal(t, "new content", env.insertedNotes[0].Content)
}

func TestResolve_Patch_Found(t *testing.T) {
	env := &mockEnv{notes: noteViews("todo.md", "- [ ] buy milk\n- [x] sleep")}
	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Patch: &model.NoteChangePatchInput{
				Path:    "todo.md",
				Find:    "- [ ] buy milk",
				Replace: "- [x] buy milk",
			}},
		},
	}
	result, err := updatenotes.Resolve(context.Background(), env, input)
	require.NoError(t, err)
	payload, ok := result.(*model.UpdateNotesSuccessPayload)
	require.True(t, ok, "expected UpdateNotesSuccessPayload, got %T", result)
	require.Equal(t, []string{"todo.md"}, payload.Paths)
	require.Len(t, env.insertedNotes, 1)
	require.Equal(t, "- [x] buy milk\n- [x] sleep", env.insertedNotes[0].Content)
}

func TestResolve_Patch_NotFound(t *testing.T) {
	env := &mockEnv{notes: noteViews("todo.md", "- [ ] buy milk")}
	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Patch: &model.NoteChangePatchInput{
				Path:    "todo.md",
				Find:    "- [ ] does not exist",
				Replace: "- [x] does not exist",
			}},
		},
	}
	result, err := updatenotes.Resolve(context.Background(), env, input)
	require.NoError(t, err)
	payload, ok := result.(*model.UpdateNotesPatchNotFoundPayload)
	require.True(t, ok, "expected UpdateNotesPatchNotFoundPayload, got %T", result)
	require.Equal(t, "todo.md", payload.Path)
	require.Equal(t, "- [ ] does not exist", payload.Find)
}

func TestResolve_Patch_MultipleOccurrences(t *testing.T) {
	env := &mockEnv{notes: noteViews("todo.md", "- [ ] task\n- [ ] task")}
	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Patch: &model.NoteChangePatchInput{
				Path:    "todo.md",
				Find:    "- [ ] task",
				Replace: "- [x] task",
			}},
		},
	}
	result, err := updatenotes.Resolve(context.Background(), env, input)
	require.NoError(t, err)
	_, ok := result.(*model.UpdateNotesPatchNotFoundPayload)
	require.True(t, ok, "expected UpdateNotesPatchNotFoundPayload for ambiguous find, got %T", result)
}

func TestResolve_Patch_NoteNotFound(t *testing.T) {
	env := &mockEnv{notes: noteViews("other.md", "content")}
	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Patch: &model.NoteChangePatchInput{
				Path:    "missing.md",
				Find:    "x",
				Replace: "y",
			}},
		},
	}
	result, err := updatenotes.Resolve(context.Background(), env, input)
	require.NoError(t, err)
	_, ok := result.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload for missing note, got %T", result)
}

func TestResolve_HashMismatch_Upsert(t *testing.T) {
	env := &mockEnv{notes: noteViews("inbox.md", "actual content")}
	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Upsert: &model.NoteChangeUpsertInput{
				Path:         "inbox.md",
				Content:      "new content",
				ExpectedHash: ptr("wronghash"),
			}},
		},
	}
	result, err := updatenotes.Resolve(context.Background(), env, input)
	require.NoError(t, err)
	payload, ok := result.(*model.UpdateNotesHashMismatchPayload)
	require.True(t, ok, "expected UpdateNotesHashMismatchPayload, got %T", result)
	require.Equal(t, "inbox.md", payload.Path)
	require.NotEmpty(t, payload.ActualHash)
}

func TestResolve_HashMismatch_Patch(t *testing.T) {
	env := &mockEnv{notes: noteViews("todo.md", "- [ ] task")}
	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Patch: &model.NoteChangePatchInput{
				Path:         "todo.md",
				Find:         "- [ ] task",
				Replace:      "- [x] task",
				ExpectedHash: ptr("wronghash"),
			}},
		},
	}
	result, err := updatenotes.Resolve(context.Background(), env, input)
	require.NoError(t, err)
	_, ok := result.(*model.UpdateNotesHashMismatchPayload)
	require.True(t, ok, "expected UpdateNotesHashMismatchPayload, got %T", result)
}

func TestResolve_Hide(t *testing.T) {
	env := &mockEnv{notes: noteViews("old.md", "content")}
	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Hide: &model.NoteChangeHideInput{Path: "old.md"}},
		},
	}
	result, err := updatenotes.Resolve(context.Background(), env, input)
	require.NoError(t, err)
	payload, ok := result.(*model.UpdateNotesSuccessPayload)
	require.True(t, ok, "expected UpdateNotesSuccessPayload, got %T", result)
	require.Equal(t, []string{"old.md"}, payload.Paths)
	require.Equal(t, []string{"old.md"}, env.hiddenPaths)
}

func TestResolve_NoOperation(t *testing.T) {
	env := &mockEnv{notes: noteViews("x.md", "y")}
	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{}, // no upsert/patch/hide set
		},
	}
	result, err := updatenotes.Resolve(context.Background(), env, input)
	require.NoError(t, err)
	_, ok := result.(*model.ErrorPayload)
	require.True(t, ok, "expected ErrorPayload for empty change, got %T", result)
}

func TestResolve_MixedBatch(t *testing.T) {
	nvs := &appmodel.NoteViews{}
	nvs.List = []*appmodel.NoteView{
		{Path: "todo.md", Content: []byte("- [ ] task"), PathID: 1},
		{Path: "inbox.md", Content: []byte("old"), PathID: 2},
		{Path: "gone.md", Content: []byte("bye"), PathID: 3},
	}
	env := &mockEnv{notes: nvs}
	input := model.UpdateNotesInput{
		Changes: []model.NoteChangeInput{
			{Patch: &model.NoteChangePatchInput{Path: "todo.md", Find: "- [ ] task", Replace: "- [x] task"}},
			{Upsert: &model.NoteChangeUpsertInput{Path: "inbox.md", Content: "new"}},
			{Hide: &model.NoteChangeHideInput{Path: "gone.md"}},
		},
	}
	result, err := updatenotes.Resolve(context.Background(), env, input)
	require.NoError(t, err)
	payload, ok := result.(*model.UpdateNotesSuccessPayload)
	require.True(t, ok, "expected UpdateNotesSuccessPayload, got %T", result)
	require.ElementsMatch(t, []string{"todo.md", "inbox.md", "gone.md"}, payload.Paths)
	require.Len(t, env.insertedNotes, 2)
	require.Equal(t, []string{"gone.md"}, env.hiddenPaths)
}
```

- [ ] **Step 2: Run test to confirm it fails to compile (resolve.go doesn't exist yet)**

```bash
go test ./internal/case/updatenotes/... 2>&1 | head -20
```

Expected: `cannot find package "trip2g/internal/case/updatenotes"` or similar compile error.

---

## Task 3: Implement the use case

**Files:**
- Create: `internal/case/updatenotes/resolve.go`

- [ ] **Step 1: Create resolve.go**

```go
package updatenotes

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

type Env interface {
	LatestNoteViews() *appmodel.NoteViews
	InsertNote(ctx context.Context, note appmodel.RawNote) (int64, error)
	HideNotePath(ctx context.Context, params db.HideNotePathParams) error
	PrepareLatestNotes(ctx context.Context, partial bool) (*appmodel.NoteViews, error)
	HandleLatestNotesAfterSave(ctx context.Context, pathIDs []int64) error
}

func Resolve(ctx context.Context, env Env, input model.UpdateNotesInput) (model.UpdateNotesOrErrorPayload, error) {
	if len(input.Changes) == 0 {
		return &model.UpdateNotesSuccessPayload{Paths: []string{}}, nil
	}

	var pathIDs []int64
	var paths []string

	for _, change := range input.Changes {
		opCount := 0
		if change.Upsert != nil {
			opCount++
		}
		if change.Patch != nil {
			opCount++
		}
		if change.Hide != nil {
			opCount++
		}
		if opCount != 1 {
			return &model.ErrorPayload{
				Message: "each change must have exactly one of: upsert, patch, hide",
			}, nil
		}

		switch {
		case change.Upsert != nil:
			payload, pathID, err := applyUpsert(ctx, env, change.Upsert)
			if err != nil {
				return nil, err
			}
			if payload != nil {
				return payload, nil
			}
			pathIDs = append(pathIDs, pathID)
			paths = append(paths, change.Upsert.Path)

		case change.Patch != nil:
			payload, pathID, err := applyPatch(ctx, env, change.Patch)
			if err != nil {
				return nil, err
			}
			if payload != nil {
				return payload, nil
			}
			pathIDs = append(pathIDs, pathID)
			paths = append(paths, change.Patch.Path)

		case change.Hide != nil:
			if err := applyHide(ctx, env, change.Hide, input.ApiKey.CreatedBy); err != nil {
				return nil, err
			}
			paths = append(paths, change.Hide.Path)
		}
	}

	if len(pathIDs) > 0 {
		if _, err := env.PrepareLatestNotes(ctx, false); err != nil {
			return nil, fmt.Errorf("prepare notes: %w", err)
		}
		if err := env.HandleLatestNotesAfterSave(ctx, pathIDs); err != nil {
			return nil, fmt.Errorf("handle after save: %w", err)
		}
	}

	return &model.UpdateNotesSuccessPayload{Paths: paths}, nil
}

func contentHash(content []byte) string {
	h := sha256.New()
	h.Write(content)
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

func applyUpsert(ctx context.Context, env Env, u *model.NoteChangeUpsertInput) (model.UpdateNotesOrErrorPayload, int64, error) {
	if u.ExpectedHash != nil {
		nv := env.LatestNoteViews().GetByPath(u.Path)
		if nv != nil {
			actual := contentHash(nv.Content)
			if actual != *u.ExpectedHash {
				return &model.UpdateNotesHashMismatchPayload{Path: u.Path, ActualHash: actual}, 0, nil
			}
		}
	}

	pathID, err := env.InsertNote(ctx, appmodel.RawNote{Path: u.Path, Content: u.Content})
	if err != nil {
		return nil, 0, fmt.Errorf("insert note %s: %w", u.Path, err)
	}
	return nil, pathID, nil
}

func applyPatch(ctx context.Context, env Env, p *model.NoteChangePatchInput) (model.UpdateNotesOrErrorPayload, int64, error) {
	nv := env.LatestNoteViews().GetByPath(p.Path)
	if nv == nil {
		return &model.ErrorPayload{Message: fmt.Sprintf("note not found: %s", p.Path)}, 0, nil
	}

	if p.ExpectedHash != nil {
		actual := contentHash(nv.Content)
		if actual != *p.ExpectedHash {
			return &model.UpdateNotesHashMismatchPayload{Path: p.Path, ActualHash: actual}, 0, nil
		}
	}

	current := string(nv.Content)
	idx := strings.Index(current, p.Find)
	if idx == -1 {
		return &model.UpdateNotesPatchNotFoundPayload{Path: p.Path, Find: p.Find}, 0, nil
	}
	if strings.Index(current[idx+len(p.Find):], p.Find) != -1 {
		return &model.UpdateNotesPatchNotFoundPayload{Path: p.Path, Find: p.Find}, 0, nil
	}

	newContent := current[:idx] + p.Replace + current[idx+len(p.Find):]
	pathID, err := env.InsertNote(ctx, appmodel.RawNote{Path: p.Path, Content: newContent})
	if err != nil {
		return nil, 0, fmt.Errorf("insert note %s: %w", p.Path, err)
	}
	return nil, pathID, nil
}

func applyHide(ctx context.Context, env Env, h *model.NoteChangeHideInput, createdBy int64) error {
	return env.HideNotePath(ctx, db.HideNotePathParams{
		HiddenBy: &createdBy,
		Value:    h.Path,
	})
}
```

- [ ] **Step 2: Run tests**

```bash
go test ./internal/case/updatenotes/... -v
```

Expected: all tests PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/case/updatenotes/
git commit -m "feat(updatenotes): implement updateNotes use case with upsert/patch/hide"
```

---

## Task 4: Wire the resolver

**Files:**
- Modify: `internal/graph/schema.resolvers.go`

- [ ] **Step 1: Add import and resolver method**

Add import at the top with other case imports:
```go
"trip2g/internal/case/updatenotes"
```

Find `func (r *mutationResolver) HideNotes(...)` and add **before** it:

```go
// UpdateNotes is the resolver for the updateNotes field.
func (r *mutationResolver) UpdateNotes(ctx context.Context, input model.UpdateNotesInput) (model.UpdateNotesOrErrorPayload, error) {
	apiKey, err := checkapikey.Resolve(ctx, r.env(ctx), "update_notes")
	if err != nil {
		return nil, err
	}

	input.ApiKey = *apiKey

	return updatenotes.Resolve(ctx, r.env(ctx), input)
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./internal/graph/...
```

Expected: exits 0, no errors.

- [ ] **Step 3: Run all use case tests**

```bash
go test ./internal/case/updatenotes/... -v
```

Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/graph/schema.resolvers.go
git commit -m "feat(graphql): wire updateNotes resolver"
```

---

## Task 5: Verify end-to-end with the dev server

- [ ] **Step 1: Start dev server**

```bash
make air
```

- [ ] **Step 2: Test patch operation via GraphQL**

Send to `http://localhost:8080/graphql` (with a valid API key header):

```graphql
mutation {
  updateNotes(input: {
    changes: [
      {
        patch: {
          path: "some-existing-note.md"
          find: "- [ ] some task"
          replace: "- [x] some task"
        }
      }
    ]
  }) {
    ... on UpdateNotesSuccessPayload { paths }
    ... on UpdateNotesHashMismatchPayload { path actualHash }
    ... on UpdateNotesPatchNotFoundPayload { path find }
    ... on ErrorPayload { message }
  }
}
```

Expected: `UpdateNotesSuccessPayload` with the note path, and the markdown file updated.

- [ ] **Step 3: Test error cases**

Send a patch with a `find` string that appears twice — expect `UpdateNotesPatchNotFoundPayload`.
Send a change with no upsert/patch/hide — expect `ErrorPayload`.

- [ ] **Step 4: Stop dev server, commit if any fixes were needed**
