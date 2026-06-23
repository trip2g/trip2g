package gitapi

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"trip2g/internal/db"
	"trip2g/internal/logger"
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

// errReader is an io.ReadCloser whose Read always returns an error.
type errReader struct{ err error }

func (e *errReader) Read(_ []byte) (int, error) { return 0, e.err }
func (e *errReader) Close() error               { return nil }

// TestMaterializeCleansUpOrphansOnFailure reproduces the production incident:
// materialize() writes loose git objects via hash-object -w before it fails.
// Those orphaned objects must not persist after the error return — otherwise a
// disk-filling "ENOSPC" retry leaves ~1.7G of dangling blobs behind (exactly
// the incident pattern).
func TestMaterializeCleansUpOrphansOnFailure(t *testing.T) {
	enospc := errors.New("write /: no space left on device")

	env := &EnvMock{
		LoggerFunc:              func() logger.Logger { return &logger.DummyLogger{} },
		LockNoteWritesFunc:      func() {},
		UnlockNoteWritesFunc:    func() {},
		PrivateObjectExistsFunc: func(_ context.Context, _ string) (bool, error) { return false, nil },
		LatestNoteSourcesFunc: func(_ context.Context) ([]NoteSource, error) {
			// One note that materializes successfully (its blob will be written).
			return []NoteSource{{Path: "note.md", Content: []byte("hello world\n")}}, nil
		},
		LatestAssetSourcesFunc: func(_ context.Context) ([]AssetSource, error) {
			return []AssetSource{
				{
					AbsolutePath: "/assets/big.bin",
					Asset:        db.NoteAsset{AbsolutePath: "/assets/big.bin", FileName: "big.bin"},
				},
			}, nil
		},
		ReadAssetObjectFunc: func(_ context.Context, _ db.NoteAsset) (io.ReadCloser, error) {
			// ReadAssetObject succeeds (object exists), but reading its body fails
			// — exactly what happens when the disk fills mid-stream.
			return &errReader{err: enospc}, nil
		},
	}

	api := newTestAPI(t, env)

	// Drive materialize: note blob is written, then asset read fails → error.
	err := api.materialize(context.Background())
	if err == nil {
		t.Fatal("expected materialize to return an error, got nil")
	}

	// (a) Confirm the error propagated.
	t.Logf("materialize returned (expected): %v", err)

	// (b) No unreachable objects should remain.
	// We run fsck and filter for lines that indicate actual dangling/unreachable
	// objects; "notice:" lines (e.g. unborn branch) are informational and safe.
	cmd := exec.Command("git", "--git-dir", api.config.RepoPath,
		"fsck", "--unreachable", "--no-reflogs")
	var fskOut strings.Builder
	cmd.Stdout = &fskOut
	cmd.Stderr = &fskOut
	_ = cmd.Run() // fsck exits 0 even when printing warnings; we care about output
	var orphanLines []string
	for _, line := range strings.Split(fskOut.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "notice:") {
			continue
		}
		orphanLines = append(orphanLines, line)
	}
	if len(orphanLines) > 0 {
		t.Fatalf("dangling/unreachable objects after failed materialize:\n%s",
			strings.Join(orphanLines, "\n"))
	}

	// (c) No git temp object files should remain.
	tmpObjs, _ := filepath.Glob(filepath.Join(api.config.RepoPath, "objects", "*", "tmp_obj_*"))
	tmpObjs2, _ := filepath.Glob(filepath.Join(api.config.RepoPath, "objects", "tmp_obj_*"))
	tmpObjs = append(tmpObjs, tmpObjs2...)
	if len(tmpObjs) > 0 {
		t.Fatalf("leftover git temp object files after failed materialize: %v", tmpObjs)
	}
}
