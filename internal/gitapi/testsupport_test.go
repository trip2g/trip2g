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
	api := &API{config: cfg, env: env, logger: &logger.DummyLogger{}, ctx: context.Background()}
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

func (f *fakeEnv) Logger() logger.Logger { return &logger.DummyLogger{} }
func (f *fakeEnv) LockNoteWrites()       {}
func (f *fakeEnv) UnlockNoteWrites()     {}

func (f *fakeEnv) LatestNoteSources(context.Context) ([]NoteSource, error)   { return f.notes, nil }
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
func (f *fakeEnv) PutPrivateObject(context.Context, io.Reader, string) error { return nil }
func (f *fakeEnv) GetPrivateObject(context.Context, string) (io.ReadCloser, error) {
	return nil, os.ErrNotExist
}
func (f *fakeEnv) PrivateObjectExists(context.Context, string) (bool, error) { return false, nil }
func (f *fakeEnv) GitTokenByValueSha256(context.Context, string) (db.GitToken, error) {
	return db.GitToken{}, nil
}
