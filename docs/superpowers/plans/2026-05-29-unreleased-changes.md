# Unreleased Changes API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `unreleasedChanges` GraphQL query (API-key gated) that lists notes diverging from the live release with line/word stats and lazy diff fields, so an AI agent can monitor accumulated edits and decide when to publish.

**Architecture:** Use the existing `AllLiveNotes` DB query (live release side) and `LatestNoteViews()` in-memory cache (latest edited side). Compare them by `pathID` to classify each note as `added/modified/removed`. Return a Connection with aggregated stats; compute `unifiedDiff` and `wordDiff` lazily in `forceResolver` fields using `pmezard/go-difflib` (already in `go.mod`).

**Tech Stack:** Go, gqlgen, `pmezard/go-difflib` (line + word diff, already indirect dep), `webhookutil.MatchesAny` (glob filter), `checkapikey.Resolve` (auth).

---

## File Map

| Action | Path | Responsibility |
|--------|------|---------------|
| Modify | `internal/graph/schema.graphqls` | Add 5 new types + `unreleasedChanges` to Query |
| Modify | `gqlgen.yml` | Map `UnreleasedChange` to custom model so gqlgen skips generating it |
| Create | `internal/graph/model/unreleased_change.go` | Custom model: `sync.Once` + `Diff()` + `computeDiff()` — not overwritten by gqlgen |
| Create | `internal/graph/model/unreleased_change_test.go` | Tests for `Diff()`: correctness, sync.Once called once, nil content handled |
| Run codegen | (make gqlgen) | Generates resolver stubs in `internal/graph/generated.go` and remaining model types |
| Create | `internal/case/listunreleasedchanges/resolve.go` | Use case: compute divergence, returns `[]*model.UnreleasedChange` |
| Create | `internal/case/listunreleasedchanges/resolve_test.go` | Table-driven tests |
| Create | `internal/case/listunreleasedchanges/mocks_test.go` | moq-generated Env mock |
| Create | `internal/graph/unreleased_changes.go` | Query resolver + Stats/UnifiedDiff/WordDiff/TotalStats field resolvers |

---

## Task 1: GraphQL Schema + Codegen

**Files:**
- Modify: `internal/graph/schema.graphqls` (two locations: near line 1888 for types, line 1502 area for query field)

- [ ] **Step 1: Add enum and types to schema**

In `internal/graph/schema.graphqls`, after the `# noteChanges subscription` comment (near line 1888), add before the `input NoteChangesFilter` block:

```graphql
enum NoteChangeType {
  ADDED     # exists in latest, not in live release (or no live release)
  MODIFIED  # in both; version IDs differ
  REMOVED   # was in live release, now absent from latest
}

type UnreleasedChangeStats {
  addedLines:   Int!
  removedLines: Int!
  changedWords: Int!
}

type UnreleasedChange {
  path:            String!
  pathId:          Int64!
  title:           String!
  changeType:      NoteChangeType!
  liveVersionId:   Int64
  latestVersionId: Int64
  stats:           UnreleasedChangeStats! @goField(forceResolver: true)
  """Git-style unified diff on raw markdown (released → latest). Computed lazily."""
  unifiedDiff:     String! @goField(forceResolver: true)
  """Inline word-level diff: {+added+} / {-removed-} markers. Computed lazily."""
  wordDiff:        String! @goField(forceResolver: true)
  oldContent:      String    # null when ADDED
  newContent:      String    # null when REMOVED
}

type UnreleasedChangesConnection {
  totalCount:       Int!
  totalStats:       UnreleasedChangeStats! @goField(forceResolver: true)
  nodes:            [UnreleasedChange!]!
}

```

- [ ] **Step 2: Add query field to Query type**

In the `type Query` block (around line 1502), add after `notePaths`:

```graphql
  """
  X-Api-Key header required.
  Notes whose latest version diverges from the current live release.
  If no live release exists, all notes are returned as added.
  Glob filter uses doublestar (same as noteChanges).
  """
  unreleasedChanges(filter: NoteChangesFilter!): UnreleasedChangesConnection!
```

