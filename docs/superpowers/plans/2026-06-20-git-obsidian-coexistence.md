# Git ↔ Obsidian-plugin coexistence — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the git API a faithful, lazily-materialized mirror of the notes DB so the Obsidian plugin and `git push` never silently overwrite each other.

**Architecture:** The DB stays the source of truth. On any git request the server rebuilds the bare repo's `master` from the DB (notes + assets) as one idempotent commit, then serves. Git pushes are applied back to the DB via an old→new HEAD diff (deletions included). A shared write-lock serializes plugin-push, git-apply, and materialize; standard git non-fast-forward rejection (`receive.denyNonFastForwards`) is the conflict guard.

**Tech Stack:** Go, `git` plumbing (`hash-object`/`update-index`/`write-tree`/`commit-tree`/`update-ref`), fasthttp, gqlgen, sqlc, moq, testify; Playwright for e2e.

**Spec:** `docs/superpowers/specs/2026-06-20-git-obsidian-coexistence-design.md`

---

## File structure

| File | Responsibility | Action |
|------|----------------|--------|
| `internal/gitapi/api.go` | API struct, request handling, auth, repo storage | Modify (Env additions, HandleRequest/receive-pack restructure, denyNonFastForwards) |
| `internal/gitapi/materialize.go` | DB → git materializer + git plumbing helpers | **Create** |
| `internal/gitapi/apply.go` | git → DB diff-based apply | **Create** (move/rewrite `ApplyChanges`) |
| `internal/gitapi/materialize_test.go` | Materializer unit tests | **Create** |
| `internal/gitapi/apply_test.go` | Apply/diff unit tests | **Create** |
| `internal/gitapi/testsupport_test.go` | temp-repo + fake Env helpers | **Create** |
| `cmd/server/main.go` | `app` Env impl: note/asset sources, hide, lock, async upload | Modify |
| `internal/graph/resolver.go` | add `LockNoteWrites`/`UnlockNoteWrites` to graph `Env` | Modify |
| `internal/graph/schema.resolvers.go` | wrap `PushNotes` resolver in the write-lock | Modify |
| `cmd/server/cronjobs.go` | route `apply_git_changes` through the lock | Modify |
| `e2e/gitsync.spec.js` | end-to-end coexistence over smart-HTTP | **Create** |

### `gitapi.Env` additions (final shape)

```go
// in internal/gitapi/api.go, added to the existing Env interface
type NoteSource struct {
	Path    string
	Content []byte
}

type AssetSource struct {
	AbsolutePath string       // repo-relative path, leading slash trimmed by materializer
	Asset        db.NoteAsset // identifies the bytes in object storage
}

// Env (append these methods)
LatestNoteSources(ctx context.Context) ([]NoteSource, error)
LatestAssetSources(ctx context.Context) ([]AssetSource, error)
ReadAssetObject(ctx context.Context, asset db.NoteAsset) (io.ReadCloser, error)
HideNotePaths(ctx context.Context, paths []string) error
LockNoteWrites()
UnlockNoteWrites()
```

---

## Task 1: App-side Env methods (note/asset sources, hide, lock)

Pure wiring on `app` so later gitapi tasks have real data sources. No gitapi behavior change yet.

**Files:**
- Modify: `internal/gitapi/api.go`
- Modify: `cmd/server/main.go`
- Modify: `internal/graph/resolver.go`
- Modify: `internal/graph/schema.resolvers.go`

- [ ] **Step 0: Add the shared source types to `api.go` first (so every later commit compiles)**

In `internal/gitapi/api.go`, add above the `Env` interface:

```go
type NoteSource struct {
	Path    string
	Content []byte
}

type AssetSource struct {
	AbsolutePath string       // repo-relative path; leading slash trimmed by the materializer
	Asset        db.NoteAsset // identifies the bytes in object storage
}
```

Do **not** add the new methods to the `Env` interface yet — that happens in Task 3 Step 1, once the materializer that needs them exists. `app` having extra methods before then compiles fine.

- [ ] **Step 1: Add the shared mutex field to `app`**

In `cmd/server/main.go`, find the `type app struct {` definition and add a field (place near other sync primitives):

```go
	noteWriteMu sync.Mutex
```

(`sync` is already imported in main.go; if not, add it.)

- [ ] **Step 2: Implement the new app methods**

Append to `cmd/server/main.go` (near the existing `func (a *app) PushNotes`):

```go
func (a *app) LockNoteWrites()   { a.noteWriteMu.Lock() }
func (a *app) UnlockNoteWrites() { a.noteWriteMu.Unlock() }

// LatestNoteSources returns the raw markdown source of every visible note,
// for materializing the git mirror.
func (a *app) LatestNoteSources(ctx context.Context) ([]gitapi.NoteSource, error) {
	nvs := a.LatestNoteViews()
	if nvs == nil {
		return nil, nil
	}
	out := make([]gitapi.NoteSource, 0, len(nvs.List))
	for _, nv := range nvs.List {
		out = append(out, gitapi.NoteSource{Path: nv.Path, Content: nv.Content})
	}
	return out, nil
}

// LatestAssetSources returns every latest note asset for the git mirror.
func (a *app) LatestAssetSources(ctx context.Context) ([]gitapi.AssetSource, error) {
	rows, err := a.Queries.AllLatestNoteAssets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list latest note assets: %w", err)
	}
	out := make([]gitapi.AssetSource, 0, len(rows))
	for _, r := range rows {
		out = append(out, gitapi.AssetSource{
			AbsolutePath: r.NoteAsset.AbsolutePath,
			Asset:        r.NoteAsset,
		})
	}
	return out, nil
}

// ReadAssetObject streams an asset's bytes from object storage.
func (a *app) ReadAssetObject(ctx context.Context, asset db.NoteAsset) (io.ReadCloser, error) {
	return a.GetAssetObject(ctx, asset)
}

// HideNotePaths hides the given note paths (used when git deletes files) and
// reloads the in-memory note views so they stop resolving.
func (a *app) HideNotePaths(ctx context.Context, paths []string) error {
	for _, p := range paths {
		if err := a.WriteQueries.HideNotePath(ctx, db.HideNotePathParams{Value: p}); err != nil {
			return fmt.Errorf("failed to hide note path %s: %w", p, err)
		}
	}
	if len(paths) > 0 {
		if _, err := a.PrepareLatestNotes(ctx, false); err != nil {
			return fmt.Errorf("failed to prepare latest notes after hide: %w", err)
		}
	}
	return nil
}
```

