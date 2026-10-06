package mdloader_test

import (
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestExplicitHeadingIDs(t *testing.T) {
	nv := loadSingleNote(t, "## Setup {#install .wide}\n\nbody\n\n## Привет мир\n\ntext\n")

	require.Equal(t, model.NoteViewHeadings{
		{Text: "Setup", Level: 1, ID: "install"},
		{Text: "Привет мир", Level: 1, ID: "privet_mir"},
	}, nv.Headings)

	html := string(nv.HTML)
	require.Contains(t, html, `data-header="Setup"`)
	require.Contains(t, html, `<h2 id="install" class="wide">Setup</h2>`)
	require.Contains(t, html, `<h2 id="privet_mir">`)
	require.NotContains(t, html, "{#")
}

func TestExplicitHeadingIDsOnSetextAndClosedATX(t *testing.T) {
	nv := loadSingleNote(t, "Overview {#top}\n========\n\n## Details ## {#more}\n")

	require.Equal(t, model.NoteViewHeadings{
		{Text: "Overview", Level: 1, ID: "top"},
		{Text: "Details", Level: 2, ID: "more"},
	}, nv.Headings)
}

func TestHeadingIDCollisions(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "duplicates get a suffix",
			src:  "## Intro\n\n## Intro\n\n## Intro\n",
			want: []string{"intro", "intro-2", "intro-3"},
		},
		{
			name: "manual id after a generated one wins",
			src:  "## Intro\n\n## Start {#intro}\n",
			want: []string{"intro-2", "intro"},
		},
		{
			name: "manual id before a generated one wins",
			src:  "## Start {#intro}\n\n## Intro\n",
			want: []string{"intro", "intro-2"},
		},
		{
			name: "generated suffix skips a manual suffix",
			src:  "## Intro\n\n## Other {#intro-2}\n\n## Intro\n",
			want: []string{"intro", "intro-2", "intro-3"},
		},
		{
			name: "non-string id is replaced by a generated one",
			src:  "## Numbers {id=5}\n",
			want: []string{"numbers"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nv := loadSingleNote(t, tt.src)

			var ids []string
			for _, h := range nv.Headings {
				ids = append(ids, h.ID)
			}
			require.Equal(t, tt.want, ids)
		})
	}
}

func TestSectionsCarryAnchors(t *testing.T) {
	nv := loadSingleNote(t, "---\ntoc: show\n---\n## First Part\n\none\n\n### Deep {#deep}\n\ntwo\n\n## Second {#second .x}\n\nthree\n")

	sections := nv.PartialRenderer.Sections(2)
	require.Len(t, sections, 2)
	require.Equal(t, "first_part", sections[0].ID)
	require.Equal(t, 2, sections[0].Level)
	require.Equal(t, "Second", sections[1].Title)
	require.Equal(t, "second", sections[1].ID)

	nested := sections[0].Sections(3)
	require.Len(t, nested, 1)
	require.Equal(t, "deep", nested[0].ID)
	require.Equal(t, "Deep", nested[0].Title)
	require.Equal(t, 3, nested[0].Level)

	var tocIDs []string
	for _, h := range nv.TOC() {
		tocIDs = append(tocIDs, h.ID)
	}
	require.Equal(t, []string{"first_part", "deep", "second"}, tocIDs)

	for _, h := range nv.TOC() {
		section, ok := nv.PartialRenderer.Section("#" + h.ID).(*model.NoteViewSection)
		require.True(t, ok, h.ID)
		require.Equal(t, h.ID, section.ID)
		require.Equal(t, h.Text, section.Title)
	}
}

func TestSectionLookup(t *testing.T) {
	nv := loadSingleNote(t, "## Getting  Started\n\nfirst\n\n## getting started\n\nsecond\n\n## Pricing {#plans}\n\nthird\n\n### Details\n\nfourth\n")

	tests := []struct {
		name    string
		query   string
		wantID  string
		content string
	}{
		{"exact title wins over a case-insensitive match", "getting started", "getting_started-2", "second"},
		{"case and spaces ignored", "  GETTING started ", "getting_started", "first"},
		{"anchor", "plans", "plans", "third"},
		{"anchor with hash", "#plans", "plans", "third"},
		{"title of a heading with explicit id", "Pricing", "plans", "third"},
		{"nested heading by anchor", "details", "details", "fourth"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section, ok := nv.PartialRenderer.Section(tt.query).(*model.NoteViewSection)
			require.True(t, ok)
			require.Equal(t, tt.wantID, section.ID)
			require.Contains(t, section.ContentHTML, tt.content)
		})
	}

	require.Nil(t, nv.PartialRenderer.Section("missing"))
	require.Nil(t, nv.PartialRenderer.Section("#"))
	require.Nil(t, nv.PartialRenderer.Section(""))

	pricing, ok := nv.PartialRenderer.Section("plans").(*model.NoteViewSection)
	require.True(t, ok)
	nested, ok := pricing.Section("#details").(*model.NoteViewSection)
	require.True(t, ok)
	require.Equal(t, "Details", nested.Title)
	require.Nil(t, pricing.Section("first"))
}

