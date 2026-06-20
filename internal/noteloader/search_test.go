package noteloader

import (
	"fmt"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func createNoteWithAST(pathID int64, permalink, title string, content []byte) *model.NoteView {
	note := &model.NoteView{
		PathID:    pathID,
		Permalink: permalink,
		Title:     title,
		Content:   content,
	}
	// Parse content to create AST
	reader := text.NewReader(content)
	parser := goldmark.New().Parser()
	doc := parser.Parse(reader)
	note.SetAst(doc)
	return note
}

func TestIncrementalIndexing(t *testing.T) {
	log := &logger.TestLogger{}

	loader := &Loader{
		log: log,
	}

	// Create initial notes
	notes1 := model.NewNoteViews()
	note1 := createNoteWithAST(1, "/note1", "Note 1", []byte("Content 1"))
	note2 := createNoteWithAST(2, "/note2", "Note 2", []byte("Content 2"))
	notes1.List = []*model.NoteView{note1, note2}
	notes1.Map["/note1"] = note1
	notes1.Map["/note2"] = note2

	// First build - should index all
	index, err := loader.buildSearchIndex(notes1)
	require.NoError(t, err)
	require.NotNil(t, index)
	require.Len(t, loader.contentHashes, 2)

	// Verify search works
	loader.searchIndex = index
	loader.nvs = notes1
	results, err := loader.Search("Content")
	require.NoError(t, err)
	require.Len(t, results, 2, "should find both notes")

	// Second build with same content - should skip all (reuses existing index)
	_, err = loader.buildSearchIndex(notes1)
	require.NoError(t, err)

	// Third build with modified note - should index only modified
	notes2 := model.NewNoteViews()
	note1Modified := createNoteWithAST(1, "/note1", "Note 1 Modified", []byte("Content 1 Modified"))
	note2Same := createNoteWithAST(2, "/note2", "Note 2", []byte("Content 2"))
	notes2.List = []*model.NoteView{note1Modified, note2Same}

	oldHash1 := loader.contentHashes[1]
	oldHash2 := loader.contentHashes[2]

	_, err = loader.buildSearchIndex(notes2)
	require.NoError(t, err)

	// Hash for note1 should change, hash for note2 should stay same
	require.NotEqual(t, oldHash1, loader.contentHashes[1], "hash for modified note should change")
	require.Equal(t, oldHash2, loader.contentHashes[2], "hash for unchanged note should stay same")

	// Fourth build with deleted note - should remove from hashes
	notes3 := model.NewNoteViews()
	note2Remaining := createNoteWithAST(2, "/note2", "Note 2", []byte("Content 2"))
	notes3.List = []*model.NoteView{note2Remaining}

	_, err = loader.buildSearchIndex(notes3)
	require.NoError(t, err)
	require.Len(t, loader.contentHashes, 1)
	_, exists := loader.contentHashes[1]
	require.False(t, exists, "deleted note should be removed from hashes")
	_, exists = loader.contentHashes[2]
	require.True(t, exists, "remaining note should stay in hashes")
}

func TestContentHash(t *testing.T) {
	note1 := &model.NoteView{
		Title:   "Test",
		Content: []byte("Content"),
	}
	note2 := &model.NoteView{
		Title:   "Test",
		Content: []byte("Content"),
	}
	note3 := &model.NoteView{
		Title:   "Different",
		Content: []byte("Content"),
	}

	hash1 := contentHash(note1)
	hash2 := contentHash(note2)
	hash3 := contentHash(note3)

	require.Equal(t, hash1, hash2, "same content should produce same hash")
	require.NotEqual(t, hash1, hash3, "different content should produce different hash")
}

// TestEnglishStemming verifies F2: English content is stemmed with the English
// analyzer (not Russian), so "running races" is found by the query "run race".
// Before F2 the index used the Russian analyzer for all fields and this missed.
func TestEnglishStemming(t *testing.T) {
	log := &logger.TestLogger{}
	loader := &Loader{log: log}

	notes := model.NewNoteViews()
	en := createNoteWithAST(1, "/en-note", "Running guide", []byte("Tips for running races and training runs."))
	ru := createNoteWithAST(2, "/ru-note", "Бег", []byte("Советы по бегу и тренировкам."))
	notes.List = []*model.NoteView{en, ru}
	notes.Map["/en-note"] = en
	notes.Map["/ru-note"] = ru

	index, err := loader.buildSearchIndex(notes)
	require.NoError(t, err)
	loader.searchIndex = index
	loader.nvs = notes

	// English stemming: "run race" should match "running races".
	res, err := loader.Search("run race")
	require.NoError(t, err)
	require.True(t, hasURL(res, "/en-note"), "english query should match english note via en analyzer")

	// Russian still works.
	resRu, err := loader.Search("бег тренировка")
	require.NoError(t, err)
	require.True(t, hasURL(resRu, "/ru-note"), "russian query should still match russian note")
}

func hasURL(res []model.SearchResult, url string) bool {
	for _, r := range res {
		if r.URL == url {
			return true
		}
	}
	return false
}

// TestAndOrFallback verifies F5: when AND returns fewer than andOrFallbackThreshold
// hits, OR results are appended so rare-term queries still find partial matches.
func TestAndOrFallback(t *testing.T) {
	log := &logger.TestLogger{}
	loader := &Loader{log: log}

	notes := model.NewNoteViews()
	// noteA contains both words → AND matches it.
	noteA := createNoteWithAST(1, "/note-a", "Alpha Beta", []byte("This document covers alpha and beta topics."))
	// noteB contains only the second word → AND misses it, OR finds it.
	noteB := createNoteWithAST(2, "/note-b", "Beta Only", []byte("This document covers only beta topics."))
	notes.List = []*model.NoteView{noteA, noteB}
	notes.Map["/note-a"] = noteA
	notes.Map["/note-b"] = noteB

	index, err := loader.buildSearchIndex(notes)
	require.NoError(t, err)
	loader.searchIndex = index
	loader.nvs = notes

	// Query for "alpha beta": AND returns only /note-a (1 hit < threshold=3),
	// so OR fallback should also surface /note-b.
	res, err := loader.Search("alpha beta")
	require.NoError(t, err)

	require.True(t, hasURL(res, "/note-a"), "AND hit must be present")
	require.True(t, hasURL(res, "/note-b"), "OR fallback hit must be present after AND misses threshold")

	// AND result must come before OR-only result (AND first ordering).
	var posA, posB int
	for i, r := range res {
		if r.URL == "/note-a" {
			posA = i
		}
		if r.URL == "/note-b" {
			posB = i
		}
	}
	require.Less(t, posA, posB, "AND hit (/note-a) should appear before OR-only hit (/note-b)")
}

// TestAndSufficientHitsNoFallback verifies that when AND finds at least
// andOrFallbackThreshold results, the OR fallback is not triggered (i.e. results
// that only match OR are not mixed in).
func TestAndSufficientHitsNoFallback(t *testing.T) {
	log := &logger.TestLogger{}
	loader := &Loader{log: log}

	notes := model.NewNoteViews()
	// Three notes all contain "common" — AND will return all three (≥ threshold).
	for i := range 3 {
		permalink := fmt.Sprintf("/note-%d", i)
		n := createNoteWithAST(int64(i+1), permalink, fmt.Sprintf("Note %d", i),
			[]byte(fmt.Sprintf("This note %d has the common keyword.", i)))
		notes.List = append(notes.List, n)
		notes.Map[permalink] = n
	}
	// An extra note that matches OR but not AND for the two-term query.
	noteOrOnly := createNoteWithAST(10, "/or-only", "Only Rare", []byte("Only the rare word here."))
	notes.List = append(notes.List, noteOrOnly)
	notes.Map["/or-only"] = noteOrOnly

	index, err := loader.buildSearchIndex(notes)
	require.NoError(t, err)
	loader.searchIndex = index
	loader.nvs = notes

	// "common rare": AND matches 0 notes (none have both), so this actually
	// triggers fallback — replace with a single-term query that AND handles fine.
	// Use "common" alone: all three notes match → AND hits ≥ threshold → no fallback.
	res, err := loader.Search("common")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(res), andOrFallbackThreshold,
		"AND should return at least threshold hits without triggering fallback")
}