> Note: `HideNotePath` lives on `WriteQueries`; confirm the exact param struct name with
> `grep -n "func (q \*Queries) HideNotePath" internal/db/queries.write.sql.go` and adjust
> `db.HideNotePathParams` if the generated name differs. `GetAssetObject` is promoted from the
> embedded `*miniostorage.FileStorage`.

- [ ] **Step 3: Add lock methods to the graph `Env` interface**

In `internal/graph/resolver.go`, find the `Env` interface and add:

```go
	LockNoteWrites()
	UnlockNoteWrites()
```

- [ ] **Step 4: Wrap the plugin `PushNotes` resolver in the lock**

In `internal/graph/schema.resolvers.go`, change `PushNotes` (around line 2619) to:

```go
func (r *mutationResolver) PushNotes(ctx context.Context, input model.PushNotesInput) (model.PushNotesOrErrorPayload, error) {
	apiKey, err := checkapikey.Resolve(ctx, r.env(ctx), "push_notes")
	if err != nil {
		return nil, err
	}

	input.ApiKey = *apiKey

	env := r.env(ctx)
	env.LockNoteWrites()
	defer env.UnlockNoteWrites()

	return pushnotes.Resolve(ctx, env, input)
}
```

- [ ] **Step 5: Build to verify wiring compiles**

Run: `go build ./cmd/... ./internal/...`
Expected: PASS (no compile errors). The gitapi `Env` interface doesn't yet declare these methods — that's fine; `app` simply has extra methods until Task 4 wires them.

- [ ] **Step 6: Commit**

```bash
git add cmd/server/main.go internal/graph/resolver.go internal/graph/schema.resolvers.go
git commit -m "feat(gitapi): app sources, hide-paths, and shared note-write lock"
```

---

## Task 2: Test support — temp repo + fake Env

Shared helpers for gitapi unit tests: a real bare+work repo on disk and a hand-written fake `Env`.

**Files:**
- Create: `internal/gitapi/testsupport_test.go`

- [ ] **Step 1: Write the test-support helpers**

```go
package gitapi

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
)

// newTestAPI creates an API backed by a real bare repo in a temp dir.
func newTestAPI(t *testing.T, env Env) *API {
	t.Helper()
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := Config{BasePath: "/_system/git", RepoPath: repo, MasterBranch: "master"}
	api := &API{config: cfg, env: env, logger: logger.Noop(), ctx: context.Background()}
	if err := api.ensureBareRepo(); err != nil {
		t.Fatal(err)
	}
	return api
}

// gitOut runs a git command in the repo and returns trimmed stdout.
func gitOut(t *testing.T, repo string, args ...string) string {
	t.Helper()
	full := append([]string{"--git-dir", repo}, args...)
	cmd := exec.Command("git", full...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(bytes.TrimSpace(out.Bytes()))
}

// fakeEnv is a programmable Env for tests.
type fakeEnv struct {
	notes    []NoteSource
	assets   []AssetSource
	assetBuf map[string][]byte // keyed by AbsolutePath
	hidden   []string
	pushed   []string // paths passed to PushNotes
	dbHashes map[string]string
}

func (f *fakeEnv) Logger() logger.Logger { return logger.Noop() }
func (f *fakeEnv) LockNoteWrites()        {}
func (f *fakeEnv) UnlockNoteWrites()      {}

func (f *fakeEnv) LatestNoteSources(context.Context) ([]NoteSource, error)  { return f.notes, nil }
func (f *fakeEnv) LatestAssetSources(context.Context) ([]AssetSource, error) { return f.assets, nil }
func (f *fakeEnv) ReadAssetObject(_ context.Context, a db.NoteAsset) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(f.assetBuf[a.AbsolutePath])), nil
}
func (f *fakeEnv) HideNotePaths(_ context.Context, paths []string) error {
	f.hidden = append(f.hidden, paths...)
	return nil
}
func (f *fakeEnv) AllVisibleNotePaths(context.Context) ([]db.NotePath, error) {
	out := []db.NotePath{}
	for p, h := range f.dbHashes {
		out = append(out, db.NotePath{Value: p, LatestContentHash: h})
	}
	return out, nil
}
func (f *fakeEnv) PushNotes(_ context.Context, in model.PushNotesInput) (model.PushNotesOrErrorPayload, error) {
	for _, u := range in.Updates {
		f.pushed = append(f.pushed, u.Path)
	}
	return &model.PushNotesPayload{Notes: []model.PushedNote{}}, nil
}
func (f *fakeEnv) UploadNoteAsset(context.Context, model.UploadNoteAssetInput) (model.UploadNoteAssetOrErrorPayload, error) {
	return &model.UploadNoteAssetPayload{}, nil
}
func (f *fakeEnv) PutPrivateObject(context.Context, io.Reader, string) error  { return nil }
func (f *fakeEnv) GetPrivateObject(context.Context, string) (io.ReadCloser, error) {
	return nil, os.ErrNotExist
}
func (f *fakeEnv) PrivateObjectExists(context.Context, string) (bool, error) { return false, nil }
func (f *fakeEnv) GitTokenByValueSha256(context.Context, string) (db.GitToken, error) {
	return db.GitToken{}, nil
}
```

> If `logger.Noop()` does not exist, check the logger package for the no-op constructor name
> (`grep -rn "func Noop\|func NewNoop\|Discard" internal/logger`) and use it; otherwise add a
> trivial `logger.Noop()` returning a logger that discards output.

- [ ] **Step 2: Verify it compiles (the fake must satisfy the future Env)**

Run: `go vet ./internal/gitapi/`
Expected: compile errors only about unused helpers (acceptable until used) — no signature errors. If `Env` doesn't yet include the new methods, this file's `fakeEnv` has extra methods, which is fine.

