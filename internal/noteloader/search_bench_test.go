package noteloader

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

func docsVaultNotes(b *testing.B) *model.NoteViews {
	b.Helper()
	root := filepath.Join("..", "..", "docs")
	var list []*model.NoteView
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(root, path)
		title := strings.TrimSuffix(filepath.Base(path), ".md")
		list = append(list, createNoteWithAST(int64(len(list)+1), "/"+rel, title, content))
		return nil
	})
	if err != nil {
		b.Fatal(err)
	}
	return notesOf(list...)
}

func BenchmarkSearch(b *testing.B) {
	queries := []string{"search", "поиск", "telegram bot", "шаблон заметки", "subscription", "frontmatter"}
	notes := docsVaultNotes(b)

	for _, mode := range []string{"mem", "disk"} {
		b.Run(mode, func(b *testing.B) {
			loader := &Loader{log: &logger.TestLogger{}, version: "bench"}
			if mode == "disk" {
				loader.searchIndexPath = b.TempDir()
			}
			index, err := loader.buildSearchIndex(notes)
			if err != nil {
				b.Fatal(err)
			}
			defer index.Close()
			loader.searchIndex = index
			loader.nvs = notes

			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				_, searchErr := loader.Search(queries[i%len(queries)], 0, 10)
				if searchErr != nil {
					b.Fatal(searchErr)
				}
			}
		})
	}
}
