package model_test

import (
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"

	"github.com/stretchr/testify/require"
)

func TestNavLinks(t *testing.T) {
	notes := []string{
		"templates.md",
		"other.md",
		"docs/templates.md",
		"docs/sub/page.md",
		"docs/page name.md",
		"ru/user/jet-functions.md",
		"docs/props.md",
	}
	contents := map[string]string{
		"docs/props.md": "---\nroute: /props\n---\nbody",
	}

	tests := []struct {
		name    string
		sidebar string
		want    []string // "heading > heading note > note", parts omitted when empty
	}{
		{
			name:    "bare .md resolves in the sidebar folder first",
			sidebar: "- [T](templates.md)",
			want:    []string{"docs/templates.md"},
		},
		{
			name:    "no extension",
			sidebar: "- [T](templates)",
			want:    []string{"docs/templates.md"},
		},
		{
			name:    "falls back to wikilink resolution",
			sidebar: "- [O](other.md)\n- [J](ru/user/jet-functions.md)",
			want:    []string{"other.md", "ru/user/jet-functions.md"},
		},
		{
			name:    "relative paths",
			sidebar: "- [P](./sub/page.md)\n- [O](../other.md)",
			want:    []string{"docs/sub/page.md", "other.md"},
		},
		{
			name:    "url-encoded",
			sidebar: "- [P](page%20name.md)",
			want:    []string{"docs/page name.md"},
		},
		{
			name:    "permalink",
			sidebar: "- [J](/ru/user/jet_functions)",
			want:    []string{"ru/user/jet-functions.md"},
		},
		{
			name:    "route alias",
			sidebar: "- [P](/props)",
			want:    []string{"docs/props.md"},
		},
		{
			name: "skips external, anchors, fragments, assets and misses",
			sidebar: "- [a](https://example.com/x.md)\n- [b](//cdn.example.com/x)\n- [c](mailto:a@b.c)\n" +
				"- [d](tel:123)\n- [e](#top)\n- [f](templates.md#part)\n- [g](pic.png)\n- [h](missing.md)\n- [i](/missing)",
		},
		{
			name:    "markdown link in a heading becomes the heading note",
			sidebar: "### [Guide](templates.md)\n\n- [O](../other.md)",
			want:    []string{"Guide > docs/templates.md > other.md"},
		},
		{
			name:    "mixed wikilinks and markdown links keep document order",
			sidebar: "### Start\n\n- [[other]]\n- [P](sub/page.md)\n- [[docs/templates|T]]\n- [J](/ru/user/jet_functions)",
			want: []string{
				"Start > other.md",
				"Start > docs/sub/page.md",
				"Start > docs/templates.md",
				"Start > ru/user/jet-functions.md",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcs := []mdloader.SourceFile{{Path: "docs/_sidebar.md", Content: []byte(tt.sidebar)}}
			for _, path := range notes {
				content := "body"
				if c, ok := contents[path]; ok {
					content = c
				}
				srcs = append(srcs, mdloader.SourceFile{Path: path, Content: []byte(content)})
			}
			for i := range srcs {
				srcs[i].PathID = int64(i + 1)
			}

			nvs, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &logger.TestLogger{}})
			require.NoError(t, err)

			var got []string
			for _, link := range nvs.NavLinks(nvs.PathMap["docs/_sidebar.md"]) {
				s := link.Note.Path
				if link.HeadingNote != nil {
					s = link.HeadingNote.Path + " > " + s
				}
				if link.Heading != "" {
					s = link.Heading + " > " + s
				}
				got = append(got, s)
			}
			require.Equal(t, tt.want, got)
		})
	}
}