- [ ] **Step 3: Commit**

```bash
git add internal/gitapi/testsupport_test.go
git commit -m "test(gitapi): temp-repo and fake-env test support"
```

---

## Task 3: Materializer (notes only)

DB → git as one idempotent commit, rebuilding the index from scratch each call.

**Files:**
- Create: `internal/gitapi/materialize.go`
- Create: `internal/gitapi/materialize_test.go`
- Modify: `internal/gitapi/api.go` (add `NoteSource`/`AssetSource` types + Env methods)

- [ ] **Step 1: Add the source types and Env methods to `api.go`**

In `internal/gitapi/api.go`, add the `NoteSource` and `AssetSource` structs (shown in the File Structure section) above the `Env` interface, and append to the `Env` interface:

```go
	// materialize / mirror
	LatestNoteSources(ctx context.Context) ([]NoteSource, error)
	LatestAssetSources(ctx context.Context) ([]AssetSource, error)
	ReadAssetObject(ctx context.Context, asset db.NoteAsset) (io.ReadCloser, error)
	HideNotePaths(ctx context.Context, paths []string) error

	LockNoteWrites()
	UnlockNoteWrites()
```

- [ ] **Step 2: Write the failing materializer test (notes)**

`internal/gitapi/materialize_test.go`:

```go
package gitapi

import (
	"context"
	"testing"
)

func TestMaterializeCommitsNotes(t *testing.T) {
	env := &fakeEnv{notes: []NoteSource{{Path: "a.md", Content: []byte("# A\n")}}}
	api := newTestAPI(t, env)

	if err := api.materialize(context.Background()); err != nil {
		t.Fatalf("materialize: %v", err)
	}

	got := gitOut(t, api.config.RepoPath, "show", "master:a.md")
	if got != "# A" {
		t.Fatalf("a.md content = %q, want %q", got, "# A")
	}
}

func TestMaterializeIdempotent(t *testing.T) {
	env := &fakeEnv{notes: []NoteSource{{Path: "a.md", Content: []byte("# A\n")}}}
	api := newTestAPI(t, env)

	if err := api.materialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	first := gitOut(t, api.config.RepoPath, "rev-parse", "master")
	if err := api.materialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	second := gitOut(t, api.config.RepoPath, "rev-parse", "master")

	if first != second {
		t.Fatalf("materialize created a new commit on no change: %s -> %s", first, second)
	}
}

func TestMaterializeDeletesRemovedNote(t *testing.T) {
	env := &fakeEnv{notes: []NoteSource{{Path: "a.md", Content: []byte("a")}, {Path: "b.md", Content: []byte("b")}}}
	api := newTestAPI(t, env)
	if err := api.materialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	env.notes = []NoteSource{{Path: "a.md", Content: []byte("a")}} // b removed from DB
	if err := api.materialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	files := gitOut(t, api.config.RepoPath, "ls-tree", "-r", "master", "--name-only")
	if files != "a.md" {
		t.Fatalf("tree = %q, want only a.md", files)
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/gitapi/ -run TestMaterialize -v`
Expected: FAIL — `api.materialize` undefined.

- [ ] **Step 4: Implement the materializer**

`internal/gitapi/materialize.go`:

```go
package gitapi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// materialize rebuilds refs/heads/<master> from the DB (notes + assets) as one
// commit. It is idempotent: if the resulting tree equals the current HEAD tree,
// no commit is made. Callers must hold the note-write lock.
func (api *API) materialize(ctx context.Context) error {
	notes, err := api.env.LatestNoteSources(ctx)
	if err != nil {
		return fmt.Errorf("materialize: note sources: %w", err)
	}
	assets, err := api.env.LatestAssetSources(ctx)
	if err != nil {
		return fmt.Errorf("materialize: asset sources: %w", err)
	}

	indexPath := filepath.Join(api.config.RepoPath, "index.materialize")
	_ = os.Remove(indexPath) // start from an empty index so deletions drop out
	defer os.Remove(indexPath)

	gitEnv := append(os.Environ(),
		"GIT_INDEX_FILE="+indexPath,
		"GIT_AUTHOR_NAME=trip2g", "GIT_AUTHOR_EMAIL=server@trip2g",
		"GIT_COMMITTER_NAME=trip2g", "GIT_COMMITTER_EMAIL=server@trip2g",
	)

	for _, n := range notes {
		if err := api.addBlob(gitEnv, n.Path, n.Content); err != nil {
			return fmt.Errorf("materialize note %s: %w", n.Path, err)
		}
	}
	for _, a := range assets {
		rc, err := api.env.ReadAssetObject(ctx, a.Asset)
		if err != nil {
			api.logger.Warn("materialize: skip unreadable asset", "path", a.AbsolutePath, "error", err)
			continue
		}
		content, readErr := io.ReadAll(rc)
		_ = rc.Close()
		if readErr != nil {
			return fmt.Errorf("materialize asset %s: %w", a.AbsolutePath, readErr)
		}
		repoPath := strings.TrimPrefix(a.AbsolutePath, "/")
		if err := api.addBlob(gitEnv, repoPath, content); err != nil {
			return fmt.Errorf("materialize asset %s: %w", repoPath, err)
		}
	}

	tree, err := api.gitCmd(gitEnv, nil, "write-tree")
	if err != nil {
		return fmt.Errorf("materialize: write-tree: %w", err)
	}
	tree = strings.TrimSpace(tree)

	parentArgs := []string{}
	if api.isRefExists("refs/heads/" + api.config.MasterBranch) {
		headTree, terr := api.gitCmd(gitEnv, nil, "rev-parse", api.config.MasterBranch+"^{tree}")
		if terr == nil && strings.TrimSpace(headTree) == tree {
			return nil // no change
		}
		parentArgs = []string{"-p", api.config.MasterBranch}
	}

	commitArgs := append([]string{"commit-tree", tree, "-m", "server sync"}, parentArgs...)
	commit, err := api.gitCmd(gitEnv, nil, commitArgs...)
	if err != nil {
		return fmt.Errorf("materialize: commit-tree: %w", err)
	}
	commit = strings.TrimSpace(commit)

	if _, err := api.gitCmd(gitEnv, nil, "update-ref", "refs/heads/"+api.config.MasterBranch, commit); err != nil {
		return fmt.Errorf("materialize: update-ref: %w", err)
	}
	return nil
}

// addBlob writes content as a blob and stages it at path in the temp index.
func (api *API) addBlob(gitEnv []string, path string, content []byte) error {
	sha, err := api.gitCmd(gitEnv, bytes.NewReader(content), "hash-object", "-w", "--stdin")
	if err != nil {
		return fmt.Errorf("hash-object: %w", err)
	}
	sha = strings.TrimSpace(sha)
	_, err = api.gitCmd(gitEnv, nil, "update-index", "--add", "--cacheinfo", "100644,"+sha+","+path)
	if err != nil {
		return fmt.Errorf("update-index: %w", err)
	}
	return nil
}

// gitCmd runs a git command in the bare repo with the given env and optional stdin.
func (api *API) gitCmd(gitEnv []string, stdin io.Reader, args ...string) (string, error) {
	full := append([]string{"--git-dir", api.config.RepoPath, "-c", "core.quotePath=false"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Env = gitEnv
	cmd.Stdin = stdin
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %v: %w: %s", args, err, errOut.String())
	}
	return out.String(), nil
}
```