- [ ] **Step 3: Configure gqlgen to use custom model**

In `gqlgen.yml`, under the `models:` section add:

```yaml
models:
  UnreleasedChange:
    model: trip2g/internal/graph/model.UnreleasedChange
```

This prevents gqlgen from generating the struct in `models_gen.go` — we define it manually so we can embed `sync.Once`.

- [ ] **Step 4: Create the custom model with lazy diff computation**

Create `internal/graph/model/unreleased_change.go` (gqlgen only writes to `models_gen.go`, this file is safe):

> **Why here?** `NoteChangeType` is in the same package — no import needed, no circular deps.
> `internal/model` would require importing `internal/graph/model` for the enum → circular.
> `sync.Once` causes no memory leak: objects are per-request and GC'd after the response.

```go
package model

import (
	"strings"
	"sync"

	"github.com/pmezard/go-difflib/difflib"
)

// DiffResult is computed once per UnreleasedChange via sync.Once.
// Shared by the Stats, UnifiedDiff, and WordDiff resolver fields.
type DiffResult struct {
	Unified      string
	Word         string
	AddedLines   int
	RemovedLines int
	ChangedWords int
}

// UnreleasedChange is defined manually (not generated) to embed sync.Once for lazy
// single-computation diff. All three lazy fields call Diff() — it runs at most once.
type UnreleasedChange struct {
	Path            string
	PathID          int64
	Title           string
	ChangeType      NoteChangeType
	LiveVersionID   *int64
	LatestVersionID *int64
	OldContent      *string // nil when ADDED
	NewContent      *string // nil when REMOVED

	once sync.Once
	diff *DiffResult
}

// Diff computes and caches the diff result. Safe for concurrent resolver calls.
func (u *UnreleasedChange) Diff() *DiffResult {
	u.once.Do(func() {
		u.diff = computeDiff(derefStr(u.OldContent), derefStr(u.NewContent))
	})
	return u.diff
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func computeDiff(old, nw string) *DiffResult {
	unified, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(old),
		B:        difflib.SplitLines(nw),
		FromFile: "released",
		ToFile:   "latest",
		Context:  3,
	})

	oldLines := difflib.SplitLines(old)
	newLines := difflib.SplitLines(nw)
	lm := difflib.NewMatcher(oldLines, newLines)
	var addedLines, removedLines int
	for _, op := range lm.GetOpCodes() {
		switch op.Tag {
		case 'i':
			addedLines += op.J2 - op.J1
		case 'd':
			removedLines += op.I2 - op.I1
		case 'r':
			addedLines += op.J2 - op.J1
			removedLines += op.I2 - op.I1
		}
	}

	oldWords := strings.Fields(old)
	newWords := strings.Fields(nw)
	wm := difflib.NewMatcher(oldWords, newWords)
	var changedWords int
	var sb strings.Builder
	for _, op := range wm.GetOpCodes() {
		switch op.Tag {
		case 'e':
			for _, w := range oldWords[op.I1:op.I2] {
				sb.WriteString(w)
				sb.WriteByte(' ')
			}
		case 'r':
			for _, w := range oldWords[op.I1:op.I2] {
				sb.WriteString("{-")
				sb.WriteString(w)
				sb.WriteString("-} ")
			}
			for _, w := range newWords[op.J1:op.J2] {
				sb.WriteString("{+")
				sb.WriteString(w)
				sb.WriteString("+} ")
			}
			changedWords += op.J2 - op.J1 + op.I2 - op.I1
		case 'i':
			for _, w := range newWords[op.J1:op.J2] {
				sb.WriteString("{+")
				sb.WriteString(w)
				sb.WriteString("+} ")
			}
			changedWords += op.J2 - op.J1
		case 'd':
			for _, w := range oldWords[op.I1:op.I2] {
				sb.WriteString("{-")
				sb.WriteString(w)
				sb.WriteString("-} ")
			}
			changedWords += op.I2 - op.I1
		}
	}

	return &DiffResult{
		Unified:      unified,
		Word:         strings.TrimSpace(sb.String()),
		AddedLines:   addedLines,
		RemovedLines: removedLines,
		ChangedWords: changedWords,
	}
}
```

