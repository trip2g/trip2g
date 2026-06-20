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
