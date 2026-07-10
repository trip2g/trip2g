package noteloader

import (
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

// Toggling search:false is an access decision even when title/content did not
// change. An incremental rebuild must remove the document already in Bleve.
func TestIncrementalIndexing_RemovesNoteWhenSearchDisabled(t *testing.T) {
	loader := &Loader{log: &logger.TestLogger{}}
	const canary = "privacysearchcanary"

	visible := createNoteWithAST(42, "/private-later", "Initially searchable", []byte(canary))
	before := model.NewNoteViews()
	before.List = []*model.NoteView{visible}
	before.Map[visible.Permalink] = visible

	index, err := loader.buildSearchIndex(before)
	require.NoError(t, err)
	loader.searchIndex = index
	loader.nvs = before

	results, err := loader.Search(canary)
	require.NoError(t, err)
	require.Len(t, results, 1, "precondition: note must initially be indexed")

	hidden := createNoteWithAST(42, "/private-later", "Initially searchable", []byte(canary))
	hidden.ExcludeSearch = true
	after := model.NewNoteViews()
	after.List = []*model.NoteView{hidden}
	after.Map[hidden.Permalink] = hidden

	index, err = loader.buildSearchIndex(after)
	require.NoError(t, err)
	loader.searchIndex = index
	loader.nvs = after

	results, err = loader.Search(canary)
	require.NoError(t, err)
	require.Empty(t, results, "search:false must remove an already-indexed document")
}

func hasURL(res []model.SearchResult, url string) bool {
	for _, r := range res {
		if r.URL == url {
			return true
		}
	}
	return false
}
