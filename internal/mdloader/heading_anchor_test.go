package mdloader_test

import (
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

const anchorTarget = `---
free: true
---
## Getting Started
## Цены и тарифы
## FAQ
## FAQ
## Pricing {#plans}
## The ` + "`coalesce`" + ` function
`

func loadAnchorNotes(t *testing.T, index string) *model.NoteViews {
	t.Helper()

	notes, err := mdloader.Load(mdloader.Options{
		Log: &logger.TestLogger{},
		Sources: []mdloader.SourceFile{
			{Path: "index.md", Content: []byte("---\nfree: true\n---\n" + index)},
			{Path: "target.md", Content: []byte(anchorTarget)},
		},
	})
	require.NoError(t, err)

	return notes
}

func brokenLinkWarnings(note *model.NoteView) []string {
	var messages []string
	for _, w := range note.Warnings {
		messages = append(messages, w.Message)
	}
	return messages
}

func TestWikilinkHeadingAnchors(t *testing.T) {
	tests := []struct {
		name string
		link string
		href string
	}{
		{"cross-note text", "[[target#Getting Started]]", `href="/target#getting_started"`},
		{"cross-note case and spaces", "[[target#getting   started]]", `href="/target#getting_started"`},
		{"cross-note cyrillic", "[[target#Цены и тарифы]]", `href="/target#cenyi_i_tarifyi"`},
		{"cross-note duplicate first", "[[target#FAQ]]", `href="/target#faq"`},
		{"cross-note duplicate by id", "[[target#faq-2]]", `href="/target#faq-2"`},
		{"cross-note explicit id", "[[target#plans]]", `href="/target#plans"`},
		{"cross-note text of explicit id", "[[target#Pricing]]", `href="/target#plans"`},
		{"cross-note code span", "[[target#The coalesce function]]", `href="/target#the_coalesce_function"`},
		{"cross-note alias", "[[target#Цены и тарифы|prices]]", `href="/target#cenyi_i_tarifyi"`},
		{"same note text", "## Фильтры экранирования\n\n[[#Фильтры экранирования]]", `href="#filjtryi_ekranirovaniya"`},
		{"same note alias", "## Setup\n\n[[#setup|see setup]]", `href="#setup"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes := loadAnchorNotes(t, tt.link)
			index := notes.PathMap["index.md"]

			html := string(index.HTML)
			require.Contains(t, html, tt.href)
			require.NotContains(t, html, `class="wip"`)
			require.Empty(t, brokenLinkWarnings(index))
		})
	}
}

func TestWikilinkHeadingAnchorUnknown(t *testing.T) {
	tests := []struct {
		name    string
		link    string
		href    string
		warning string
	}{
		{"cross-note", "[[target#Missing]]", `href="/target#Missing"`, "broken link: target#Missing"},
		{"same note", "## Setup\n\n[[#Missing|see]]", `href="#Missing"`, "broken link: #Missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes := loadAnchorNotes(t, tt.link)
			index := notes.PathMap["index.md"]

			html := string(index.HTML)
			require.Contains(t, html, tt.href)
			require.Contains(t, html, `class="wip"`)
			require.Equal(t, []string{tt.warning}, brokenLinkWarnings(index))
		})
	}
}

func TestWikilinkBlockRefUnchanged(t *testing.T) {
	notes := loadAnchorNotes(t, "[[target#^block-id]] [[#^own]]")
	index := notes.PathMap["index.md"]

	html := string(index.HTML)
	require.Contains(t, html, `href="/target#%5Eblock-id"`)
	require.Contains(t, html, `href="#%5Eown"`)
	require.NotContains(t, html, `class="wip"`)
	require.Empty(t, brokenLinkWarnings(index))
}
