package defaulttemplate

import (
	"strings"
	"testing"

	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/stretchr/testify/require"
)

// makeNote creates a NoteView with the given path and raw frontmatter meta.
// Permalink is derived by lower-casing the path without the .md extension.
func makeNote(path string, meta map[string]interface{}) *model.NoteView {
	permalink := "/" + strings.TrimSuffix(strings.ToLower(strings.ReplaceAll(path, " ", "_")), ".md")
	return &model.NoteView{
		Path:      path,
		Permalink: permalink,
		RawMeta:   meta,
	}
}

// makeNVS builds a minimal templateviews.NVS from a slice of NoteViews.
func makeNVS(notes []*model.NoteView) *templateviews.NVS {
	nvs := model.NewNoteViews()
	for _, nv := range notes {
		nvs.PathMap[nv.Path] = nv
		nvs.Map[nv.Permalink] = nv
	}
	return templateviews.NewNVS(nvs, "live")
}

func TestResolveLayoutSection_GlobMatch(t *testing.T) {
	headerNote := makeNote("layouts/header.md", map[string]interface{}{
		"header_includes": []interface{}{"blog/**"},
	})
	postNote := makeNote("blog/post1.md", nil)

	nvs := makeNVS([]*model.NoteView{headerNote, postNote})
	ctx := &Ctx{
		Note:  nvs.ByPath("blog/post1.md"),
		Notes: nvs,
		LayoutSections: []model.LayoutSectionEntry{
			{NotePath: "layouts/header.md", Section: "header", Includes: []string{"blog/**"}},
		},
	}

	match := ctx.resolveLayoutSection("header")
	require.NotNil(t, match)
	require.Equal(t, "layouts/header.md", match.NotePath)
}

func TestResolveLayoutSection_GlobNoMatch(t *testing.T) {
	postNote := makeNote("blog/post1.md", nil)
	nvs := makeNVS([]*model.NoteView{postNote})
	ctx := &Ctx{
		Note:  nvs.ByPath("blog/post1.md"),
		Notes: nvs,
		LayoutSections: []model.LayoutSectionEntry{
			{NotePath: "layouts/header.md", Section: "header", Includes: []string{"docs/**"}},
		},
	}

	match := ctx.resolveLayoutSection("header")
	require.Nil(t, match)
}

func TestResolveLayoutSection_ExcludePattern(t *testing.T) {
	postNote := makeNote("blog/draft1.md", nil)
	nvs := makeNVS([]*model.NoteView{postNote})
	ctx := &Ctx{
		Note:  nvs.ByPath("blog/draft1.md"),
		Notes: nvs,
		LayoutSections: []model.LayoutSectionEntry{
			{
				NotePath: "layouts/header.md",
				Section:  "header",
				Includes: []string{"blog/**"},
				Excludes: []string{"blog/draft*.md"},
			},
		},
	}

	match := ctx.resolveLayoutSection("header")
	require.Nil(t, match)
}

func TestResolveLayoutSection_IncludeProperty(t *testing.T) {
	postNote := makeNote("blog/post1.md", map[string]interface{}{
		"published": true,
	})
	unpublishedNote := makeNote("blog/post2.md", nil)

	nvs := makeNVS([]*model.NoteView{postNote, unpublishedNote})

	entry := model.LayoutSectionEntry{
		NotePath:        "layouts/header.md",
		Section:         "header",
		Includes:        []string{"blog/**"},
		IncludeProperty: "published",
	}

	// Note with property matches.
	ctx1 := &Ctx{Note: nvs.ByPath("blog/post1.md"), Notes: nvs, LayoutSections: []model.LayoutSectionEntry{entry}}
	require.NotNil(t, ctx1.resolveLayoutSection("header"))

	// Note without property does not match.
	ctx2 := &Ctx{Note: nvs.ByPath("blog/post2.md"), Notes: nvs, LayoutSections: []model.LayoutSectionEntry{entry}}
	require.Nil(t, ctx2.resolveLayoutSection("header"))
}

