package sitemap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/model"
)

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
