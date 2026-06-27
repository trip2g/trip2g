package templateviews

import (
	"testing"
	"time"

	"trip2g/internal/model"
)

func TestNoteLastEditedByNilWithoutResolver(t *testing.T) {
	note := NewNote(&model.NoteView{Path: "a.md"})
	if got := note.LastEditedBy(); got != nil {
		t.Fatalf("expected nil editor without resolver, got %+v", got)
	}
}

func TestNoteLastEditedByReturnsResolvedEditor(t *testing.T) {
	when := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)
	want := &NoteEditor{
		UserEmail:         "editor@example.com",
		APIKeyDescription: "",
		Client:            "obsidian-plugin/4.2.1",
		CreatedAt:         when,
	}

	note := NewNote(&model.NoteView{Path: "a.md"})
	note.SetLastEditedByResolver(func() *NoteEditor { return want })

	got := note.LastEditedBy()
	if got == nil {
		t.Fatal("expected resolved editor, got nil")
	}
	if got.UserEmail != "editor@example.com" {
		t.Errorf("UserEmail = %q, want %q", got.UserEmail, "editor@example.com")
	}
	if got.APIKeyDescription != "" {
		t.Errorf("APIKeyDescription = %q, want empty", got.APIKeyDescription)
	}
	if got.Client != "obsidian-plugin/4.2.1" {
		t.Errorf("Client = %q, want %q", got.Client, "obsidian-plugin/4.2.1")
	}
	if !got.CreatedAt.Equal(when) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, when)
	}
}

func TestNoteLastEditedByClientEmpty(t *testing.T) {
	want := &NoteEditor{
		UserEmail: "editor@example.com",
		Client:    "", // absent header
	}

	note := NewNote(&model.NoteView{Path: "b.md"})
	note.SetLastEditedByResolver(func() *NoteEditor { return want })

	got := note.LastEditedBy()
	if got == nil {
		t.Fatal("expected resolved editor, got nil")
	}
	if got.Client != "" {
		t.Errorf("Client = %q, want empty for absent header", got.Client)
	}
}
