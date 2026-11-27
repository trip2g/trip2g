package templateviews_test

import (
	"testing"

	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/stretchr/testify/require"
)

func TestNVS_ByPath(t *testing.T) {
	nvs := model.NewNoteViews()

	nvs.PathMap["_sidebar.md"] = &model.NoteView{
		Path:  "_sidebar.md",
		Title: "Sidebar",
	}
	nvs.PathMap["docs/intro.md"] = &model.NoteView{
		Path:  "docs/intro.md",
		Title: "Introduction",
	}

	wrapper := templateviews.NewNVS(nvs, "live")

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"without leading slash", "_sidebar.md", "Sidebar"},
		{"with leading slash", "/_sidebar.md", "Sidebar"},
		{"nested without slash", "docs/intro.md", "Introduction"},
		{"nested with slash", "/docs/intro.md", "Introduction"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := wrapper.NoteByPath(tt.path)
			require.NotNil(t, note)
			require.Equal(t, tt.expected, note.Title())
		})
	}
}

func TestNVS_ByPath_NotFound(t *testing.T) {
	nvs := model.NewNoteViews()
	wrapper := templateviews.NewNVS(nvs, "live")

	note := wrapper.NoteByPath("/nonexistent.md")
	require.Nil(t, note)
}

func TestNVS_ByPath_NilNVS(t *testing.T) {
	var wrapper *templateviews.NVS

	// Should not panic
	require.Nil(t, wrapper)
}

func TestNVS_BackLinks_ExcludesSystemNotes(t *testing.T) {
	nvs := model.NewNoteViews()

	targetNote := &model.NoteView{
		PathID:    1,
		Path:      "docs/article.md",
		Permalink: "/docs/article",
		InLinks: map[string]struct{}{
			"/_footer":    {},
			"/docs/intro": {},
		},
	}
	normalNote := &model.NoteView{
		PathID:    2,
		Path:      "docs/intro.md",
		Permalink: "/docs/intro",
	}
	systemNote := &model.NoteView{
		PathID:    3,
		Path:      "_footer.md",
		Permalink: "/_footer",
	}

	nvs.Map["/docs/article"] = targetNote
	nvs.Map["/docs/intro"] = normalNote
	nvs.Map["/_footer"] = systemNote
	nvs.PathMap["docs/article.md"] = targetNote
	nvs.PathMap["docs/intro.md"] = normalNote
	nvs.PathMap["_footer.md"] = systemNote

	wrapper := templateviews.NewNVS(nvs, "live")
	target := wrapper.NoteByPath("docs/article.md")

	backlinks := wrapper.BackLinks(target)
	require.Len(t, backlinks, 1)
	require.Equal(t, "docs/intro.md", backlinks[0].Path())
}

func TestNVS_BackLinks_SortedByTitleThenPermalink(t *testing.T) {
	nvs := model.NewNoteViews()

	inLinks := map[string]struct{}{}
	notes := []*model.NoteView{
		{PathID: 2, Path: "b.md", Permalink: "/b", Title: "zebra"},
		{PathID: 3, Path: "c.md", Permalink: "/c", Title: "Apple"},
		{PathID: 4, Path: "d.md", Permalink: "/d", Title: "mango"},
		{PathID: 5, Path: "a.md", Permalink: "/a", Title: "mango"},
		{PathID: 6, Path: "e.md", Permalink: "/e", Title: "Яблоко"},
	}
	for _, n := range notes {
		inLinks[n.Permalink] = struct{}{}
		nvs.Map[n.Permalink] = n
		nvs.PathMap[n.Path] = n
	}

	target := &model.NoteView{PathID: 1, Path: "target.md", Permalink: "/target", InLinks: inLinks}
	nvs.Map["/target"] = target
	nvs.PathMap["target.md"] = target

	wrapper := templateviews.NewNVS(nvs, "live")

	for range 20 {
		backlinks := wrapper.BackLinks(wrapper.NoteByPath("target.md"))
		got := make([]string, 0, len(backlinks))
		for _, bl := range backlinks {
			got = append(got, bl.Path())
		}
		require.Equal(t, []string{"c.md", "a.md", "d.md", "b.md", "e.md"}, got)
	}
}
