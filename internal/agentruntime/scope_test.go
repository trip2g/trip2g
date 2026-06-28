package agentruntime

import (
	"context"
	"errors"
	"testing"
)

func TestScopedKB_ReadEnforcement(t *testing.T) {
	kb := newMemKB(map[string]string{
		"subjects/subj1/profile.md": "subj1 profile",
		"subjects/subj2/profile.md": "subj2 profile",
	})
	scoped := NewScopedKB(kb, []string{"subjects/subj1/**"}, nil)

	got, err := scoped.Read(context.Background(), "subjects/subj1/profile.md")
	if err != nil {
		t.Fatalf("in-scope read failed: %v", err)
	}
	if got != "subj1 profile" {
		t.Fatalf("unexpected content: %q", got)
	}

	_, err = scoped.Read(context.Background(), "subjects/subj2/profile.md")
	if !errors.Is(err, ErrReadDenied) {
		t.Fatalf("out-of-scope read should be denied, got %v", err)
	}
}

func TestScopedKB_SearchFiltersOutOfScope(t *testing.T) {
	kb := newMemKB(map[string]string{
		"subjects/subj1/profile.md": "shared keyword",
		"subjects/subj2/profile.md": "shared keyword",
	})
	scoped := NewScopedKB(kb, []string{"subjects/subj1/**"}, nil)

	docs, err := scoped.Search(context.Background(), "shared")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(docs) != 1 || docs[0].Path != "subjects/subj1/profile.md" {
		t.Fatalf("search must return only in-scope docs, got %+v", docs)
	}
}

func TestScopedKB_WriteEnforcement(t *testing.T) {
	kb := newMemKB(nil)
	scoped := NewScopedKB(kb, []string{"subjects/subj1/**"}, []string{"subjects/subj1/**"})

	err := scoped.Write(context.Background(), "subjects/subj1/note.md", "ok")
	if err != nil {
		t.Fatalf("in-scope write failed: %v", err)
	}
	if kb.docs["subjects/subj1/note.md"] != "ok" {
		t.Fatalf("write did not persist")
	}

	err = scoped.Write(context.Background(), "subjects/subj2/note.md", "nope")
	if !errors.Is(err, ErrWriteDenied) {
		t.Fatalf("out-of-scope write should be denied, got %v", err)
	}
}

func TestScopedKB_EmptyWritePatternsIsReadOnly(t *testing.T) {
	kb := newMemKB(nil)
	scoped := NewScopedKB(kb, []string{"subjects/subj1/**"}, nil)

	err := scoped.Write(context.Background(), "subjects/subj1/note.md", "x")
	if !errors.Is(err, ErrWriteDenied) {
		t.Fatalf("empty write patterns must deny all writes, got %v", err)
	}
}