- [ ] **Step 5: Run codegen**

```bash
make gqlgen
```

Expected: regenerates `internal/graph/generated.go` and `internal/graph/model/models_gen.go`. `UnreleasedChange` should NOT appear in `models_gen.go` (gqlgen uses the custom type). May print stub resolver warnings — note them, they will be fixed in Task 3.

- [ ] **Step 6: Verify build**

```bash
go build ./...
```

Expected: compile errors about unimplemented resolver methods — note them, they will be fixed in Task 3.

- [ ] **Step 7: Write and run model tests**

Create `internal/graph/model/unreleased_change_test.go` covering:
- `Diff()` returns correct line/word stats for a simple modification
- `Diff()` handles nil `OldContent` (ADDED) and nil `NewContent` (REMOVED) without panic
- `Diff()` is called only once even when invoked multiple times (sync.Once)

```bash
go test ./internal/graph/model/... -v -run TestUnreleasedChange
```

Expected: all tests PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/graph/schema.graphqls gqlgen.yml internal/graph/model/unreleased_change.go internal/graph/model/unreleased_change_test.go internal/graph/generated.go internal/graph/model/models_gen.go
git commit -m "feat(graphql): add unreleasedChanges schema types and custom model with lazy diff"
```

---

## Task 2: Use Case (TDD)

**Files:**
- Create: `internal/case/listunreleasedchanges/resolve.go`
- Create: `internal/case/listunreleasedchanges/resolve_test.go`
- Create: `internal/case/listunreleasedchanges/mocks_test.go`

The use case classifies diverging notes and computes stats. It does NOT compute diff strings (those are lazy resolver fields).

- [ ] **Step 1: Write the failing test**

Create `internal/case/listunreleasedchanges/resolve_test.go`:

```go
package listunreleasedchanges_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"trip2g/internal/case/listunreleasedchanges"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

func makeNoteViews(views ...*appmodel.NoteView) *appmodel.NoteViews {
	nv := &appmodel.NoteViews{}
	for _, v := range views {
		nv.List = append(nv.List, v)
	}
	return nv
}

func liveNote(pathID, versionID int64, path, content string) db.AllLiveNotesRow {
	return db.AllLiveNotesRow{
		PathID:    pathID,
		VersionID: versionID,
		Path:      path,
		Content:   content,
	}
}

func latestNote(pathID, versionID int64, path, content string) *appmodel.NoteView {
	return &appmodel.NoteView{
		PathID:    pathID,
		VersionID: versionID,
		Path:      path,
		Content:   []byte(content),
		Title:     path,
	}
}

func TestResolve_NoLiveRelease_AllAdded(t *testing.T) {
	env := &EnvMock{
		AllLiveNotesFunc: func(ctx context.Context) ([]db.AllLiveNotesRow, error) {
			return nil, nil // no live release
		},
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return makeNoteViews(latestNote(1, 10, "posts/a.md", "hello world"))
		},
	}

	changes, err := listunreleasedchanges.Resolve(context.Background(), env, model.NoteChangesFilter{
		IncludePatterns: []string{"**"},
	})
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.Equal(t, model.NoteChangeTypeAdded, changes[0].ChangeType)
	require.Equal(t, int64(10), *changes[0].LatestVersionID)
	require.Nil(t, changes[0].LiveVersionID)
	require.Nil(t, changes[0].OldContent)                      // nil for ADDED
	require.Equal(t, "hello world", *changes[0].NewContent)
}

func TestResolve_Identical_EmptyResult(t *testing.T) {
	env := &EnvMock{
		AllLiveNotesFunc: func(ctx context.Context) ([]db.AllLiveNotesRow, error) {
			return []db.AllLiveNotesRow{liveNote(1, 10, "posts/a.md", "hello")}, nil
		},
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return makeNoteViews(latestNote(1, 10, "posts/a.md", "hello"))
		},
	}

	changes, err := listunreleasedchanges.Resolve(context.Background(), env, model.NoteChangesFilter{
		IncludePatterns: []string{"**"},
	})
	require.NoError(t, err)
	require.Empty(t, changes)
}

