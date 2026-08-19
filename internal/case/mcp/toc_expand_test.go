package mcp

import (
	"html/template"
	"strings"
	"testing"

	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

// noteWithLeadingH1 mirrors what the real load pipeline produces for a note that
// opens with an H1: the H1 is in Headings at level 1 and wraps the rest of the
// document in its own data-header div, while the template suppresses the visible
// duplicate because NoteView.HasH1 is set.
func noteWithLeadingH1() *model.NoteView {
	return &model.NoteView{
		Path:   "goethe.md",
		PathID: 7,
		Title:  "Goethe — Maxims and Reflections",
		HasH1:  true,
		Headings: model.NoteViewHeadings{
			{Text: "Goethe — Maxims and Reflections", Level: 1, ID: "goethe"},
			{Text: "Maxims", Level: 2, ID: "maxims"},
			{Text: "Art", Level: 3, ID: "art"},
			{Text: "Reflections", Level: 2, ID: "reflections"},
		},
		HTML: template.HTML(`<div data-header="Goethe — Maxims and Reflections" data-level="1"><h1>Goethe — Maxims and Reflections</h1>` +
			`<div data-header="Maxims" data-level="2"><h2>Maxims</h2><p>maxims body</p>` +
			`<div data-header="Art" data-level="3"><h3>Art</h3><p>art body</p></div></div>` +
			`<div data-header="Reflections" data-level="2"><h2>Reflections</h2><p>reflections body</p></div></div>`),
	}
}

func TestExpandDropsTheLeadingH1(t *testing.T) {
	note := noteWithLeadingH1()

	top := tocChildren(note, nil)
	titles := make([]string, len(top))
	for i, c := range top {
		titles[i] = c.Title
	}
	require.Equal(t, []string{"Maxims", "Reflections"}, titles,
		"the leading H1 is the title and is suppressed in the body, so it must not eat a level of navigation")

	// The shortened path still resolves against the unchanged HTML.
	require.NotEmpty(t, sectionHTMLByTocPath(string(note.HTML), []string{"Maxims"}))
	require.Equal(t, []string{"Art"}, []string{tocChildren(note, []string{"Maxims"})[0].Title})
}

func TestExpandKeepsLaterH1s(t *testing.T) {
	note := &model.NoteView{
		Path:  "book.md",
		Title: "Book",
		HasH1: true,
		Headings: model.NoteViewHeadings{
			{Text: "Book", Level: 1},
			{Text: "Part One", Level: 1},
			{Text: "Part Two", Level: 1},
		},
		HTML: template.HTML(`<div data-header="Book" data-level="1"><h1>Book</h1>` +
			`<div data-header="Part One" data-level="1"><h1>Part One</h1><p>one</p></div>` +
			`<div data-header="Part Two" data-level="1"><h1>Part Two</h1><p>two</p></div></div>`),
	}

	top := tocChildren(note, nil)
	require.Len(t, top, 2, "only the first H1 is the title; later ones are real sections")
	require.Equal(t, "Part One", top[0].Title)
	require.Equal(t, "Part Two", top[1].Title)
}

func TestExpandLeavesNotesWithoutLeadingH1Alone(t *testing.T) {
	note := &model.NoteView{
		Path:  "article.md",
		Title: "Article",
		Headings: model.NoteViewHeadings{
			{Text: "Introduction", Level: 1},
			{Text: "Details", Level: 1},
		},
		HTML: template.HTML(`<div data-header="Introduction" data-level="1"><h1>Introduction</h1><p>x</p></div>` +
			`<div data-header="Details" data-level="1"><h1>Details</h1><p>y</p></div>`),
	}

	top := tocChildren(note, nil)
	require.Len(t, top, 2)
	require.Equal(t, "Introduction", top[0].Title)
}

func TestExpandPreviewsShortHeadings(t *testing.T) {
	// An aphoristic corpus: 39 subsections titled "1", "2", ... carry no meaning
	// on their own, so an agent has nothing to choose between when descending.
	note := &model.NoteView{
		Path:  "meditations.md",
		Title: "Книга 10",
		Headings: model.NoteViewHeadings{
			{Text: "1", Level: 1},
			{Text: "2", Level: 1},
			{Text: "A heading long enough to stand on its own", Level: 1},
		},
		HTML: template.HTML(`<div data-header="1" data-level="1"><h1>1</h1><p>Душа моя, ужели ты никогда не будешь доброй и простой?</p></div>` +
			`<div data-header="2" data-level="1"><h1>2</h1><p>Замечай, чего требует твоя природа.</p></div>` +
			`<div data-header="A heading long enough to stand on its own" data-level="1"><h1>A heading long enough to stand on its own</h1><p>body</p></div>`),
	}

	top := tocChildren(note, nil)
	require.Len(t, top, 3)
	require.True(t, strings.HasPrefix(top[0].Preview, "Душа моя"), "got %q", top[0].Preview)
	require.True(t, strings.HasPrefix(top[1].Preview, "Замечай"), "got %q", top[1].Preview)
	require.Empty(t, top[2].Preview, "a heading that already reads on its own needs no preview")
}

func TestExpandPreviewCountsRunesNotBytes(t *testing.T) {
	// "Книга 10" is 8 runes but 15 bytes: a byte-length threshold would call it
	// long enough and skip the preview exactly where it is needed most.
	note := &model.NoteView{
		Path:     "b.md",
		Title:    "Меди",
		Headings: model.NoteViewHeadings{{Text: "Книга 10", Level: 1}},
		HTML:     template.HTML(`<div data-header="Книга 10" data-level="1"><h1>Книга 10</h1><p>Первая строка книги.</p></div>`),
	}

	top := tocChildren(note, nil)
	require.Len(t, top, 1)
	require.NotEmpty(t, top[0].Preview)
}

func TestSectionAnchorForTocPath(t *testing.T) {
	note := noteWithLeadingH1()
	url := func(*model.NoteView) string { return "https://kb.test/goethe" }

	require.Equal(t, "https://kb.test/goethe#maxims",
		sectionAnchorURL(note, []string{"Maxims"}, url))
	require.Equal(t, "https://kb.test/goethe#art",
		sectionAnchorURL(note, []string{"Maxims", "Art"}, url))

	// No path, or a path that names no heading, means there is nothing to
	// anchor to — the note's own URL already covers that.
	require.Empty(t, sectionAnchorURL(note, nil, url))
	require.Empty(t, sectionAnchorURL(note, []string{"Nope"}, url))
}

func TestSnippetBreadcrumbDropsTheTitleH1(t *testing.T) {
	// search and expand have to agree on what a toc_path looks like: expand
	// drops the title H1, so the breadcrumb walked out of the rendered HTML —
	// where that H1 div still wraps everything — must drop it too.
	note := noteWithLeadingH1()

	require.Equal(t, []string{"Maxims"},
		snippetTocPath(note, "maxims body", ""))
	require.Equal(t, []string{"Reflections"},
		snippetTocPath(note, "reflections body", ""))
}
