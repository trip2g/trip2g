package templateviews_test

import (
	"testing"
	"time"

	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/stretchr/testify/require"
)

func createTestNVS() *templateviews.NVS {
	nvs := model.NewNoteViews()

	// Blog posts
	nvs.PathMap["blog/first.md"] = &model.NoteView{
		Path:      "blog/first.md",
		Title:     "Zebra Post",
		Permalink: "/blog/first",
		CreatedAt: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		RawMeta: map[string]interface{}{
			"order":    3,
			"category": "tech",
		},
	}
	nvs.PathMap["blog/second.md"] = &model.NoteView{
		Path:      "blog/second.md",
		Title:     "Alpha Post",
		Permalink: "/blog/second",
		CreatedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		RawMeta: map[string]interface{}{
			"order":    1,
			"category": "design",
		},
	}
	nvs.PathMap["blog/third.md"] = &model.NoteView{
		Path:      "blog/third.md",
		Title:     "Beta Post",
		Permalink: "/blog/third",
		CreatedAt: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
		RawMeta: map[string]interface{}{
			"order":    2,
			"category": "tech",
		},
	}

	// Nested posts
	nvs.PathMap["projects/web/readme.md"] = &model.NoteView{
		Path:      "projects/web/readme.md",
		Title:     "Web Project",
		Permalink: "/projects/web/readme",
		CreatedAt: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	nvs.PathMap["projects/mobile/readme.md"] = &model.NoteView{
		Path:      "projects/mobile/readme.md",
		Title:     "Mobile Project",
		Permalink: "/projects/mobile/readme",
		CreatedAt: time.Date(2024, 2, 5, 0, 0, 0, 0, time.UTC),
	}

	// Other files
	nvs.PathMap["about.md"] = &model.NoteView{
		Path:      "about.md",
		Title:     "About",
		Permalink: "/about",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	return templateviews.NewNVS(nvs, "live")
}

func TestNoteQuery_ByGlob(t *testing.T) {
	nvs := createTestNVS()

	tests := []struct {
		name     string
		pattern  string
		expected int
	}{
		{"all blog posts", "blog/*.md", 3},
		{"all readme files recursive", "projects/**/readme.md", 2},
		{"all md files", "*.md", 1}, // only root level
		{"no match", "nonexistent/*.md", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes := nvs.ByGlob(tt.pattern).All()
			require.Len(t, notes, tt.expected)
		})
	}
}

func TestNoteQuery_SortByTitle(t *testing.T) {
	nvs := createTestNVS()

	notes := nvs.ByGlob("blog/*.md").SortBy("Title").All()

	require.Len(t, notes, 3)
	require.Equal(t, "Alpha Post", notes[0].Title())
	require.Equal(t, "Beta Post", notes[1].Title())
	require.Equal(t, "Zebra Post", notes[2].Title())
}

func TestNoteQuery_SortByTitleDesc(t *testing.T) {
	nvs := createTestNVS()

	notes := nvs.ByGlob("blog/*.md").SortBy("Title").Desc().All()

	require.Len(t, notes, 3)
	require.Equal(t, "Zebra Post", notes[0].Title())
	require.Equal(t, "Beta Post", notes[1].Title())
	require.Equal(t, "Alpha Post", notes[2].Title())
}

func TestNoteQuery_SortByCreatedAt(t *testing.T) {
	nvs := createTestNVS()

	notes := nvs.ByGlob("blog/*.md").SortBy("CreatedAt").All()

	require.Len(t, notes, 3)
	require.Equal(t, "Alpha Post", notes[0].Title()) // Jan 10
	require.Equal(t, "Zebra Post", notes[1].Title()) // Jan 15
	require.Equal(t, "Beta Post", notes[2].Title())  // Jan 20
}

func TestNoteQuery_SortByCreatedAtDesc(t *testing.T) {
	nvs := createTestNVS()

	notes := nvs.ByGlob("blog/*.md").SortBy("CreatedAt").Desc().All()

	require.Len(t, notes, 3)
	require.Equal(t, "Beta Post", notes[0].Title())  // Jan 20
	require.Equal(t, "Zebra Post", notes[1].Title()) // Jan 15
	require.Equal(t, "Alpha Post", notes[2].Title()) // Jan 10
}

func TestNoteQuery_SortBySnakeCase(t *testing.T) {
	nvs := createTestNVS()

	// Test snake_case field names
	notes := nvs.ByGlob("blog/*.md").SortBy("created_at").Desc().All()

	require.Len(t, notes, 3)
	require.Equal(t, "Beta Post", notes[0].Title())
}

func TestNoteQuery_SortByMeta(t *testing.T) {
	nvs := createTestNVS()

	notes := nvs.ByGlob("blog/*.md").SortByMeta("order").All()

	require.Len(t, notes, 3)
	require.Equal(t, "Alpha Post", notes[0].Title()) // order: 1
	require.Equal(t, "Beta Post", notes[1].Title())  // order: 2
	require.Equal(t, "Zebra Post", notes[2].Title()) // order: 3
}

func TestNoteQuery_SortByMetaDesc(t *testing.T) {
	nvs := createTestNVS()

	notes := nvs.ByGlob("blog/*.md").SortByMeta("order").Desc().All()

	require.Len(t, notes, 3)
	require.Equal(t, "Zebra Post", notes[0].Title()) // order: 3
	require.Equal(t, "Beta Post", notes[1].Title())  // order: 2
	require.Equal(t, "Alpha Post", notes[2].Title()) // order: 1
}

func TestNoteQuery_MultipleSortCriteria(t *testing.T) {
	nvs := createTestNVS()

	// Sort by category (meta) asc, then by title asc
	notes := nvs.ByGlob("blog/*.md").SortByMeta("category").SortBy("Title").All()

	require.Len(t, notes, 3)
	// design comes before tech
	require.Equal(t, "Alpha Post", notes[0].Title()) // design
	// tech posts sorted by title
	require.Equal(t, "Beta Post", notes[1].Title())  // tech
	require.Equal(t, "Zebra Post", notes[2].Title()) // tech
}

func TestNoteQuery_Limit(t *testing.T) {
	nvs := createTestNVS()

	notes := nvs.ByGlob("blog/*.md").SortBy("Title").Limit(2).All()

	require.Len(t, notes, 2)
	require.Equal(t, "Alpha Post", notes[0].Title())
	require.Equal(t, "Beta Post", notes[1].Title())
}

func TestNoteQuery_Offset(t *testing.T) {
	nvs := createTestNVS()

	notes := nvs.ByGlob("blog/*.md").SortBy("Title").Offset(1).All()

	require.Len(t, notes, 2)
	require.Equal(t, "Beta Post", notes[0].Title())
	require.Equal(t, "Zebra Post", notes[1].Title())
}

func TestNoteQuery_OffsetAndLimit(t *testing.T) {
	nvs := createTestNVS()

	notes := nvs.ByGlob("blog/*.md").SortBy("Title").Offset(1).Limit(1).All()

	require.Len(t, notes, 1)
	require.Equal(t, "Beta Post", notes[0].Title())
}

func TestNoteQuery_First(t *testing.T) {
	nvs := createTestNVS()

	note := nvs.ByGlob("blog/*.md").SortBy("Title").First()

	require.NotNil(t, note)
	require.Equal(t, "Alpha Post", note.Title())
}

func TestNoteQuery_FirstEmpty(t *testing.T) {
	nvs := createTestNVS()

	note := nvs.ByGlob("nonexistent/*.md").First()

	require.Nil(t, note)
}

func TestNoteQuery_Last(t *testing.T) {
	nvs := createTestNVS()

	note := nvs.ByGlob("blog/*.md").SortBy("Title").Last()

	require.NotNil(t, note)
	require.Equal(t, "Zebra Post", note.Title())
}

func TestNoteQuery_LastEmpty(t *testing.T) {
	nvs := createTestNVS()

	note := nvs.ByGlob("nonexistent/*.md").Last()

	require.Nil(t, note)
}

func TestNoteQuery_QueryAll(t *testing.T) {
	nvs := createTestNVS()

	// Query() without glob returns all notes
	notes := nvs.Query().SortBy("Title").All()

	require.Len(t, notes, 6)
	require.Equal(t, "About", notes[0].Title())
}

func TestNoteQuery_NilNVS(t *testing.T) {
	var nvs *templateviews.NVS

	// Should not panic
	require.Nil(t, nvs)
}

func TestNoteQuery_ChainedDescAsc(t *testing.T) {
	nvs := createTestNVS()

	// First sort by category desc, then by order asc
	notes := nvs.ByGlob("blog/*.md").SortByMeta("category").Desc().SortByMeta("order").Asc().All()

	require.Len(t, notes, 3)
	// tech (desc first) sorted by order asc: Beta (2), Zebra (3)
	// design: Alpha (1)
	require.Equal(t, "Beta Post", notes[0].Title())  // tech, order 2
	require.Equal(t, "Zebra Post", notes[1].Title()) // tech, order 3
	require.Equal(t, "Alpha Post", notes[2].Title()) // design, order 1
}
