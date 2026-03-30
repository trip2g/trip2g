package defaulttemplate

import (
	"testing"
	"time"

	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/stretchr/testify/require"
)

func createMagazineTestNVS() *templateviews.NVS {
	nvs := model.NewNoteViews()

	nvs.PathMap["blog/post1.md"] = &model.NoteView{
		Path:      "blog/post1.md",
		Title:     "Post 1",
		Permalink: "/blog/post1",
		CreatedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
	}
	nvs.PathMap["blog/post2.md"] = &model.NoteView{
		Path:      "blog/post2.md",
		Title:     "Post 2",
		Permalink: "/blog/post2",
		CreatedAt: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
	}
	nvs.PathMap["drafts/draft1.md"] = &model.NoteView{
		Path:      "drafts/draft1.md",
		Title:     "Draft 1",
		Permalink: "/drafts/draft1",
		CreatedAt: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	nvs.PathMap["about.md"] = &model.NoteView{
		Path:      "about.md",
		Title:     "About",
		Permalink: "/about",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	nvs.PathMap["index.md"] = &model.NoteView{
		Path:      "index.md",
		Title:     "Index",
		Permalink: "/",
		CreatedAt: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
		RawMeta:   map[string]interface{}{},
	}

	return templateviews.NewNVS(nvs, "live")
}

func TestMagazineExcludeFiles_NoExclude(t *testing.T) {
	nvs := createMagazineTestNVS()
	ctx := &Ctx{
		Note:  nvs.ByPath("index.md"),
		Notes: nvs,
	}

	items := ctx.MagazineItems()
	// All non-system notes except index.md itself: blog/post1, blog/post2, drafts/draft1, about
	require.Len(t, items, 4)
}

func TestMagazineExcludeFiles_ExcludeDrafts(t *testing.T) {
	nvs := createMagazineTestNVS()
	indexNote := nvs.ByPath("index.md")
	indexNote.Unwrap().RawMeta["magazine_exclude_files"] = "drafts/**"

	ctx := &Ctx{
		Note:  indexNote,
		Notes: nvs,
	}

	items := ctx.MagazineItems()
	// Should exclude drafts/draft1.md, leaving: blog/post1, blog/post2, about
	require.Len(t, items, 3)
	for _, item := range items {
		require.NotContains(t, item.Note.Path(), "drafts/")
	}
}

func TestMagazineExcludeFiles_ExcludeMultipleWithInclude(t *testing.T) {
	nvs := createMagazineTestNVS()
	indexNote := nvs.ByPath("index.md")
	indexNote.Unwrap().RawMeta["magazine_include_files"] = "blog/*.md"
	indexNote.Unwrap().RawMeta["magazine_exclude_files"] = "blog/post1.md"

	ctx := &Ctx{
		Note:  indexNote,
		Notes: nvs,
	}

	items := ctx.MagazineItems()
	// include only blog/*.md (post1, post2), then exclude blog/post1.md → only post2
	require.Len(t, items, 1)
	require.Equal(t, "Post 2", items[0].Note.Title())
}

func TestMagazineExcludeFiles_DefaultEmpty(t *testing.T) {
	ctx := &Ctx{}
	require.Equal(t, "", ctx.MagazineExcludeFiles())
}

func TestMagazineExcludeFiles_ReturnsValue(t *testing.T) {
	nvs := createMagazineTestNVS()
	indexNote := nvs.ByPath("index.md")
	indexNote.Unwrap().RawMeta["magazine_exclude_files"] = "drafts/**"

	ctx := &Ctx{
		Note: indexNote,
	}
	require.Equal(t, "drafts/**", ctx.MagazineExcludeFiles())
}
