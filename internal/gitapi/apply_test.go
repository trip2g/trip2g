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
	_ = commitFile(t, api, "a.md", "one")
	old := commitFile(t, api, "b.md", "two")
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
}

func TestApplyReceivedRollsBackRef(t *testing.T) {
	env := &fakeEnv{pushErr: true}
	api := newTestAPI(t, env)
	old := commitFile(t, api, "a.md", "one")    // ref at commit1
	newRev := commitFile(t, api, "a.md", "two") // ref now at commit2
	if newRev == old {
		t.Fatal("precondition: revs should differ")
	}
	if err := api.applyReceived(old, newRev); err == nil {
		t.Fatal("expected apply error")
	}
	got := gitOut(t, api.config.RepoPath, "rev-parse", api.config.MasterBranch)
	if got != old {
		t.Fatalf("ref = %s, want rolled back to %s", got, old)
	}
}