func TestResolve_Modified(t *testing.T) {
	env := &EnvMock{
		AllLiveNotesFunc: func(ctx context.Context) ([]db.AllLiveNotesRow, error) {
			return []db.AllLiveNotesRow{liveNote(1, 10, "posts/a.md", "old line\n")}, nil
		},
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return makeNoteViews(latestNote(1, 11, "posts/a.md", "new line\n"))
		},
	}

	changes, err := listunreleasedchanges.Resolve(context.Background(), env, model.NoteChangesFilter{
		IncludePatterns: []string{"**"},
	})
	require.NoError(t, err)
	require.Len(t, changes, 1)
	ch := changes[0]
	require.Equal(t, model.NoteChangeTypeModified, ch.ChangeType)
	require.Equal(t, int64(10), *ch.LiveVersionID)
	require.Equal(t, int64(11), *ch.LatestVersionID)
	require.Equal(t, "old line\n", *ch.OldContent)
	require.Equal(t, "new line\n", *ch.NewContent)
}

func TestResolve_Removed(t *testing.T) {
	env := &EnvMock{
		AllLiveNotesFunc: func(ctx context.Context) ([]db.AllLiveNotesRow, error) {
			return []db.AllLiveNotesRow{liveNote(1, 10, "posts/a.md", "old content")}, nil
		},
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return makeNoteViews() // empty: note was removed
		},
	}

	changes, err := listunreleasedchanges.Resolve(context.Background(), env, model.NoteChangesFilter{
		IncludePatterns: []string{"**"},
	})
	require.NoError(t, err)
	require.Len(t, changes, 1)
	ch := changes[0]
	require.Equal(t, model.NoteChangeTypeRemoved, ch.ChangeType)
	require.Equal(t, int64(10), *ch.LiveVersionID)
	require.Nil(t, ch.LatestVersionID)
	require.Equal(t, "old content", *ch.OldContent)
	require.Nil(t, ch.NewContent)                              // nil for REMOVED
}

func TestResolve_GlobFilter(t *testing.T) {
	env := &EnvMock{
		AllLiveNotesFunc: func(ctx context.Context) ([]db.AllLiveNotesRow, error) {
			return nil, nil
		},
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return makeNoteViews(
				latestNote(1, 10, "posts/a.md", "A"),
				latestNote(2, 20, "drafts/b.md", "B"),
			)
		},
	}

	changes, err := listunreleasedchanges.Resolve(context.Background(), env, model.NoteChangesFilter{
		IncludePatterns: []string{"posts/**"},
	})
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.Equal(t, "posts/a.md", changes[0].Path)
}

```

- [ ] **Step 2: Generate the Env mock**

Add the generate directive to the top of `resolve_test.go` (before `package`):

```go
//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env
```

Then create a minimal `resolve.go` skeleton so moq can find the `Env` interface, then generate:

```bash
# Create skeleton first (see Step 3), then:
cd internal/case/listunreleasedchanges && go generate ./...
```

Expected: creates `mocks_test.go` with `EnvMock` struct.

- [ ] **Step 3: Run the tests to confirm they fail**

```bash
go test ./internal/case/listunreleasedchanges/... -v
```

Expected: compile errors (Resolve not implemented yet). That's correct — proceed.

- [ ] **Step 4: Implement resolve.go**

Create `internal/case/listunreleasedchanges/resolve.go`:

```go
package listunreleasedchanges

import (
	"context"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/webhookutil"
)

type Env interface {
	AllLiveNotes(ctx context.Context) ([]db.AllLiveNotesRow, error)
	LatestNoteViews() *appmodel.NoteViews
}