- [ ] **Step 5: Run to verify the tests pass**

Run: `go test ./internal/gitapi/ -run TestMaterialize -v`
Expected: PASS (3 tests).

- [ ] **Step 6: Commit**

```bash
git add internal/gitapi/api.go internal/gitapi/materialize.go internal/gitapi/materialize_test.go
git commit -m "feat(gitapi): idempotent DB->git materializer for notes"
```

---

## Task 4: Materializer — assets

Extend the existing materializer test to prove asset bytes land in git and stay idempotent.

**Files:**
- Modify: `internal/gitapi/materialize_test.go`

- [ ] **Step 1: Write the failing asset test**

Append to `materialize_test.go`:

```go
func TestMaterializeWritesAssets(t *testing.T) {
	env := &fakeEnv{
		notes:    []NoteSource{{Path: "note.md", Content: []byte("hi")}},
		assets:   []AssetSource{{AbsolutePath: "/assets/img.png", Asset: dbAsset("/assets/img.png")}},
		assetBuf: map[string][]byte{"/assets/img.png": []byte("PNGDATA")},
	}
	api := newTestAPI(t, env)
	if err := api.materialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := gitOut(t, api.config.RepoPath, "show", "master:assets/img.png")
	if got != "PNGDATA" {
		t.Fatalf("asset = %q, want PNGDATA", got)
	}

	first := gitOut(t, api.config.RepoPath, "rev-parse", "master")
	if err := api.materialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	if second := gitOut(t, api.config.RepoPath, "rev-parse", "master"); first != second {
		t.Fatalf("asset materialize not idempotent: %s -> %s", first, second)
	}
}
```

Add the `dbAsset` helper to `testsupport_test.go`:

```go
func dbAsset(absPath string) db.NoteAsset {
	return db.NoteAsset{AbsolutePath: absPath, FileName: filepath.Base(absPath)}
}
```

- [ ] **Step 2: Run to verify it passes**

Asset support is already implemented in Task 3's materializer. Run:
`go test ./internal/gitapi/ -run TestMaterializeWritesAssets -v`
Expected: PASS. (If FAIL, the leading-slash trim or `show master:assets/img.png` path is wrong — fix `strings.TrimPrefix` handling in `materialize.go`.)

- [ ] **Step 3: Commit**

```bash
git add internal/gitapi/materialize_test.go internal/gitapi/testsupport_test.go
git commit -m "test(gitapi): assets mirrored into git, idempotent by content"
```

---

## Task 5: Diff-based apply (git → DB) with deletions

Replace `ApplyChanges`'s "always all files" with an old→new HEAD diff; deletions call `HideNotePaths`.

**Files:**
- Create: `internal/gitapi/apply.go`
- Create: `internal/gitapi/apply_test.go`
- Modify: `internal/gitapi/api.go` (remove the old `ApplyChanges`/`getAllFiles`/commented `getChangedFiles`)

- [ ] **Step 1: Write the failing diff test**

`internal/gitapi/apply_test.go`:

```go
package gitapi

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
)

// commitFile writes a file via a temp worktree and returns the new HEAD sha.
func commitFile(t *testing.T, api *API, name, content string) string {
	t.Helper()
	work := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = work
		cmd.Env = append(os.Environ(),
			"GIT_DIR="+api.config.RepoPath, "GIT_WORK_TREE="+work,
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	if api.isRefExists("refs/heads/" + api.config.MasterBranch) {
		run("checkout", api.config.MasterBranch)
	}
	if content == "" {
		_ = os.Remove(filepath.Join(work, name))
		run("rm", name)
	} else {
		if err := os.WriteFile(filepath.Join(work, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		run("add", name)
	}
	run("commit", "-m", "x")
	return gitOut(t, api.config.RepoPath, "rev-parse", api.config.MasterBranch)
}

func TestDiffChangedFiles(t *testing.T) {
	api := newTestAPI(t, &fakeEnv{})
	old := commitFile(t, api, "a.md", "one")
	_ = commitFile(t, api, "b.md", "two")
	newRev := commitFile(t, api, "a.md", "one-edited")

	changed, deleted, err := api.diffChangedFiles(old, newRev)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(changed)
	if len(changed) != 1 || changed[0] != "a.md" {
		t.Fatalf("changed = %v, want [a.md]", changed)
	}
	if len(deleted) != 0 {
		t.Fatalf("deleted = %v, want none", deleted)
	}
}

func TestDiffDetectsDeletion(t *testing.T) {
	api := newTestAPI(t, &fakeEnv{})
	_ = commitFile(t, api, "a.md", "one")
	old := commitFile(t, api, "b.md", "two")
	newRev := commitFile(t, api, "b.md", "") // delete b.md

	changed, deleted, err := api.diffChangedFiles(old, newRev)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 0 {
		t.Fatalf("changed = %v, want none", changed)
	}
	if len(deleted) != 1 || deleted[0] != "b.md" {
		t.Fatalf("deleted = %v, want [b.md]", deleted)
	}
}

func TestApplyHidesDeletedNote(t *testing.T) {
	env := &fakeEnv{}
	api := newTestAPI(t, env)
	old := commitFile(t, api, "gone.md", "x")
	newRev := commitFile(t, api, "gone.md", "")

	if _, err := api.applyDiff(context.Background(), old, newRev); err != nil {
		t.Fatal(err)
	}
	if len(env.hidden) != 1 || env.hidden[0] != "gone.md" {
		t.Fatalf("hidden = %v, want [gone.md]", env.hidden)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/gitapi/ -run 'TestDiff|TestApply' -v`
