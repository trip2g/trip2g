package mdloader_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
)

func TestNameIndex(t *testing.T) {
	log := logger.TestLogger{}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path:    "Plugins/Bookmarks.md",
			Content: []byte("---\naliases:\n  - Plugins/Starred\n  - Звёздочки\n---\nBookmarks body."),
		}, {
			Path:    "Aliases.md",
			Content: []byte("---\ntitle: Псевдонимы\naliases: Aliases\n---\nBody."),
		}, {
			Path:    "Broken.md",
			Content: []byte("---\naliases:\n  - 42\n  - \"\"\n  - Real name\n---\nBody."),
		}},
		Log: &log,
	})
	require.NoError(t, err)

	pathsFor := func(name string) []string {
		var paths []string
		for _, n := range pages.NameMap[model.NormalizeName(name)] {
			paths = append(paths, n.Path)
		}
		return paths
	}

	require.Equal(t, []string{"Plugins/Bookmarks.md"}, pathsFor("plugins/starred"))
	require.Equal(t, []string{"Plugins/Bookmarks.md"}, pathsFor("звездочки"), "ё folds to е")
	require.Equal(t, []string{"Plugins/Bookmarks.md"}, pathsFor("Bookmarks"), "the title is a name too")
	require.Equal(t, []string{"Aliases.md"}, pathsFor("aliases"), "a single-string alias counts")
	require.Equal(t, []string{"Aliases.md"}, pathsFor("  ПСЕВДОНИМЫ "))
	require.Equal(t, []string{"Broken.md"}, pathsFor("real name"), "non-string and empty aliases are skipped")
}

func TestNormalizeName(t *testing.T) {
	require.Equal(t, "елка и дерево", model.NormalizeName("  Ёлка   и\tДерево "))
	// NFD input (й decomposed into и + U+0306) must match its NFC form.
	require.Equal(t, model.NormalizeName("ключ й"), model.NormalizeName("ключ й"))
}
