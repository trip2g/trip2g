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