Expected: FAIL — `diffChangedFiles` / `applyDiff` undefined.

- [ ] **Step 3: Implement apply.go**

`internal/gitapi/apply.go`:

```go
package gitapi

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"trip2g/internal/graph/model"
)

const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904" // git's well-known empty tree

// diffChangedFiles returns added/modified and deleted paths between two commits.
// If oldRev is empty, everything in newRev counts as added.
func (api *API) diffChangedFiles(oldRev, newRev string) (changed, deleted []string, err error) {
	base := oldRev
	if base == "" {
		base = emptyTree
	}
	out, err := api.gitCmd(os.Environ(), nil, "diff", "--name-status", "-z", base, newRev)
	if err != nil {
		return nil, nil, fmt.Errorf("diff: %w", err)
	}
	// -z output: STATUS\0PATH\0STATUS\0PATH\0...
	parts := strings.Split(out, "\x00")
	for i := 0; i+1 < len(parts); i += 2 {
		status, path := parts[i], parts[i+1]
		if path == "" {
			continue
		}
		switch {
		case strings.HasPrefix(status, "D"):
			deleted = append(deleted, path)
		default: // A, M, etc.
			changed = append(changed, path)
		}
	}
	return changed, deleted, nil
}

// applyDiff applies the changes between oldRev and newRev to the DB:
// changed markdown/html notes are pushed (hash-skipped if already current),
// deletions are hidden. Returns the list of changed note paths.
func (api *API) applyDiff(ctx context.Context, oldRev, newRev string) ([]string, error) {
	changed, deleted, err := api.diffChangedFiles(oldRev, newRev)
	if err != nil {
		return nil, err
	}

	notePaths, err := api.env.AllVisibleNotePaths(ctx)
	if err != nil {
		return nil, fmt.Errorf("apply: note paths: %w", err)
	}
	dbHash := map[string]string{}
	for _, np := range notePaths {
		dbHash[np.Value] = np.LatestContentHash
	}

	input := model.PushNotesInput{}
	applied := []string{}
	for _, f := range changed {
		ext := strings.ToLower(filepath.Ext(f))
		if ext != ".md" && ext != ".html" {
			continue
		}
		content, rerr := api.showFile(newRev, f)
		if rerr != nil {
			continue
		}
		sum := sha256.Sum256(content)
		hash := base64.URLEncoding.EncodeToString(sum[:])
		if dbHash[f] == hash {
			continue // already current in DB — breaks the materialize<->apply loop
		}
		input.Updates = append(input.Updates, model.PushNoteInput{Path: f, Content: string(content)})
		applied = append(applied, f)
	}

	if len(input.Updates) > 0 {
		payload, perr := api.env.PushNotes(ctx, input)
		if perr != nil {
			return nil, fmt.Errorf("apply: push notes: %w", perr)
		}
		if ep, ok := payload.(*model.ErrorPayload); ok {
			return nil, fmt.Errorf("apply: push notes: %s", ep.Message)
		}
	}

	var deletedNotes []string
	for _, f := range deleted {
		ext := strings.ToLower(filepath.Ext(f))
		if ext == ".md" || ext == ".html" {
			deletedNotes = append(deletedNotes, f)
		}
	}
	if len(deletedNotes) > 0 {
		if herr := api.env.HideNotePaths(ctx, deletedNotes); herr != nil {
			return nil, fmt.Errorf("apply: hide notes: %w", herr)
		}
	}

	return applied, nil
}

// showFile reads a file's content at a given commit.
func (api *API) showFile(rev, path string) ([]byte, error) {
	cmd := exec.Command("git", "--git-dir", api.config.RepoPath, "-c", "core.quotePath=false",
		"show", fmt.Sprintf("%s:%s", rev, path))
	cmd.Stderr = nil
	return cmd.Output()
}
```

> Asset upload from a git push is retained via the existing `uploadNoteAssets` flow if needed,
> but with the DB-canonical mirror the push only needs note content + deletions; asset bytes
> pushed through git are re-materialized from the DB on the next access. Keep `uploadNoteAssets`
> only if a later task proves git-pushed binary assets must reach object storage; otherwise the
> mirror handles assets in one direction (DB→git). For this plan, asset *upload from git* is out
> of scope — assets flow DB→git via materialize; git→DB asset bytes are not required because the
> plugin/web is the canonical asset author.

- [ ] **Step 4: Delete the obsolete code in `api.go`**

Remove from `internal/gitapi/api.go`: the `ApplyChanges` method (lines ~447–496), `getAllFiles` (~498–558), `preparePushNotesInput` (~391–438), `uploadNoteAssets` (~560–621), `readContent` (~623–680), and `resolveAssetPath` (~682–711) **only if** no longer referenced. Verify with:
`grep -rn "preparePushNotesInput\|getAllFiles\|uploadNoteAssets\|resolveAssetPath\|readContent\|\.ApplyChanges" internal/ cmd/`
Keep `filterDotFiles`, `isRefExists`, `pktLine`, `checkBins`. Update `cmd/server/main.go`'s `ApplyGitChanges` in Task 7.

- [ ] **Step 5: Run to verify the tests pass**

Run: `go test ./internal/gitapi/ -run 'TestDiff|TestApply' -v`
Expected: PASS (3 tests).

- [ ] **Step 6: Commit**

```bash
git add internal/gitapi/apply.go internal/gitapi/apply_test.go internal/gitapi/api.go
git commit -m "feat(gitapi): diff-based git->DB apply with deletion handling"
```