func TestResolveLayoutSection_ExcludeProperty(t *testing.T) {
	draftNote := makeNote("blog/draft1.md", map[string]interface{}{
		"draft": true,
	})
	nvs := makeNVS([]*model.NoteView{draftNote})

	ctx := &Ctx{
		Note:  nvs.ByPath("blog/draft1.md"),
		Notes: nvs,
		LayoutSections: []model.LayoutSectionEntry{
			{
				NotePath:        "layouts/header.md",
				Section:         "header",
				Includes:        []string{"blog/**"},
				ExcludeProperty: "draft",
			},
		},
	}

	match := ctx.resolveLayoutSection("header")
	require.Nil(t, match)
}

func TestResolveLayoutSection_HigherPriorityWins(t *testing.T) {
	postNote := makeNote("blog/post1.md", nil)
	nvs := makeNVS([]*model.NoteView{postNote})

	ctx := &Ctx{
		Note:  nvs.ByPath("blog/post1.md"),
		Notes: nvs,
		LayoutSections: []model.LayoutSectionEntry{
			{NotePath: "layouts/generic.md", Section: "header", Includes: []string{"**/*.md"}, Priority: 1},
			{NotePath: "layouts/blog.md", Section: "header", Includes: []string{"blog/**"}, Priority: 10},
		},
	}

	match := ctx.resolveLayoutSection("header")
	require.NotNil(t, match)
	require.Equal(t, "layouts/blog.md", match.NotePath)
}

func TestResolveLayoutSection_PriorityTie_ReturnsOne(t *testing.T) {
	postNote := makeNote("blog/post1.md", nil)
	nvs := makeNVS([]*model.NoteView{postNote})

	ctx := &Ctx{
		Note:  nvs.ByPath("blog/post1.md"),
		Notes: nvs,
		LayoutSections: []model.LayoutSectionEntry{
			{NotePath: "layouts/a.md", Section: "header", Includes: []string{"blog/**"}, Priority: 5},
			{NotePath: "layouts/b.md", Section: "header", Includes: []string{"blog/**"}, Priority: 5},
		},
	}

	match := ctx.resolveLayoutSection("header")
	require.NotNil(t, match)
	// Both are valid — just verify one is returned without panic.
}

func TestHeaderRef_PerNoteOverrideWins(t *testing.T) {
	// Note has explicit header: frontmatter key — it should win over layout sections.
	postNote := makeNote("blog/post1.md", map[string]interface{}{
		"header": "custom_header",
	})
	nvs := makeNVS([]*model.NoteView{postNote})

	ctx := &Ctx{
		Note:  nvs.ByPath("blog/post1.md"),
		Notes: nvs,
		LayoutSections: []model.LayoutSectionEntry{
			{NotePath: "layouts/header.md", Section: "header", Includes: []string{"blog/**"}},
		},
	}

	ref := ctx.HeaderRef()
	// Per-note value takes precedence — not the layout section path.
	require.NotEqual(t, "layouts/header.md", ref.Value)
}

func TestHeaderRef_FallbackToDefaultHeader(t *testing.T) {
	// No layout section match, but _header note exists → fallback used.
	postNote := makeNote("blog/post1.md", nil)
	headerNote := makeNote("_header.md", nil)
	headerNote.Permalink = "/_header"

	nvs := makeNVS([]*model.NoteView{postNote, headerNote})

	ctx := &Ctx{
		Note:  nvs.ByPath("blog/post1.md"),
		Notes: nvs,
		LayoutSections: []model.LayoutSectionEntry{
			// Only matches docs/**, not blog/**
			{NotePath: "layouts/header.md", Section: "header", Includes: []string{"docs/**"}},
		},
	}

	ref := ctx.HeaderRef()
	require.Equal(t, ContentRefWikiLink, ref.Kind)
	require.Equal(t, "_header", ref.Value)
}

func TestResolveLayoutSection_NilNoteReturnsNil(t *testing.T) {
	ctx := &Ctx{
		Note: nil,
		LayoutSections: []model.LayoutSectionEntry{
			{NotePath: "layouts/header.md", Section: "header", Includes: []string{"**/*.md"}},
		},
	}

	match := ctx.resolveLayoutSection("header")
	require.Nil(t, match)
}
