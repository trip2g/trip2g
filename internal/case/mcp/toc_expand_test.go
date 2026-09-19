package mcp

import (
	"html/template"
	"strconv"
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

// A one-line "- Title — preview" listing made the whole line look like the
// title: a model copied "Prerequisites — Docker running locally Node.js …"
// into toc_path and got "section not found". The bullet line now carries
// nothing but the title, so copying it verbatim resolves.
func TestExpandSummaryPutsThePreviewOnItsOwnLine(t *testing.T) {
	note := &model.NoteView{Title: "Guide"}
	tests := []struct {
		name      string
		children  []TOCNode
		wantLines []string
	}{
		{
			name:     "a child with a preview takes two lines",
			children: []TOCNode{{Title: "Prerequisites", Preview: "Docker running locally Node.js 20+"}},
			wantLines: []string{
				"- Prerequisites",
				"    preview: Docker running locally Node.js 20+",
			},
		},
		{
			name:      "a child without a preview takes one",
			children:  []TOCNode{{Title: "A heading long enough to stand on its own"}},
			wantLines: []string{"- A heading long enough to stand on its own"},
		},
		{
			name:     "the has_children marker stays on the title line",
			children: []TOCNode{{Title: "Setup", HasChildren: true, Preview: "pick a package manager"}},
			wantLines: []string{
				"- Setup (has subsections)",
				"    preview: pick a package manager",
			},
		},
		{
			name: "children keep their own lines",
			children: []TOCNode{
				{Title: "1", Preview: "Душа моя, ужели ты никогда не будешь доброй?"},
				{Title: "2", HasChildren: true},
			},
			wantLines: []string{
				"- 1",
				"    preview: Душа моя, ужели ты никогда не будешь доброй?",
				"- 2 (has subsections)",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := strings.Split(strings.TrimRight(expandSummary(note, nil, tt.children, len(tt.children), 0, 0), "\n"), "\n")
			require.Equal(t, `Guide — "top level", `+strconv.Itoa(len(tt.children))+" subsection(s):", lines[0])
			require.Equal(t, tt.wantLines, lines[1:])

			// A bullet line is exactly the string an agent copies into toc_path,
			// so no preview text may leak onto one.
			for _, line := range lines[1:] {
				if !strings.HasPrefix(line, "- ") {
					continue
				}
				for _, c := range tt.children {
					if c.Preview != "" {
						require.NotContains(t, line, c.Preview)
					}
				}
			}
		})
	}
}

// dailyLog is the shape a note takes when something appends a dated section to
// it every day: a year of them, newest last, each holding a line or two. The
// listing of such a note is the thing `last` exists to bound.
func dailyLog(sections int) *model.NoteView {
	note := &model.NoteView{Path: "log.md", PathID: 11, Title: "Log"}
	var html strings.Builder
	for i := range sections {
		day := "2026-01-" + strconv.Itoa(i+1)
		note.Headings = append(note.Headings, model.NoteViewHeading{Text: day, Level: 3, ID: day})
		html.WriteString(`<div data-header="` + day + `" data-level="3"><h3>` + day + `</h3><p>what moved on ` + day + `</p></div>`)
	}
	note.HTML = template.HTML(html.String())
	return note
}

func TestBoundChildrenKeepsTheEndsAndCountsTheMiddle(t *testing.T) {
	all := tocChildren(dailyLog(365), nil)
	require.Len(t, all, 365)

	for _, tt := range []struct {
		name          string
		first, last   int
		wantLen       int
		wantOmitted   int
		wantFirstItem string
		wantLastItem  string
	}{
		{"neither bound is everything", 0, 0, 365, 0, "2026-01-1", "2026-01-365"},
		{"last alone keeps the newest", 0, 30, 30, 335, "2026-01-336", "2026-01-365"},
		{"first alone keeps the oldest", 5, 0, 5, 360, "2026-01-1", "2026-01-5"},
		{"both keep the ends", 5, 30, 35, 330, "2026-01-1", "2026-01-365"},
		{"bounds that meet are everything", 100, 265, 365, 0, "2026-01-1", "2026-01-365"},
		{"bounds that overlap are everything", 300, 300, 365, 0, "2026-01-1", "2026-01-365"},
		{"a negative bound is no bound", -5, -5, 365, 0, "2026-01-1", "2026-01-365"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, omitted := boundChildren(all, tt.first, tt.last)

			require.Len(t, got, tt.wantLen)
			require.Equal(t, tt.wantOmitted, omitted)
			require.Equal(t, tt.wantFirstItem, got[0].Title)
			require.Equal(t, tt.wantLastItem, got[len(got)-1].Title)

			seen := map[string]bool{}
			for _, c := range got {
				require.False(t, seen[c.Title], "a section must not be listed twice: %s", c.Title)
				seen[c.Title] = true
			}
		})
	}
}

func TestExpandSummarySaysWhichEndsItKept(t *testing.T) {
	note := dailyLog(365)
	all := tocChildren(note, nil)

	newest, omitted := boundChildren(all, 0, 30)
	summary := expandSummary(note, nil, newest, len(all), 0, omitted)
	require.Contains(t, summary, "newest 30 of 365 subsection(s)",
		"a bounded listing that reads as a complete one sends the caller away believing the rest is not there")
	require.Contains(t, summary, "2026-01-365", "the newest section is the one the caller came for")

	oldest, omitted := boundChildren(all, 5, 0)
	summary = expandSummary(note, nil, oldest, len(all), 5, omitted)
	require.Contains(t, summary, "oldest 5 of 365 subsection(s)")

	ends, omitted := boundChildren(all, 5, 30)
	summary = expandSummary(note, nil, ends, len(all), 5, omitted)
	require.Contains(t, summary, "oldest 5 and newest 30 of 365 subsection(s)")
	require.Contains(t, summary, "… 330 subsection(s) not listed …",
		"the gap goes where it falls, or two dates that are a year apart read as consecutive")
	require.Less(t, strings.Index(summary, "2026-01-5"), strings.Index(summary, "not listed"))
	require.Less(t, strings.Index(summary, "not listed"), strings.Index(summary, "2026-01-336"))
}

func TestExpandSummaryUnboundedListingIsUnchanged(t *testing.T) {
	note := dailyLog(3)
	all := tocChildren(note, nil)

	summary := expandSummary(note, nil, all, len(all), 0, 0)

	require.Contains(t, summary, "3 subsection(s)")
	require.NotContains(t, summary, "newest",
		"a listing that holds everything must not describe itself as a selection")
	require.NotContains(t, summary, "not listed")
}