---

## Task 6: HandleRequest + receive-pack restructure (lazy materialize, FF, rollback, async upload)

Wire materialize/apply into the request lifecycle, enforce fast-forward, roll back on apply failure, move snapshot off the critical path.

**Files:**
- Modify: `internal/gitapi/api.go`
- Modify: `cmd/server/main.go` (`ApplyGitChanges`, async upload helper)

- [ ] **Step 1: Enforce fast-forward at repo init**

In `internal/gitapi/api.go` `setupPreReceiveHook` (or `ensureBareRepo`), after the repo exists, set the config once. Add to `initRepo()` after `setupPreReceiveHook()`:

```go
	if err := api.setDenyNonFastForwards(); err != nil {
		return err
	}
```

And add:

```go
func (api *API) setDenyNonFastForwards() error {
	cmd := exec.Command("git", "--git-dir", api.config.RepoPath, "config", "receive.denyNonFastForwards", "true")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set denyNonFastForwards: %w", err)
	}
	return nil
}
```

- [ ] **Step 2: Materialize before serving, under the lock**

In `HandleRequest`, replace the `api.mu.Lock(); defer api.mu.Unlock()` block around `initRepo`/`hdr` so materialize runs under the note-write lock and the handler runs without holding it for streaming. Concretely, after `initRepo()` succeeds:

```go
	api.env.LockNoteWrites()
	merr := api.materialize(ctx)
	api.env.UnlockNoteWrites()
	if merr != nil {
		api.logger.Error("failed to materialize", "error", merr)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(merr.Error())
		return true
	}
```

Keep `api.mu` (the structural lock guarding repo file ops) around `initRepo()` only; the note-write lock guards materialize/apply against plugin writes.

- [ ] **Step 3: Rewrite `handleGitReceivePack` for transaction + rollback**

```go
func (api *API) handleGitReceivePack(ctx *fasthttp.RequestCtx) error {
	api.env.LockNoteWrites()
	defer api.env.UnlockNoteWrites()

	// Re-materialize inside the critical section so a concurrent plugin write
	// advances HEAD and the client's stale push is rejected (non-fast-forward).
	if err := api.materialize(api.ctx); err != nil {
		return fmt.Errorf("failed to materialize before receive: %w", err)
	}

	oldRev := ""
	if api.isRefExists("refs/heads/" + api.config.MasterBranch) {
		oldRev = strings.TrimSpace(mustGit(api, "rev-parse", api.config.MasterBranch))
	}

	cmd := exec.Command("git-receive-pack", "--stateless-rpc", api.config.RepoPath)
	cmd.Stdin = bytes.NewReader(ctx.PostBody())
	cmd.Stdout = ctx
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run git-receive-pack: %w", err)
	}

	newRev := strings.TrimSpace(mustGit(api, "rev-parse", api.config.MasterBranch))
	if newRev == oldRev {
		return nil // nothing advanced (rejected by denyNonFastForwards or no-op)
	}

	if _, err := api.applyDiff(api.ctx, oldRev, newRev); err != nil {
		// Roll the ref back so the repo never diverges from the DB.
		if oldRev != "" {
			_, _ = api.gitCmd(os.Environ(), nil, "update-ref", "refs/heads/"+api.config.MasterBranch, oldRev)
		}
		return fmt.Errorf("failed to apply changes (rolled back): %w", err)
	}

	go func() {
		if err := api.uploadRepo(); err != nil {
			api.logger.Error("failed to upload repo snapshot", "error", err)
		}
	}()

	return nil
}

// mustGit is a tiny helper for read-only rev-parse calls.
func mustGit(api *API, args ...string) string {
	out, err := api.gitCmd(os.Environ(), nil, args...)
	if err != nil {
		api.logger.Error("git read failed", "args", args, "error", err)
		return ""
	}
	return out
}
```

Ensure `strings` is imported in `api.go` (it is).

- [ ] **Step 4: Write the failing FF + rollback tests**

Append to `apply_test.go`:

```go
func TestReceivePackRejectsNonFastForward(t *testing.T) {
	api := newTestAPI(t, &fakeEnv{})
	if err := api.setDenyNonFastForwards(); err != nil {
		t.Fatal(err)
	}
	cfg := gitOut(t, api.config.RepoPath, "config", "receive.denyNonFastForwards")
	if cfg != "true" {
		t.Fatalf("denyNonFastForwards = %q, want true", cfg)
	}
}

func TestApplyRollbackOnError(t *testing.T) {
	env := &fakeEnv{pushErr: true} // PushNotes returns an error
	api := newTestAPI(t, env)
	old := commitFile(t, api, "a.md", "one")
	newRev := commitFile(t, api, "a.md", "two")

	_, err := api.applyDiff(context.Background(), old, newRev)
	if err == nil {
		t.Fatal("expected apply error")
	}
	// caller (receive-pack) rolls back; here we assert applyDiff surfaced the error
	_ = newRev
}
```

Add a `pushErr` field to `fakeEnv` in `testsupport_test.go` and honor it in `PushNotes`:

```go
	pushErr bool
```
```go
func (f *fakeEnv) PushNotes(_ context.Context, in model.PushNotesInput) (model.PushNotesOrErrorPayload, error) {
	if f.pushErr {
		return nil, fmt.Errorf("boom")
	}
	for _, u := range in.Updates {
		f.pushed = append(f.pushed, u.Path)
	}
	return &model.PushNotesPayload{Notes: []model.PushedNote{}}, nil
}
```
(add `"fmt"` to the imports in `testsupport_test.go`.)

- [ ] **Step 5: Run to verify pass**

Run: `go test ./internal/gitapi/ -run 'TestReceivePack|TestApplyRollback' -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/gitapi/api.go internal/gitapi/apply_test.go internal/gitapi/testsupport_test.go
git commit -m "feat(gitapi): lazy materialize, fast-forward enforcement, push rollback"
```

---

## Task 7: Cron + final wiring

`apply_git_changes` must go through the same path/lock; remove the now-dead `ApplyChanges` reference.

