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
