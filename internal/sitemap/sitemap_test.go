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

func TestGenerateForDomain_ExcludeNoIndex(t *testing.T) {
	nvs := model.NewNoteViews()

	listed := &model.NoteView{
		Permalink:         "/listed",
		PermalinkOriginal: "/listed",
		Path:              "listed.md",
		Free:              true,
		Routes:            []model.ParsedRoute{{Host: "foo.com", Path: "/listed"}},
	}
	hidden := &model.NoteView{
		Permalink:         "/hidden",
		PermalinkOriginal: "/hidden",
		Path:              "hidden.md",
		Free:              true,
		NoIndex:           true,
		Routes:            []model.ParsedRoute{{Host: "foo.com", Path: "/hidden"}},
	}
	nvs.RegisterNote(listed)
	nvs.RegisterNote(hidden)

	result, err := GenerateForDomain(nvs, "foo.com", "https://foo.com")
	require.NoError(t, err)
	require.NotNil(t, result)

	xml := string(result)
	require.Contains(t, xml, "https://foo.com/listed")
	require.NotContains(t, xml, "hidden")
}

func TestGenerateForDomain_NothingIndexable(t *testing.T) {
	nvs := model.NewNoteViews()

	hidden := &model.NoteView{
		Permalink:         "/hidden",
		PermalinkOriginal: "/hidden",
		Path:              "hidden.md",
		Free:              true,
		NoIndex:           true,
		Routes:            []model.ParsedRoute{{Host: "foo.com", Path: "/"}},
	}
	nvs.RegisterNote(hidden)

	// A known domain with nothing to index gets an empty urlset of its own,
	// never nil: nil makes the handler serve the main domain's sitemap.
	result, err := GenerateForDomain(nvs, "foo.com", "https://foo.com")
	require.NoError(t, err)
	require.NotNil(t, result)

	xml := string(result)
	require.Contains(t, xml, "<urlset")
	require.NotContains(t, xml, "<url>")
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

func TestGenerate_HreflangAlternates(t *testing.T) {
	en := &model.NoteView{Permalink: "/en/guide", PermalinkOriginal: "/en/guide", Free: true, Lang: "en"}
	ru := &model.NoteView{Permalink: "/ru/guide", PermalinkOriginal: "/ru/guide", Free: true, Lang: "ru"}
	group := &model.LangGroup{
		Hub: en,
		Versions: []model.LangRedirect{
			{Lang: "en", Note: en, URL: "/en/guide"},
			{Lang: "ru", Note: ru, URL: "/ru/guide"},
		},
	}
	en.LangGroup = group
	ru.LangGroup = group

	nvs := &model.NoteViews{List: []*model.NoteView{en, ru}}

	result, err := Generate(nvs, "https://example.com")
	require.NoError(t, err)
	xml := string(result)

	require.Contains(t, xml, `xmlns:xhtml="http://www.w3.org/1999/xhtml"`)
	require.Contains(t, xml, `<xhtml:link rel="alternate" hreflang="en" href="https://example.com/en/guide">`)
	require.Contains(t, xml, `<xhtml:link rel="alternate" hreflang="ru" href="https://example.com/ru/guide">`)
	require.Contains(t, xml, `<xhtml:link rel="alternate" hreflang="x-default" href="https://example.com/en/guide">`)
}

func TestGenerate_HreflangAlternatesSkipNoIndex(t *testing.T) {
	en := &model.NoteView{Permalink: "/en/guide", PermalinkOriginal: "/en/guide", Free: true, Lang: "en"}
	ru := &model.NoteView{Permalink: "/ru/guide", PermalinkOriginal: "/ru/guide", Free: true, Lang: "ru"}
	de := &model.NoteView{Permalink: "/de/guide", PermalinkOriginal: "/de/guide", Free: true, Lang: "de", NoIndex: true}
	group := &model.LangGroup{
		Hub: en,
		Versions: []model.LangRedirect{
			{Lang: "en", Note: en, URL: "/en/guide"},
			{Lang: "ru", Note: ru, URL: "/ru/guide"},
			{Lang: "de", Note: de, URL: "/de/guide"},
		},
	}
	en.LangGroup = group
	ru.LangGroup = group
	de.LangGroup = group

	nvs := &model.NoteViews{List: []*model.NoteView{en, ru, de}}

	result, err := Generate(nvs, "https://example.com")
	require.NoError(t, err)
	xml := string(result)

	require.Contains(t, xml, `<xhtml:link rel="alternate" hreflang="ru" href="https://example.com/ru/guide">`)
	require.NotContains(t, xml, "/de/guide", "a noindex version is neither listed nor named as an alternate")
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
			name: "noindex notes excluded",
			nvs: &model.NoteViews{
				List: []*model.NoteView{
					{Permalink: "/public", Free: true},
					{Permalink: "/offer", Free: true, NoIndex: true},
				},
			},
			contains:    []string{"<loc>https://example.com/public</loc>"},
			notContains: []string{"offer"},
		},
		{
			name: "free note in a sign-in subgraph listed",
			nvs: &model.NoteViews{
				List: []*model.NoteView{
					{Permalink: "/members-free", Path: "members-free.md", Free: true, SubgraphNames: []string{"members"}},
					{Permalink: "/members-noindex", Path: "members-noindex.md", Free: true, NoIndex: true, SubgraphNames: []string{"members"}},
					{Permalink: "/members-closed", Path: "members-closed.md", SubgraphNames: []string{"members"}},
				},
				Subgraphs: map[string]*model.NoteSubgraph{"members": {Name: "members", RequireSignin: true}},
			},
			contains:    []string{"<loc>https://example.com/members-free</loc>"},
			notContains: []string{"members-noindex", "members-closed"},
		},
		{
			name: "html files excluded",
			nvs: &model.NoteViews{
				List: []*model.NoteView{
					{Permalink: "/public", Path: "public.md", Free: true},
					{Permalink: "/page.html", Path: "page.html", Free: true},
				},
			},
			contains:    []string{"<loc>https://example.com/public</loc>"},
			notContains: []string{"page.html"},
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

func TestGenerateForDomain_FreeNoteInSigninSubgraph(t *testing.T) {
	nvs := model.NewNoteViews()
	nvs.Subgraphs["members"] = &model.NoteSubgraph{Name: "members", RequireSignin: true}

	free := &model.NoteView{
		Permalink:         "/free",
		PermalinkOriginal: "/free",
		Path:              "free.md",
		Free:              true,
		SubgraphNames:     []string{"members"},
		Routes:            []model.ParsedRoute{{Host: "foo.com", Path: "/free"}},
	}
	noindex := &model.NoteView{
		Permalink:         "/quiet",
		PermalinkOriginal: "/quiet",
		Path:              "quiet.md",
		Free:              true,
		NoIndex:           true,
		SubgraphNames:     []string{"members"},
		Routes:            []model.ParsedRoute{{Host: "foo.com", Path: "/quiet"}},
	}
	nvs.RegisterNote(free)
	nvs.RegisterNote(noindex)

	result, err := GenerateForDomain(nvs, "foo.com", "https://foo.com")
	require.NoError(t, err)

	xml := string(result)
	require.Contains(t, xml, "https://foo.com/free")
	require.NotContains(t, xml, "quiet")
}

func TestGenerateForDomain_UnderscoreFolderRoutedToDomain(t *testing.T) {
	nvs := model.NewNoteViews()

	about := &model.NoteView{
		Permalink:         "/_mysite/about",
		PermalinkOriginal: "/_mysite/about",
		Path:              "_mysite/about.md",
		Free:              true,
		Routes:            []model.ParsedRoute{{Host: "mysite.com", Path: "/about"}},
	}
	nvs.RegisterNote(about)

	result, err := GenerateForDomain(nvs, "mysite.com", "https://mysite.com")
	require.NoError(t, err)
	require.Contains(t, string(result), "<loc>https://mysite.com/about</loc>")

	main, err := Generate(nvs, "https://example.com")
	require.NoError(t, err)
	require.NotContains(t, string(main), "_mysite")
}