// Resolve returns notes diverging between the live release and latest edited versions.
// model.UnreleasedChange is used directly — no local conversion needed in the resolver.
func Resolve(ctx context.Context, env Env, filter model.NoteChangesFilter) ([]*model.UnreleasedChange, error) {
	liveRows, err := env.AllLiveNotes(ctx)
	if err != nil {
		return nil, err
	}

	liveByPathID := make(map[int64]db.AllLiveNotesRow, len(liveRows))
	for _, row := range liveRows {
		if matchesFilter(row.Path, filter) {
			liveByPathID[row.PathID] = row
		}
	}

	latestViews := env.LatestNoteViews()
	seen := make(map[int64]bool)
	var changes []*model.UnreleasedChange

	for _, nv := range latestViews.List {
		if !matchesFilter(nv.Path, filter) {
			continue
		}
		newContent := string(nv.Content)
		lv, inLive := liveByPathID[nv.PathID]
		seen[nv.PathID] = true

		if !inLive {
			vid := nv.VersionID
			changes = append(changes, &model.UnreleasedChange{
				Path:            nv.Path,
				PathID:          nv.PathID,
				Title:           nv.Title,
				ChangeType:      model.NoteChangeTypeAdded,
				LatestVersionID: &vid,
				NewContent:      &newContent,
			})
		} else if lv.VersionID != nv.VersionID {
			lvid := lv.VersionID
			nvid := nv.VersionID
			oldContent := lv.Content
			changes = append(changes, &model.UnreleasedChange{
				Path:            nv.Path,
				PathID:          nv.PathID,
				Title:           nv.Title,
				ChangeType:      model.NoteChangeTypeModified,
				LiveVersionID:   &lvid,
				LatestVersionID: &nvid,
				OldContent:      &oldContent,
				NewContent:      &newContent,
			})
		}
	}

	for pathID, lv := range liveByPathID {
		if seen[pathID] {
			continue
		}
		lvid := lv.VersionID
		oldContent := lv.Content
		changes = append(changes, &model.UnreleasedChange{
			Path:          lv.Path,
			PathID:        pathID,
			ChangeType:    model.NoteChangeTypeRemoved,
			LiveVersionID: &lvid,
			OldContent:    &oldContent,
		})
	}

	return changes, nil
}

func matchesFilter(path string, filter model.NoteChangesFilter) bool {
	if !webhookutil.MatchesAny(path, filter.IncludePatterns) {
		return false
	}
	if len(filter.ExcludePatterns) > 0 && webhookutil.MatchesAny(path, filter.ExcludePatterns) {
		return false
	}
	return true
}
```

- [ ] **Step 5: Generate the moq mock**

```bash
cd internal/case/listunreleasedchanges
go generate ./...
cd ../../..
```

Expected: `mocks_test.go` created with `EnvMock`.

- [ ] **Step 6: Run tests and confirm they pass**

```bash
go test ./internal/case/listunreleasedchanges/... -v
```

Expected: all 6 tests PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/case/listunreleasedchanges/
git commit -m "feat(listunreleasedchanges): use case for live-vs-latest divergence"
```

---

## Task 3: Query Resolver + Diff Field Resolvers

**Files:**
- Create: `internal/graph/unreleased_changes.go`
- Modify: `internal/graph/schema.resolvers.go` (or let gqlgen stub handle it — see step 1)

- [ ] **Step 1: Implement UnreleasedChanges query resolver and diff helpers**

Create `internal/graph/unreleased_changes.go`:

