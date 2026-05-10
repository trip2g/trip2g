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
			note := wrapper.ByPath(tt.path)
			require.NotNil(t, note)
			require.Equal(t, tt.expected, note.Title())
		})
	}
}

func TestNVS_ByPath_NotFound(t *testing.T) {
	nvs := model.NewNoteViews()
	wrapper := templateviews.NewNVS(nvs, "live")

	note := wrapper.ByPath("/nonexistent.md")
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
	target := wrapper.ByPath("docs/article.md")

	backlinks := wrapper.BackLinks(target)
	require.Len(t, backlinks, 1)
	require.Equal(t, "docs/intro.md", backlinks[0].Path())
}
