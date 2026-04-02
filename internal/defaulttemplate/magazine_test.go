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
	nvs.PathMap["blog/post2. Telegram.md"] = &model.NoteView{
		Path:      "blog/post2. Telegram.md",
		Title:     "Post 2 Telegram",
		Permalink: "/blog/post2_telegram",
		CreatedAt: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
		RawMeta: map[string]interface{}{
			"telegram_publish_at": "2024-01-20",
		},
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
	// All non-system notes except index.md itself: blog/post1, blog/post2, blog/post2.Telegram, drafts/draft1, about
	require.Len(t, items, 5)
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
	// Should exclude drafts/draft1.md, leaving: blog/post1, blog/post2, blog/post2.Telegram, about
	require.Len(t, items, 4)
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
	// include only blog/*.md (post1, post2, post2.Telegram), then exclude blog/post1.md → post2 + post2.Telegram
	require.Len(t, items, 2)
	for _, item := range items {
		require.NotEqual(t, "Post 1", item.Note.Title())
	}
}

func TestMagazineExcludeFiles_DefaultEmpty(t *testing.T) {
	ctx := &Ctx{}
	require.Empty(t, ctx.MagazineExcludeFiles())
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

func TestMagazineExcludeFiles_TelegramGlob(t *testing.T) {
	nvs := createMagazineTestNVS()
	indexNote := nvs.ByPath("index.md")
	indexNote.Unwrap().RawMeta["magazine_exclude_files"] = "**/*Telegram.md"

	ctx := &Ctx{
		Note:  indexNote,
		Notes: nvs,
	}

	items := ctx.MagazineItems()
	// Should exclude "blog/post2. Telegram.md", leaving: blog/post1, blog/post2, drafts/draft1, about
	require.Len(t, items, 4)
	for _, item := range items {
		require.NotContains(t, item.Note.Path(), "Telegram")
	}
}

func TestMagazineExcludeProperty_NoExclude(t *testing.T) {
	ctx := &Ctx{}
	require.Empty(t, ctx.MagazineExcludeProperty())
}

func TestMagazineExcludeProperty_ReturnsValue(t *testing.T) {
	nvs := createMagazineTestNVS()
	indexNote := nvs.ByPath("index.md")
	indexNote.Unwrap().RawMeta["magazine_exclude_property"] = "telegram_publish_at"

	ctx := &Ctx{
		Note: indexNote,
	}
	require.Equal(t, "telegram_publish_at", ctx.MagazineExcludeProperty())
}

func TestMagazineExcludeProperty_ExcludesByProperty(t *testing.T) {
	nvs := createMagazineTestNVS()
	indexNote := nvs.ByPath("index.md")
	indexNote.Unwrap().RawMeta["magazine_exclude_property"] = "telegram_publish_at"

	ctx := &Ctx{
		Note:  indexNote,
		Notes: nvs,
	}

	items := ctx.MagazineItems()
	// Should exclude blog/post2.Telegram (has telegram_publish_at), leaving: blog/post1, blog/post2, drafts/draft1, about
	require.Len(t, items, 4)
	for _, item := range items {
		require.NotEqual(t, "Post 2 Telegram", item.Note.Title())
	}
}

func TestMagazineExcludeProperty_CombinedWithIncludeProperty(t *testing.T) {
	nvs := createMagazineTestNVS()
	indexNote := nvs.ByPath("index.md")
	// Add "published" to some notes
	nvs.ByPath("blog/post1.md").Unwrap().RawMeta = map[string]interface{}{"published": true}
	nvs.ByPath("blog/post2.md").Unwrap().RawMeta = map[string]interface{}{"published": true}
	// post2.Telegram has both published and telegram_publish_at
	nvs.ByPath("blog/post2. Telegram.md").Unwrap().RawMeta["published"] = true

	indexNote.Unwrap().RawMeta["magazine_include_property"] = "published"
	indexNote.Unwrap().RawMeta["magazine_exclude_property"] = "telegram_publish_at"

	ctx := &Ctx{
		Note:  indexNote,
		Notes: nvs,
	}

	items := ctx.MagazineItems()
	// Include only published (post1, post2, post2.Telegram), then exclude telegram_publish_at (post2.Telegram) → post1, post2
	require.Len(t, items, 2)
	for _, item := range items {
		require.NotEqual(t, "Post 2 Telegram", item.Note.Title())
	}
}
