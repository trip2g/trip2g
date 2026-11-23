package gitapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"trip2g/internal/graph/model"

	"github.com/stretchr/testify/require"
)

// commitTree writes and removes files in one commit and returns the new HEAD.
func commitTree(t *testing.T, api *API, writes map[string]string, removes ...string) string {
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
		require.NoError(t, cmd.Run(), "git %v", args)
	}
	if api.isRefExists("refs/heads/" + api.config.MasterBranch) {
		run("checkout", api.config.MasterBranch)
	}
	for _, name := range removes {
		run("rm", "-q", name)
	}
	for name, content := range writes {
		full := filepath.Join(work, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
		run("add", name)
	}
	run("commit", "-q", "-m", "x")
	return gitOut(t, api.config.RepoPath, "rev-parse", api.config.MasterBranch)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

const pngBytes = "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\xff\xfe\x00binary"

func TestApplyStoresPushedAsset(t *testing.T) {
	env := &fakeEnv{
		pushedNotes: []model.PushedNote{
			{ID: 7, Path: "note.md", Assets: []model.PushedNoteAsset{{Path: "img.png"}}},
		},
	}
	api := newTestAPI(t, env)
	newRev := commitTree(t, api, map[string]string{
		"note.md": "![[img.png]]\n",
		"img.png": pngBytes,
	})

	require.NoError(t, api.applyDiff(context.Background(), "", newRev))

	require.Equal(t, []string{"note.md"}, env.pushed)
	require.Len(t, env.uploads, 1)
	got := env.uploads[0]
	require.Equal(t, int64(7), got.input.NoteID)
	require.Equal(t, "img.png", got.input.Path)
	require.Equal(t, "img.png", got.input.AbsolutePath)
	require.Equal(t, sha256Hex(pngBytes), got.input.Sha256Hash)
	require.Equal(t, int64(len(pngBytes)), got.input.File.Size)
	require.Equal(t, []byte(pngBytes), got.content)
	require.Equal(t, 1, env.commits)
	require.NotNil(t, env.pushInputs[0].SkipCommit)
	require.True(t, *env.pushInputs[0].SkipCommit)
}

func TestApplyUploadsChangedAssetOfUnchangedNote(t *testing.T) {
	env := &fakeEnv{}
	api := newTestAPI(t, env)
	old := commitTree(t, api, map[string]string{
		"docs/note.md":      "![](./pics/a.png)\n",
		"docs/pics/a.png":   "v1",
		"docs/untouched.md": "x",
	})
	newRev := commitTree(t, api, map[string]string{"docs/pics/a.png": "v2"})

	stored := sha256Hex("v1")
	env.pushedNotes = []model.PushedNote{
		{ID: 3, Path: "docs/note.md", Assets: []model.PushedNoteAsset{
			{Path: "./pics/a.png", Sha256Hash: &stored, AbsolutePath: "docs/pics/a.png"},
		}},
	}

	require.NoError(t, api.applyDiff(context.Background(), old, newRev))

	require.Len(t, env.pushInputs, 1)
	require.Empty(t, env.pushInputs[0].Updates)
	require.Len(t, env.uploads, 1)
	require.Equal(t, int64(3), env.uploads[0].input.NoteID)
	require.Equal(t, "docs/pics/a.png", env.uploads[0].input.AbsolutePath)
	require.Equal(t, sha256Hex("v2"), env.uploads[0].input.Sha256Hash)
	require.Equal(t, 1, env.commits)
}

func TestApplySkipsAssetAlreadyStored(t *testing.T) {
	env := &fakeEnv{}
	api := newTestAPI(t, env)
	old := commitTree(t, api, map[string]string{"note.md": "![[img.png]]\n", "img.png": pngBytes})
	newRev := commitTree(t, api, map[string]string{"note.md": "![[img.png]]\nmore\n"})

	stored := sha256Hex(pngBytes)
	env.pushedNotes = []model.PushedNote{
		{ID: 9, Path: "note.md", Assets: []model.PushedNoteAsset{
			{Path: "img.png", Sha256Hash: &stored, AbsolutePath: "img.png"},
		}},
	}

	require.NoError(t, api.applyDiff(context.Background(), old, newRev))

	require.Equal(t, []string{"note.md"}, env.pushed)
	require.Empty(t, env.uploads)
	require.Equal(t, 1, env.commits)
}

func TestApplyRenamedAssetRelinksUnchangedNote(t *testing.T) {
	env := &fakeEnv{}
	api := newTestAPI(t, env)
	old := commitTree(t, api, map[string]string{"note.md": "![[img.png]]\n", "img.png": pngBytes})
	newRev := commitTree(t, api, map[string]string{"pics/img.png": pngBytes}, "img.png")

	stored := sha256Hex(pngBytes)
	env.pushedNotes = []model.PushedNote{
		{ID: 4, Path: "note.md", Assets: []model.PushedNoteAsset{
			{Path: "img.png", Sha256Hash: &stored, AbsolutePath: "img.png"},
		}},
	}

	require.NoError(t, api.applyDiff(context.Background(), old, newRev))

	require.Len(t, env.uploads, 1)
	require.Equal(t, "img.png", env.uploads[0].input.Path)
	require.Equal(t, "pics/img.png", env.uploads[0].input.AbsolutePath)
	require.Equal(t, []byte(pngBytes), env.uploads[0].content)
}

func TestApplyIgnoresUnresolvedAndUntouchedAssets(t *testing.T) {
	env := &fakeEnv{}
	api := newTestAPI(t, env)
	old := commitTree(t, api, map[string]string{"other.md": "![[o.png]]\n", "o.png": "o"})
	newRev := commitTree(t, api, map[string]string{"note.md": "![[missing.png]]\n"})

	env.pushedNotes = []model.PushedNote{
		{ID: 1, Path: "note.md", Assets: []model.PushedNoteAsset{{Path: "missing.png"}}},
		{ID: 2, Path: "other.md", Assets: []model.PushedNoteAsset{{Path: "o.png"}}},
	}

	require.NoError(t, api.applyDiff(context.Background(), old, newRev))
	require.Empty(t, env.uploads)
}

func TestApplyFailsWhenAssetIsRefused(t *testing.T) {
	env := &fakeEnv{
		pushedNotes: []model.PushedNote{
			{ID: 7, Path: "note.md", Assets: []model.PushedNoteAsset{{Path: "img.png"}}},
		},
		uploadPayload: &model.ErrorPayload{Message: "assets storage limit exceeded"},
	}
	api := newTestAPI(t, env)
	old := commitTree(t, api, map[string]string{"seed.md": "x"})
	newRev := commitTree(t, api, map[string]string{"note.md": "![[img.png]]\n", "img.png": pngBytes})

	err := api.applyReceived(old, newRev)
	require.ErrorContains(t, err, "assets storage limit exceeded")
	require.Equal(t, 0, env.commits)
	require.Equal(t, old, gitOut(t, api.config.RepoPath, "rev-parse", api.config.MasterBranch))
}

func TestResolveAssetPath(t *testing.T) {
	files := map[string]struct{}{
		"root.png":            {},
		"assets/shared.png":   {},
		"docs/local.png":      {},
		"docs/sub/x.png":      {},
		"deep/a/b/far.png":    {},
		"deep/near/far.png":   {},
		"docs/root.png":       {},
		"Mixed/Case.PNG":      {},
		"_layouts/m/hero.png": {},
	}

	tests := []struct {
		name, link, note, want string
	}{
		{"bare name at root wins over note folder", "root.png", "docs/n.md", "root.png"},
		{"bare name in assets folder", "shared.png", "docs/n.md", "assets/shared.png"},
		{"bare name next to the note", "local.png", "docs/n.md", "docs/local.png"},
		{"bare name anywhere, shortest path", "far.png", "n.md", "deep/near/far.png"},
		{"bare name matches case-insensitively", "case.png", "n.md", "Mixed/Case.PNG"},
		{"explicit relative", "./sub/x.png", "docs/n.md", "docs/sub/x.png"},
		{"explicit absolute", "/docs/sub/x.png", "n.md", "docs/sub/x.png"},
		{"vault path", "_layouts/m/hero.png", "_layouts/m/index.html", "_layouts/m/hero.png"},
		{"bare folder path is not note-relative", "sub/x.png", "docs/n.md", ""},
		{"missing", "nope.png", "docs/n.md", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, resolveAssetPath(files, tt.link, tt.note))
		})
	}
}