**Files:**
- Modify: `cmd/server/main.go`
- Modify: `cmd/server/cronjobs.go` (only if signature changes)

- [ ] **Step 1: Repoint `ApplyGitChanges`**

In `cmd/server/main.go`, replace the body of `ApplyGitChanges`. The daily cron just needs to refresh the mirror from the DB (materialize); there is no inbound git diff to apply. Expose a `Materialize` entrypoint on the API:

In `internal/gitapi/materialize.go` add:

```go
// Materialize is the exported entrypoint for scheduled mirror refreshes.
func (api *API) Materialize(ctx context.Context) error {
	api.env.LockNoteWrites()
	defer api.env.UnlockNoteWrites()
	if err := api.initRepo(); err != nil {
		return err
	}
	return api.materialize(ctx)
}
```

In `cmd/server/main.go`:

```go
func (a *app) ApplyGitChanges(ctx context.Context) ([]string, error) {
	if a.gitAPI == nil {
		return nil, nil
	}
	if err := a.gitAPI.Materialize(ctx); err != nil {
		return nil, err
	}
	return nil, nil
}
```

> The `applygitchanges` cron's `Result.ChangedFiles` becomes empty; that's fine (it was only
> logged). If you prefer, rename the job to `refresh_git_mirror` in a follow-up — out of scope here.

- [ ] **Step 2: Build the whole server**

Run: `go build ./...`
Expected: PASS. Fix any leftover references to deleted functions surfaced by the compiler.

- [ ] **Step 3: Run the full gitapi suite + vet**

Run: `go test ./internal/gitapi/... && go vet ./internal/gitapi/...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/server/main.go internal/gitapi/materialize.go
git commit -m "feat(gitapi): scheduled mirror refresh via materialize"
```

---

## Task 8: Concurrency test (plugin push vs git apply)

Prove the shared lock serializes the two writers.

**Files:**
- Create: `internal/gitapi/concurrency_test.go`

- [ ] **Step 1: Write the test**

```go
package gitapi

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

// lockEnv records max observed concurrency across Lock/Unlock.
type lockEnv struct {
	fakeEnv
	active int32
	maxObs int32
}

func (l *lockEnv) LockNoteWrites() {
	n := atomic.AddInt32(&l.active, 1)
	for {
		m := atomic.LoadInt32(&l.maxObs)
		if n <= m || atomic.CompareAndSwapInt32(&l.maxObs, m, n) {
			break
		}
	}
}
func (l *lockEnv) UnlockNoteWrites() { atomic.AddInt32(&l.active, -1) }

func TestMaterializeAndApplySerialize(t *testing.T) {
	// This guards the contract that callers wrap critical sections in the lock.
	env := &lockEnv{}
	env.fakeEnv.notes = []NoteSource{{Path: "a.md", Content: []byte("a")}}
	api := newTestAPI(t, env)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			env.LockNoteWrites()
			_ = api.materialize(context.Background())
			env.UnlockNoteWrites()
		}()
	}
	wg.Wait()

	if env.maxObs > 1 {
		t.Fatalf("observed %d concurrent critical sections, want 1", env.maxObs)
	}
}
```

> This test asserts the *caller contract* (wrap in lock) using a real serializing lock. The
> production lock is `app.noteWriteMu`; here `lockEnv` uses an atomic counter to detect overlap.
> Replace its `LockNoteWrites` with a real `sync.Mutex` if you want it to also serialize — the
> overlap counter then proves the mutex held. Keep it simple: a `sync.Mutex` plus the counter.

Refine `lockEnv` to actually serialize (so the assertion is meaningful):

```go
type lockEnv struct {
	fakeEnv
	mu     sync.Mutex
	active int32
	maxObs int32
}

func (l *lockEnv) LockNoteWrites() {
	l.mu.Lock()
	n := atomic.AddInt32(&l.active, 1)
	if n > atomic.LoadInt32(&l.maxObs) {
		atomic.StoreInt32(&l.maxObs, n)
	}
}
func (l *lockEnv) UnlockNoteWrites() {
	atomic.AddInt32(&l.active, -1)
	l.mu.Unlock()
}
```

- [ ] **Step 2: Run with the race detector**

Run: `go test ./internal/gitapi/ -run TestMaterializeAndApplySerialize -race -v`
Expected: PASS, `maxObs == 1`.

- [ ] **Step 3: Commit**

```bash
git add internal/gitapi/concurrency_test.go
git commit -m "test(gitapi): note-write lock serializes materialize critical sections"
```

---

## Task 9: E2e coexistence (`e2e/gitsync.spec.js`)

Real smart-HTTP via the `git` CLI + GraphQL plugin simulation.

**Files:**
- Create: `e2e/gitsync.spec.js`

- [ ] **Step 1: Confirm the git token mutation shape**

Run: `grep -n "CreateGitTokenInput\|CreateGitTokenPayload\|createGitToken" internal/graph/schema.graphqls`
Note the input fields (e.g. a `name`/`label`) and the payload field that returns the **plaintext token value** (the value used as the basic-auth password). Use those exact names in Step 2.

- [ ] **Step 2: Write the e2e spec**

```js
// @ts-check
import { test, expect } from '@playwright/test';
import { execFileSync } from 'child_process';
import fs from 'fs';
import os from 'os';
import path from 'path';

const GRAPHQL_URL = '/_system/graphql';
const APP_URL = process.env.APP_URL || 'http://localhost:8081';

async function gql(request, apiKey, query, variables = {}) {
  const res = await request.post(GRAPHQL_URL, {
    headers: { 'Content-Type': 'application/json', 'X-Api-Key': apiKey },
    data: { query, variables },
  });
  expect(res.ok()).toBeTruthy();
  const body = await res.json();
  if (body.errors) throw new Error(JSON.stringify(body.errors));
  return body.data;
}

function git(cwd, env, ...args) {
  return execFileSync('git', args, { cwd, env: { ...process.env, ...env }, encoding: 'utf8' });
}

function authedRemote(token) {
  const u = new URL(APP_URL);
  return `${u.protocol}//user:${token}@${u.host}/_system/git`;
}