```go
package graph

import (
	"context"

	"trip2g/internal/case/checkapikey"
	"trip2g/internal/case/listunreleasedchanges"
	"trip2g/internal/graph/model"
)

func (r *queryResolver) UnreleasedChanges(ctx context.Context, filter model.NoteChangesFilter) (*model.UnreleasedChangesConnection, error) {
	if _, err := checkapikey.Resolve(ctx, r.env(ctx), "unreleased_changes"); err != nil {
		return nil, err
	}
	changes, err := listunreleasedchanges.Resolve(ctx, r.env(ctx), filter)
	if err != nil {
		return nil, err
	}
	return &model.UnreleasedChangesConnection{
		TotalCount: len(changes),
		Nodes:      changes,
	}, nil
}

// Stats, UnifiedDiff, WordDiff, TotalStats are lazy forceResolver fields.
// All three diff fields (stats, unifiedDiff, wordDiff) share one diff computation
// per node via sync.Once embedded in model.UnreleasedChange (custom model, not generated).
// See internal/graph/model/unreleased_change.go.

func (r *unreleasedChangeResolver) Stats(_ context.Context, obj *model.UnreleasedChange) (*model.UnreleasedChangeStats, error) {
	d := obj.Diff()
	return &model.UnreleasedChangeStats{
		AddedLines:   d.AddedLines,
		RemovedLines: d.RemovedLines,
		ChangedWords: d.ChangedWords,
	}, nil
}

func (r *unreleasedChangeResolver) UnifiedDiff(_ context.Context, obj *model.UnreleasedChange) (string, error) {
	return obj.Diff().Unified, nil
}

func (r *unreleasedChangeResolver) WordDiff(_ context.Context, obj *model.UnreleasedChange) (string, error) {
	return obj.Diff().Word, nil
}

func (r *unreleasedChangesConnectionResolver) TotalStats(_ context.Context, obj *model.UnreleasedChangesConnection) (*model.UnreleasedChangeStats, error) {
	var s model.UnreleasedChangeStats
	for _, ch := range obj.Nodes {
		d := ch.Diff()
		s.AddedLines += d.AddedLines
		s.RemovedLines += d.RemovedLines
		s.ChangedWords += d.ChangedWords
	}
	return &s, nil
}
```

- [ ] **Step 2: Register resolver structs**

gqlgen generates `UnreleasedChange()` and `UnreleasedChangesConnection()` method stubs in `schema.resolvers.go`. Confirm they exist; if not, add them:

```go
// in schema.resolvers.go, near the other resolver struct registrations at the bottom:
type unreleasedChangeResolver struct{ *Resolver }
type unreleasedChangesConnectionResolver struct{ *Resolver }
```

Also check that gqlgen generated `func (r *Resolver) UnreleasedChange() UnreleasedChangeResolver` and `func (r *Resolver) UnreleasedChangesConnection() UnreleasedChangesConnectionResolver` — if it did, they're fine as-is. If it generated panic stubs for `UnreleasedChanges`/`Stats`/`UnifiedDiff`/`WordDiff`/`TotalStats` in `schema.resolvers.go`, delete those (the real implementations are in `unreleased_changes.go`).

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

Expected: clean build, no errors. If gqlgen generated duplicate stubs for `UnifiedDiff`/`WordDiff`/`UnreleasedChanges` in `schema.resolvers.go`, delete those stubs (the implementations in `unreleased_changes.go` take priority).

- [ ] **Step 4: Run all tests**

```bash
go test ./internal/... -count=1
```

Expected: all tests PASS.

- [ ] **Step 5: Run lint and fix**

```bash
make lint
```

Common issues to fix:
- Import order (`goimports` — group stdlib / external / internal)
- Unused imports
- `golangci-lint` cognitive complexity warnings (split large functions if flagged)
- `testifylint` — use `require.Len` not `require.Equal(t, N, len(...))`

Fix all reported issues, then re-run until clean:

```bash
make lint
```

Expected: no output (exit 0).

- [ ] **Step 6: Commit**

```bash
git add internal/graph/unreleased_changes.go internal/graph/schema.resolvers.go
git commit -m "feat(graph): UnreleasedChanges resolver with unified and word diff fields"
```

---

## Task 4: E2E Integration Test

**Files:**
- Create: `e2e/unreleased-changes.spec.js`

**Delegate to a sub-agent:** spawn an `oh-my-claudecode:qa-tester` agent with this context and goal:

