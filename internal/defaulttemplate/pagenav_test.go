package defaulttemplate

import (
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"

	"github.com/stretchr/testify/require"
)

func TestFindPageNav(t *testing.T) {
	a := &model.NoteView{PathID: 1, Title: "A", Permalink: "/a"}
	b := &model.NoteView{PathID: 2, Title: "B", Permalink: "/b"}
	c := &model.NoteView{PathID: 3, Title: "C", Permalink: "/c"}
	sec := &model.NoteView{PathID: 9, Title: "Section", Permalink: "/sec"}

	links := []model.NoteNavLink{
		{Note: a},
		{Note: b, Heading: "Start", Label: "Bee"},
		{Note: c, Heading: "Guide", HeadingNote: sec},
		{Note: a, Heading: "Guide", HeadingNote: sec},
	}

	tests := []struct {
		name   string
		pathID int64
		want   PageNav
		found  bool
	}{
		{
			name:   "first link has no prev and no heading",
			pathID: 1,
			want:   PageNav{Next: &NavLink{Label: "Bee", Href: "/b"}},
			found:  true,
		},
		{
			name:   "plain heading becomes a crumb without href",
			pathID: 2,
			want: PageNav{
				Prev:        &NavLink{Label: "A", Href: "/a"},
				Next:        &NavLink{Label: "C", Href: "/c"},
				Breadcrumbs: []NavLink{{Label: "Start"}},
			},
			found: true,
		},
		{
			name:   "linked heading becomes a linked crumb",
			pathID: 3,
			want: PageNav{
				Prev:        &NavLink{Label: "Bee", Href: "/b"},
				Next:        &NavLink{Label: "A", Href: "/a"},
				Breadcrumbs: []NavLink{{Label: "Guide", Href: "/sec"}},
			},
			found: true,
		},
		{
			name:   "not listed",
			pathID: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := findPageNav(links, tt.pathID)
			require.Equal(t, tt.found, found)
			require.Equal(t, tt.want, got)
		})
	}
}

const pageNavSidebar = `### Start

- [[a]]
- [[b|Bee]]
- [External](https://example.com)
- [[c#Part|C part]]
- ![[pic.png]]

### [[guide|Guide]]

- [[c]]
- [[a]]
- [[override]]
`

func TestCtxPageNav(t *testing.T) {
	page := func(path, frontmatter string) mdloader.SourceFile {
		return mdloader.SourceFile{Path: path, Content: []byte("---\n" + frontmatter + "\n---\nbody")}
	}
	srcs := []mdloader.SourceFile{
		{Path: "docs/_sidebar.md", Content: []byte(pageNavSidebar)},
		{Path: "docs/_right.md", Content: []byte("- [Dee](d.md)\n- [[guide]]\n")},
		page("docs/a.md", "title: A\nleft_sidebar: docs/_sidebar.md"),
		page("docs/b.md", "title: B\nleft_sidebar: docs/_sidebar.md"),
		page("docs/c.md", "title: C\nleft_sidebar: docs/_sidebar.md"),
		page("docs/guide.md", "title: Guide page\nleft_sidebar: toc\nright_sidebar: docs/_right.md"),
		page("docs/d.md", "title: D\nleft_sidebar: [toc, docs/_sidebar.md]\nright_sidebar: docs/_right.md"),
		page("docs/lonely.md", "title: Lonely\nleft_sidebar: docs/_sidebar.md"),
		page("docs/override.md", `title: Override
left_sidebar: docs/_sidebar.md
prev: false
next: "[[docs/guide]]"
breadcrumbs:
  - "[[docs/guide]]"
  - docs/missing.md
  - {label: Plain}
  - {label: Home, href: /}`),
		page("docs/b2.md", "title: B2\nleft_sidebar: docs/_sidebar.md\nnext: docs/a.md\nbreadcrumbs: false"),
		page("docs/short.md", "title: Short\nprev: \"[[a|Alpha]]\"\nnext: \"[[guide#Intro]]\""),
	}

	for i := range srcs {
		srcs[i].PathID = int64(i + 1)
	}

	log := logger.TestLogger{}
	pages, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &log})
	require.NoError(t, err)
	nvs := templateviews.NewNVS(pages, "")

	tests := []struct {
		path string
		want PageNav
	}{
		{
			path: "docs/b.md",
			want: PageNav{
				Prev:        &NavLink{Label: "A", Href: "/docs/a"},
				Next:        &NavLink{Label: "C", Href: "/docs/c"},
				Breadcrumbs: []NavLink{{Label: "Start"}},
			},
		},
		{
			path: "docs/c.md",
			want: PageNav{
				Prev:        &NavLink{Label: "Bee", Href: "/docs/b"},
				Next:        &NavLink{Label: "A", Href: "/docs/a"},
				Breadcrumbs: []NavLink{{Label: "Guide", Href: "/docs/guide"}},
			},
		},
		{
			path: "docs/guide.md",
			want: PageNav{Prev: &NavLink{Label: "Dee", Href: "/docs/d"}},
		},
		{
			// Not in the left sidebar note, so the right one is used. A bare wikilink keeps the title.
			path: "docs/d.md",
			want: PageNav{Next: &NavLink{Label: "Guide page", Href: "/docs/guide"}},
		},
		{path: "docs/lonely.md"},
		{
			path: "docs/b2.md",
			want: PageNav{Next: &NavLink{Label: "A", Href: "/docs/a"}},
		},
		{
			// Short Obsidian-style links: alias and heading are ignored.
			path: "docs/short.md",
			want: PageNav{
				Prev: &NavLink{Label: "A", Href: "/docs/a"},
				Next: &NavLink{Label: "Guide page", Href: "/docs/guide"},
			},
		},
		{
			path: "docs/override.md",
			want: PageNav{
				Next: &NavLink{Label: "Guide page", Href: "/docs/guide"},
				Breadcrumbs: []NavLink{
					{Label: "Guide page", Href: "/docs/guide"},
					{Label: "Plain"},
					{Label: "Home", Href: "/"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			ctx := &Ctx{Note: nvs.NoteByPath(tt.path), Notes: nvs}
			require.Equal(t, tt.want, ctx.PageNav())
		})
	}
}