func TestListTaskStates(t *testing.T) {
	nv := loadSingleNote(
		t,
		"- plain\n- [ ] open\n- [x] closed\n- [X] closed\n- [/] half **done**\n- [-] dropped\n- [>] moved [there](https://example.com)\n- [link](https://example.com)\n",
	)

	list, ok := nv.PartialRenderer.FirstList().(*model.NoteViewList)
	require.True(t, ok)

	type state struct{ Text, URL, Task, TaskMark string }
	var got []state
	for _, item := range list.Items {
		got = append(got, state{item.Text, item.URL, item.Task, item.TaskMark})
	}

	require.Equal(t, []state{
		{"plain", "", "", ""},
		{"open", "", "todo", " "},
		{"closed", "", "done", "x"},
		{"closed", "", "done", "X"},
		{"half ", "", "done", "/"},
		{"dropped", "", "done", "-"},
		{"there", "https://example.com", "done", ">"},
		{"link", "https://example.com", "", ""},
	}, got)
}

func TestNestedListTaskStates(t *testing.T) {
	nv := loadSingleNote(t, "- [ ] parent\n  - [x] child\n  - [?] question\n")

	list, ok := nv.PartialRenderer.FirstList().(*model.NoteViewList)
	require.True(t, ok)
	require.Len(t, list.Items, 1)
	require.Equal(t, "todo", list.Items[0].Task)
	require.Len(t, list.Items[0].Children, 2)
	require.Equal(t, "x", list.Items[0].Children[0].TaskMark)
	require.Equal(t, "?", list.Items[0].Children[1].TaskMark)
	require.Equal(t, "question", list.Items[0].Children[1].Text)
}

func TestImages(t *testing.T) {
	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path: "gallery.md",
			Content: []byte("Intro ![first](./a.png)\n\n" +
				"## Shots\n\n![[b.jpg|Second shot|200]]\n\n" +
				"- ![third](https://cdn.example.com/c.webp \"Caption\")\n\n" +
				"![[doc.pdf]] ![[clip.mp4]] ![](https://www.youtube.com/watch?v=dQw4w9WgXcQ)\n\n" +
				"![[d.png]]\n"),
			Assets: map[string]*model.NoteAssetReplace{
				"./a.png": {ID: 1, URL: "https://cdn.example.com/a-resolved.png"},
				"b.jpg":   {ID: 2, URL: "https://cdn.example.com/b-resolved.jpg"},
			},
		}},
		Log: &logger.TestLogger{},
	})
	require.NoError(t, err)

	nv := pages.PathMap["gallery.md"]
	require.Equal(t, []model.NoteViewImage{
		{URL: "https://cdn.example.com/a-resolved.png", Alt: "first"},
		{URL: "https://cdn.example.com/b-resolved.jpg", Alt: "Second shot"},
		{URL: "https://cdn.example.com/c.webp", Alt: "third", Title: "Caption"},
		{URL: "d.png"},
	}, nv.PartialRenderer.Images())
}

func TestCodeBlocks(t *testing.T) {
	nv := loadSingleNote(t, "```go\nfmt.Println(1)\n```\n\n- item\n\n  ```mychart {\"h\":2}\n  {\"a\": 1}\n  ```\n\n```\nplain\n```\n\n    indented\n")

	all := nv.PartialRenderer.CodeBlocks("")
	require.Len(t, all, 3)

	require.Equal(t, "go", all[0].Lang)
	require.Equal(t, "go", all[0].Info)
	require.Equal(t, "fmt.Println(1)\n", all[0].Content)
	require.Contains(t, all[0].HTML, "Println")

	require.Equal(t, "mychart", all[1].Lang)
	require.Equal(t, `mychart {"h":2}`, all[1].Info)
	require.Equal(t, "{\"a\": 1}\n", all[1].Content)

	require.Empty(t, all[2].Lang)
	require.Equal(t, "plain\n", all[2].Content)

	charts := nv.PartialRenderer.CodeBlocks("mychart")
	require.Len(t, charts, 1)
	require.Equal(t, all[1], charts[0])

	require.Empty(t, nv.PartialRenderer.CodeBlocks("python"))
}
