package sitemap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/model"
)

func TestGenerateForDomain_Basic(t *testing.T) {
	nvs := model.NewNoteViews()
	note := &model.NoteView{
		Permalink:         "/my-note",
		PermalinkOriginal: "/my-note",
		Path:              "my-note.md",
		Free:              true,
		Routes:            []model.ParsedRoute{{Host: "foo.com", Path: "/hello"}},
	}
	nvs.RegisterNote(note)

	result, err := GenerateForDomain(nvs, "foo.com", "https://foo.com")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, string(result), "https://foo.com/hello")
}

func TestGenerateForDomain_OnlyFreeNotes(t *testing.T) {
	nvs := model.NewNoteViews()

	freeNote := &model.NoteView{
		Permalink:         "/free",
		PermalinkOriginal: "/free",
		Path:              "free.md",
		Free:              true,
		Routes:            []model.ParsedRoute{{Host: "foo.com", Path: "/free"}},
	}
	paidNote := &model.NoteView{
		Permalink:         "/paid",
		PermalinkOriginal: "/paid",
		Path:              "paid.md",
		Free:              false,
		Routes:            []model.ParsedRoute{{Host: "foo.com", Path: "/paid"}},
	}
	nvs.RegisterNote(freeNote)
	nvs.RegisterNote(paidNote)

	result, err := GenerateForDomain(nvs, "foo.com", "https://foo.com")
	require.NoError(t, err)
	require.NotNil(t, result)

	xml := string(result)
	require.Contains(t, xml, "https://foo.com/free")
	require.NotContains(t, xml, "https://foo.com/paid")
}

func TestGenerateForDomain_ExcludeSystemPages(t *testing.T) {
	nvs := model.NewNoteViews()

	publicNote := &model.NoteView{
		Permalink:         "/public",
		PermalinkOriginal: "/public",
		Path:              "public.md",
		Free:              true,
		Routes:            []model.ParsedRoute{{Host: "foo.com", Path: "/public"}},
	}
	systemNote := &model.NoteView{
		Permalink:         "/_system",
		PermalinkOriginal: "/_system",
		Path:              "_system.md",
		Free:              true,
		Routes:            []model.ParsedRoute{{Host: "foo.com", Path: "/_system/config"}},
	}
	nvs.RegisterNote(publicNote)
	nvs.RegisterNote(systemNote)

	result, err := GenerateForDomain(nvs, "foo.com", "https://foo.com")
	require.NoError(t, err)
	require.NotNil(t, result)

	xml := string(result)
	require.Contains(t, xml, "https://foo.com/public")
	require.NotContains(t, xml, "_system")
}

func TestGenerateForDomain_EmptyDomain(t *testing.T) {
	nvs := model.NewNoteViews()

	note := &model.NoteView{
		Permalink:         "/my-note",
		PermalinkOriginal: "/my-note",
		Path:              "my-note.md",
		Free:              true,
		Routes:            []model.ParsedRoute{{Host: "bar.com", Path: "/hello"}},
	}
	nvs.RegisterNote(note)

	// Request sitemap for "foo.com" but no routes registered for it.
	result, err := GenerateForDomain(nvs, "foo.com", "https://foo.com")
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestGenerateForDomain_EmptyNvs(t *testing.T) {
	nvs := model.NewNoteViews()

	result, err := GenerateForDomain(nvs, "foo.com", "https://foo.com")
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestGenerateForDomain_LastMod(t *testing.T) {
	nvs := model.NewNoteViews()
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	note := &model.NoteView{
		Permalink:         "/dated",
		PermalinkOriginal: "/dated",
		Path:              "dated.md",
		Free:              true,
		CreatedAt:         ts,
		Routes:            []model.ParsedRoute{{Host: "foo.com", Path: "/dated"}},
	}
	nvs.RegisterNote(note)

	result, err := GenerateForDomain(nvs, "foo.com", "https://foo.com")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, string(result), "<lastmod>2025-06-15T10:30:00Z</lastmod>")
}

func TestGenerate(t *testing.T) {
	tests := []struct {
		name        string
		nvs         *model.NoteViews
		contains    []string
		notContains []string
	}{
		{
			name: "only free notes included",
			nvs: &model.NoteViews{
				List: []*model.NoteView{
					{Permalink: "/free-page", Free: true, CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
					{Permalink: "/paid-page", Free: false},
				},
			},
			contains:    []string{"<loc>https://example.com/free-page</loc>"},
			notContains: []string{"paid-page"},
		},
		{
			name: "system pages excluded",
			nvs: &model.NoteViews{
				List: []*model.NoteView{
					{Permalink: "/public", Free: true},
					{Permalink: "/_system/config", Free: true},
				},
			},
			contains:    []string{"<loc>https://example.com/public</loc>"},
			notContains: []string{"_system"},
		},
		{
			name: "empty notes",
			nvs: &model.NoteViews{
				List: []*model.NoteView{},
			},
			contains:    []string{`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>`},
			notContains: []string{"<url>"},
		},
		{
			name: "lastmod from created_at",
			nvs: &model.NoteViews{
				List: []*model.NoteView{
					{Permalink: "/page", Free: true, CreatedAt: time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)},
				},
			},
			contains: []string{"<lastmod>2025-06-15T10:30:00Z</lastmod>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Generate(tt.nvs, "https://example.com")
			require.NoError(t, err)

			xml := string(result)

			for _, s := range tt.contains {
				require.Contains(t, xml, s)
			}

			for _, s := range tt.notContains {
				require.NotContains(t, xml, s)
			}
		})
	}
}
