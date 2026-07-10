package rssfeed

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/wikilink"

	"trip2g/internal/model"
)

// parseNote creates a NoteView with parsed AST from markdown content.
func parseNote(t *testing.T, content string, nv *model.NoteView) {
	t.Helper()

	md := goldmark.New(
		goldmark.WithExtensions(&wikilink.Extender{}),
	)

	src := []byte(content)
	ctx := parser.NewContext()
	doc := md.Parser().Parse(text.NewReader(src), parser.WithContext(ctx))

	nv.Content = src
	nv.SetAst(doc)
}

func TestGenerate(t *testing.T) {
	tests := []struct {
		name        string
		markdown    string
		note        *model.NoteView
		notes       *model.NoteViews
		contains    []string
		notContains []string
	}{
		{
			name:     "empty note",
			markdown: "No links here.",
			note: &model.NoteView{
				Title:         "Empty",
				Permalink:     "/empty",
				ResolvedLinks: map[string]string{},
			},
			notes: &model.NoteViews{Map: map[string]*model.NoteView{}},
			contains: []string{
				`<title>Empty</title>`,
				`<link>https://example.com/empty</link>`,
			},
			notContains: []string{"<item>"},
		},
		{
			name:     "standard markdown links",
			markdown: "[Google](https://google.com)\n\n[Local page](/about)",
			note: &model.NoteView{
				Title:         "Links",
				Permalink:     "/links",
				ResolvedLinks: map[string]string{},
			},
			notes: &model.NoteViews{
				Map: map[string]*model.NoteView{
					"/about": {
						Title:       "About",
						Permalink:   "/about",
						Free:        true,
						Description: strPtr("About page description"),
						CreatedAt:   time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			contains: []string{
				`<title>Google</title>`,
				`<link>https://google.com</link>`,
				`<title>Local page</title>`,
				`<link>https://example.com/about</link>`,
				`<description>About page description</description>`,
			},
		},
		{
			name:     "wikilinks resolved",
			markdown: "Check out [[my-page]].",
			note: &model.NoteView{
				Title:     "Index",
				Permalink: "/",
				ResolvedLinks: map[string]string{
					"my-page": "/my-page",
				},
			},
			notes: &model.NoteViews{
				Map: map[string]*model.NoteView{
					"/my-page": {
						Title:       "My Page",
						Permalink:   "/my-page",
						Free:        true,
						Description: strPtr("A cool page"),
						CreatedAt:   time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
					},
				},
			},
			contains: []string{
				`<title>my-page</title>`,
				`<link>https://example.com/my-page</link>`,
				`<description>A cool page</description>`,
			},
		},
		{
			name:     "frontmatter overrides",
			markdown: "[Link](/page)",
			note: &model.NoteView{
				Title:          "Original Title",
				Permalink:      "/feed",
				RSSTitle:       "Custom Feed Title",
				RSSDescription: "Custom feed description",
				ResolvedLinks:  map[string]string{},
			},
			notes: &model.NoteViews{Map: map[string]*model.NoteView{}},
			contains: []string{
				`<title>Custom Feed Title</title>`,
				`<description>Custom feed description</description>`,
			},
			notContains: []string{"Original Title"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseNote(t, tt.markdown, tt.note)

			result, err := Generate(tt.note, "https://example.com", tt.notes)
			require.NoError(t, err)

			xml := string(result)

			for _, s := range tt.contains {
				require.Contains(t, xml, s, "expected XML to contain: %s", s)
			}

			for _, s := range tt.notContains {
				require.NotContains(t, xml, s, "expected XML to NOT contain: %s", s)
			}
		})
	}
}

func TestGenerateOmitsPaywalledTargetContent(t *testing.T) {
	note := &model.NoteView{
		Title:         "Feed",
		Permalink:     "/feed",
		Free:          true,
		ResolvedLinks: map[string]string{"secret": "/secret"},
	}
	parseNote(t, "[[secret]]", note)

	result, err := Generate(note, "https://example.com", &model.NoteViews{Map: map[string]*model.NoteView{
		"/secret": {
			Title:       "Secret",
			Permalink:   "/secret",
			Free:        false,
			Description: strPtr("private description"),
			HTML:        "<p>private body</p>",
		},
	}})
	require.NoError(t, err)
	xml := string(result)
	require.Contains(t, xml, `<link>https://example.com/secret</link>`)
	require.NotContains(t, xml, "private body")
	require.NotContains(t, xml, "private description")
	require.NotContains(t, xml, "content:encoded")
}

func TestGenerateOmitsSignInOnlyTargetContent(t *testing.T) {
	note := &model.NoteView{
		Title:         "Feed",
		Permalink:     "/feed",
		Free:          true,
		ResolvedLinks: map[string]string{"members": "/members"},
	}
	parseNote(t, "[[members]]", note)

	result, err := Generate(note, "https://example.com", &model.NoteViews{Map: map[string]*model.NoteView{
		"/members": {
			Title:       "Members",
			Permalink:   "/members",
			Free:        true,
			Description: strPtr("members-only description"),
			HTML:        "<p>members-only body</p>",
			Subgraphs: map[string]*model.NoteSubgraph{
				"members": {RequireSignin: true},
			},
		},
	}})
	require.NoError(t, err)
	xml := string(result)
	require.Contains(t, xml, `<link>https://example.com/members</link>`)
	require.NotContains(t, xml, "members-only body")
	require.NotContains(t, xml, "members-only description")
}

func TestIsPubliclyReadable(t *testing.T) {
	tests := []struct {
		name string
		note *model.NoteView
		want bool
	}{
		{name: "public free note", note: &model.NoteView{Path: "post.md", Permalink: "/post", Free: true}, want: true},
		{name: "paywalled", note: &model.NoteView{Path: "post.md", Permalink: "/post"}},
		{name: "sign-in-only", note: &model.NoteView{
			Path:      "post.md",
			Permalink: "/post",
			Free:      true,
			Subgraphs: map[string]*model.NoteSubgraph{"members": {RequireSignin: true}},
		}},
		{name: "internal route", note: &model.NoteView{Path: "_system/config.md", Permalink: "/_system/config", Free: true}},
		{name: "nested internal route", note: &model.NoteView{Path: "docs/_internal/note.md", Permalink: "/docs/_internal/note", Free: true}},
		{name: "HTML template", note: &model.NoteView{Path: "_layouts/page.html", Permalink: "/page.html", Free: true}},
		{name: "missing note"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, IsPubliclyReadable(tt.note))
		})
	}
}

func strPtr(s string) *string {
	return &s
}
