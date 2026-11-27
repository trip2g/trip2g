package defaulttemplate

import (
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/templateviews"

	"github.com/stretchr/testify/require"
)

// Frontmatter links must resolve like links in note text: Obsidian writes the
// shortest form, e.g. [[_sidebar]] or [[Note|alias]].
func TestResolveNoteRef_FrontmatterWikilinkLikeObsidian(t *testing.T) {
	var srcs []mdloader.SourceFile
	for i, p := range []string{"_sidebar.md", "ru/user/_sidebar.md", "ru/user/page.md", "Site Footer.md"} {
		srcs = append(srcs, mdloader.SourceFile{Path: p, PathID: int64(i + 1), Content: []byte("body")})
	}
	loaded, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &logger.TestLogger{}})
	require.NoError(t, err)
	nvs := templateviews.NewNVS(loaded, "live")

	ctx := &Ctx{Note: nvs.NoteByPath("ru/user/page.md"), Notes: nvs}

	for value, want := range map[string]string{
		"[[ru/user/_sidebar|Menu]]": "ru/user/_sidebar.md",
		"[[Site Footer]]":           "Site Footer.md",
		"ru/user/_sidebar.md":       "ru/user/_sidebar.md",
	} {
		note := ctx.resolveNoteRef(parseContentRef(value))
		require.NotNil(t, note, value)
		require.Equal(t, want, note.Path(), value)
	}
}