> Write and verify an e2e Playwright spec at `e2e/unreleased-changes.spec.js`. Use the patterns from `e2e/queue.spec.js` and `scripts/test-e2e.sh` as reference (API key from `.test-api-key`, GraphQL calls via `X-Api-Key` header to `/_system/graphql`). Config mutations use admin session cookie.
>
> **Part 1 — `unreleasedChanges` API:**
> 1. Read API key from `.test-api-key`; sign in as admin to get session cookie
> 2. Call `createRelease` → assert `unreleasedChanges(filter: {includePatterns: ["**"]})` returns `totalCount: 0`
> 3. Push 2 notes via `pushNotes` + `commitNotes` → assert `totalCount: 2`, both `changeType: ADDED`
> 4. Modify one note → assert it appears as `MODIFIED` with non-empty `unifiedDiff`
> 5. Hide one note via `hideNotes` + `commitNotes` → assert it appears as `REMOVED`
> 6. Call `createRelease` again → assert `totalCount` resets to 0
>
> **Part 2 — `ShowDraftVersions` block:**
> Context: `ShowDraftVersions` (`show_draft_versions` in site config, `internal/configregistry/registry.go`) controls whether admins see latest drafts. Has not been tested for ~6 months — may be broken. If so, investigate and fix before writing assertions.
> Structure: `ShowDraftVersions=false` at the start of the block, `ShowDraftVersions=true` at the end.
> 7. **Set `show_draft_versions = false`**
> 8. Push a note with content A, `commitNotes`, `createRelease` (live = A). Update the note to content B, `commitNotes` (latest = B, live = A).
> 9. Guest (unauthenticated) fetches the note page → assert contains A (live), not B
> 10. Admin (authenticated) fetches the note page → assert contains A (live) — ShowDraftVersions is off, so even admins see live
> 11. **Set `show_draft_versions = true`**
> 12. Admin fetches the note page → assert contains B (latest) — ShowDraftVersions is on
> 13. Guest fetches the note page → assert contains A (live) — guests always see live
>
> Verify the spec runs green on the dev server (`make air` must be running, app on `:8081`).
> Run: `APP_URL=http://localhost:8081 npx playwright test e2e/unreleased-changes.spec.js --reporter=line`
> Fix until all assertions pass. Commit: `test(e2e): unreleased-changes and showDraftVersions integration spec`

- [ ] **Step 1: Delegate to qa-tester sub-agent**

Spawn `oh-my-claudecode:qa-tester` (model: sonnet) with the prompt above. Wait for it to return passing results before marking this task complete.

---

## Self-Review Checklist

**Spec coverage:**
- `unreleasedChanges` query ✓ (Task 1 schema + Task 3 resolver)
- `NoteChangesFilter` reused ✓ (resolve.go + schema)
- `UnreleasedChangesConnection` with `totalCount` + `totalStats` ✓ (Task 1 schema + Task 3 resolver)
- `added/modified/removed` enum ✓ (Task 1 schema)
- `unifiedDiff` + `wordDiff` lazy fields ✓ (Task 3)
- `oldContent`/`newContent` eager fields ✓ (Task 2 use case + Task 3 resolver)
- Per-node `stats` lazy ✓ (Task 1 `forceResolver` + Task 3 `Stats` resolver via `obj.Diff()`)
- `totalStats` lazy ✓ (Task 1 `forceResolver` + Task 3 `TotalStats` resolver)
- `sync.Once` per-node diff cache ✓ (Task 1 custom model `Diff()` method)
- API key auth ✓ (Task 3 `checkapikey.Resolve`)
- Glob filter ✓ (Task 2 `matchesFilter` using `webhookutil.MatchesAny`)
- No live release → all notes added ✓ (Task 2 test + logic)
- No new SQL query needed ✓ (uses existing `AllLiveNotes`)
- No new dependencies ✓ (`pmezard/go-difflib` already in `go.mod`)

**Potential issues to verify after codegen:**
- `model.NoteChangeType` constants: gqlgen generates `NoteChangeTypeAdded = "ADDED"` etc. Used directly in `listunreleasedchanges/resolve.go`.
- `model.UnreleasedChangesConnection.Nodes` type: likely `[]*model.UnreleasedChange` — confirm after codegen and adjust if needed.
- gqlgen must NOT generate `UnreleasedChange` in `models_gen.go` — verify after running `make gqlgen`. If it does, the `gqlgen.yml` model mapping is missing or wrong.
- `model.UnreleasedChangesConnection.Nodes` may be `[]*model.UnreleasedChange` (pointer slice) — adjust Task 3 accordingly.
- `unreleasedChangeResolver` struct: gqlgen may generate it in `schema.resolvers.go` — if so, delete duplicate in Task 3.
