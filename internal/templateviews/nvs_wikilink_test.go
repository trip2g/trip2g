package templateviews_test

import (
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/templateviews"

	"github.com/stretchr/testify/require"
)

func TestNoteByWikilinkFrom(t *testing.T) {
	paths := []string{
		"_header.md",
		"Site Footer.md",
		"ru/_header.md",
		"ru/user/_sidebar.md",
		"ru/user/page.md",
		"ru/user/Свойства заметок.md",
		"docs/page.md",
	}
	var srcs []mdloader.SourceFile
	for i, p := range paths {
		srcs = append(srcs, mdloader.SourceFile{Path: p, PathID: int64(i + 1), Content: []byte("body")})
	}
	loaded, err := mdloader.Load(mdloader.Options{Sources: srcs, Log: &logger.TestLogger{}})
	require.NoError(t, err)
	nvs := templateviews.NewNVS(loaded, "live")
	page := nvs.NoteByPath("ru/user/page.md")

	tests := []struct {
		name   string
		source *templateviews.Note
		target string
		want   string // "" means not found
	}{
		{"bare name", page, "_sidebar", "ru/user/_sidebar.md"},
		{"name with spaces and cyrillic", page, "Свойства заметок", "ru/user/Свойства заметок.md"},
		{"alias is ignored", page, "ru/user/_sidebar|Menu", "ru/user/_sidebar.md"},
		{"heading is ignored", page, "docs/page#Intro", "docs/page.md"},
		{".md suffix", page, "docs/page.md", "docs/page.md"},
		{"relative path", page, "../_header", "ru/_header.md"},
		{"no source: bare name", nil, "Site Footer", "Site Footer.md"},
		{"no source: path", nil, "ru/user/_sidebar", "ru/user/_sidebar.md"},
		{"miss", page, "missing", ""},
		{"empty", page, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nvs.NoteByWikilinkFrom(tt.source, tt.target)
			if tt.want == "" {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Equal(t, tt.want, got.Path())
		})
	}
}
