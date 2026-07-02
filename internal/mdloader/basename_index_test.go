package mdloader_test

import (
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"

	"github.com/stretchr/testify/require"
)

// TestBasenameIndexDeterministic proves that BasenameMap candidate order does
// not depend on Go map iteration order: for a bilingual en/X + ru/X pair
// (equal path depth) the slice is sorted lexicographically by path, so a bare
// [[X]] link resolves to the same note on every load.
func TestBasenameIndexDeterministic(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{{
		Path:    "index.md",
		Content: []byte(`Hello [[Topic]]`),
	}, {
		Path:    "en/Topic.md",
		Content: []byte(`English topic.`),
	}, {
		Path:    "ru/Topic.md",
		Content: []byte(`Русская тема.`),
	}}

	for i := 0; i < 20; i++ {
		pages, err := mdloader.Load(mdloader.Options{
			Sources: sourceFiles,
			Log:     &log,
		})
		require.NoError(t, err)

		candidates := pages.BasenameMap["topic"]
		require.Len(t, candidates, 2)
		require.Equal(t, "en/Topic.md", candidates[0].Path,
			"equal-depth candidates must be sorted lexicographically by path")
		require.Equal(t, "ru/Topic.md", candidates[1].Path)

		require.Equal(t, "/en/topic", pages.PathMap["index.md"].ResolvedLinks["Topic"],
			"bare [[Topic]] must resolve identically on every load")
	}
}