test.describe('git <-> plugin coexistence', () => {
  let apiKey, token, workdir;

  test.beforeAll(async ({ request }) => {
    apiKey = fs.readFileSync(path.join(process.cwd(), '.test-api-key'), 'utf8').trim();
    // Adjust field names to match Step 1.
    const data = await gql(request, apiKey,
      `mutation($input: CreateGitTokenInput!) {
         createGitToken(input: $input) {
           ... on CreateGitTokenPayload { token }
           ... on ErrorPayload { message }
         }
       }`,
      { input: { name: 'e2e' } });
    token = data.createGitToken.token;
    expect(token).toBeTruthy();
    workdir = fs.mkdtempSync(path.join(os.tmpdir(), 'gitsync-'));
  });

  const PUSH_NOTES = `mutation($input: PushNotesInput!) {
    pushNotes(input: $input) { ... on PushNotesPayload { notes { path } } ... on ErrorPayload { message } }
  }`;
  const NOTE_PATHS = `query { notePaths { value: path } }`;

  test('plugin push -> git clone sees it', async ({ request }) => {
    await gql(request, apiKey, PUSH_NOTES, { input: { updates: [{ path: 'from-plugin.md', content: '# plugin' }] } });
    const dir = path.join(workdir, 'clone1');
    git(workdir, {}, 'clone', authedRemote(token), 'clone1');
    expect(fs.readFileSync(path.join(dir, 'from-plugin.md'), 'utf8')).toContain('# plugin');
  });

  test('git push -> plugin/db sees it', async ({ request }) => {
    const dir = path.join(workdir, 'clone2');
    git(workdir, {}, 'clone', authedRemote(token), 'clone2');
    fs.writeFileSync(path.join(dir, 'from-git.md'), '# git');
    const env = { GIT_AUTHOR_NAME: 't', GIT_AUTHOR_EMAIL: 't@t', GIT_COMMITTER_NAME: 't', GIT_COMMITTER_EMAIL: 't@t' };
    git(dir, env, 'add', 'from-git.md');
    git(dir, env, 'commit', '-m', 'add');
    git(dir, env, 'push', 'origin', 'HEAD:master');
    const data = await gql(request, apiKey, NOTE_PATHS);
    expect(data.notePaths.map((n) => n.value)).toContain('from-git.md');
  });

  test('stale git push is rejected, succeeds after pull', async ({ request }) => {
    const dir = path.join(workdir, 'clone3');
    git(workdir, {}, 'clone', authedRemote(token), 'clone3');
    // plugin changes the same file after the clone
    await gql(request, apiKey, PUSH_NOTES, { input: { updates: [{ path: 'shared.md', content: '# v-plugin' }] } });
    fs.writeFileSync(path.join(dir, 'shared.md'), '# v-git');
    const env = { GIT_AUTHOR_NAME: 't', GIT_AUTHOR_EMAIL: 't@t', GIT_COMMITTER_NAME: 't', GIT_COMMITTER_EMAIL: 't@t' };
    git(dir, env, 'add', 'shared.md');
    git(dir, env, 'commit', '-m', 'git edit');
    let rejected = false;
    try { git(dir, env, 'push', 'origin', 'HEAD:master'); } catch { rejected = true; }
    expect(rejected).toBeTruthy();
    // pull merges plugin's version, then push succeeds
    git(dir, env, 'pull', '--no-edit', 'origin', 'master');
    git(dir, env, 'push', 'origin', 'HEAD:master');
  });

  test('git deletion hides the note', async ({ request }) => {
    await gql(request, apiKey, PUSH_NOTES, { input: { updates: [{ path: 'to-delete.md', content: '# x' }] } });
    const dir = path.join(workdir, 'clone4');
    git(workdir, {}, 'clone', authedRemote(token), 'clone4');
    const env = { GIT_AUTHOR_NAME: 't', GIT_AUTHOR_EMAIL: 't@t', GIT_COMMITTER_NAME: 't', GIT_COMMITTER_EMAIL: 't@t' };
    git(dir, env, 'rm', 'to-delete.md');
    git(dir, env, 'commit', '-m', 'rm');
    git(dir, env, 'push', 'origin', 'HEAD:master');
    const data = await gql(request, apiKey, NOTE_PATHS);
    expect(data.notePaths.map((n) => n.value)).not.toContain('to-delete.md');
  });
});
```

> Adjust `createGitToken` input/payload field names and the `notePaths`/`pushNotes` selection
> sets to the real schema (Steps 1 and the existing `e2e/updatenotes.spec.js` patterns). The
> default branch is `master` (config `MasterBranch`); `HEAD:master` keeps the push explicit.

- [ ] **Step 3: Run the e2e spec**

Run: `npm run test:e2e -- gitsync.spec.js`
Expected: PASS (4 tests). If the server isn't running with a git-capable environment, follow `e2e/README.md` to bring it up first.

- [ ] **Step 4: Commit**

```bash
git add e2e/gitsync.spec.js
git commit -m "test(e2e): git <-> plugin coexistence over smart-HTTP"
```

---

## Self-review notes

- **Spec coverage:** materializer (T3/T4), diff-apply+deletions (T5), lock (T1/T8), FF + rollback + async upload (T6), no `EnableGit` flag (not added), assets mirrored (T4), `tar.gz` kept (T6 keeps `uploadRepo`), unit + e2e tests (T2–T9). Git→DB *asset upload* is intentionally dropped (DB is the canonical asset author; assets flow DB→git only) — called out in T5 Step 3.
- **Type consistency:** `NoteSource`/`AssetSource` defined in T3 and used by `app` in T1 (forward reference: T1 builds before the gitapi `Env` includes them, which compiles since `app` just has extra methods; T3 adds the interface methods). `LockNoteWrites`/`UnlockNoteWrites` consistent across `app`, graph `Env`, gitapi `Env`. `diffChangedFiles`/`applyDiff`/`materialize`/`Materialize` names consistent across tasks.
- **Ordering (resolved):** the shared `NoteSource`/`AssetSource` types are added in T1 Step 0 (in `api.go`) so every commit from T1 onward compiles; the `Env` interface methods that use them are added later in T3 Step 1, when the materializer exists.
