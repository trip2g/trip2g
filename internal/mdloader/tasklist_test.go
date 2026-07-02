package mdloader_test

import (
	"strings"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func loadSingleNote(t *testing.T, content string) *model.NoteView {
	t.Helper()

	pages, err := mdloader.Load(mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path:    "note.md",
			Content: []byte(content),
		}},
		Log: &logger.TestLogger{},
	})
	require.NoError(t, err)

	nv := pages.PathMap["note.md"]
	require.NotNil(t, nv)
	return nv
}

func TestTaskListExtraction(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []model.NoteViewTaskItem
	}{
		{
			name: "no tasks",
			src:  "# Hello\n\nSome text.\n",
			want: nil,
		},
		{
			name: "one unchecked",
			src:  "- [ ] Buy milk\n",
			want: []model.NoteViewTaskItem{
				{Index: 0, Line: 1, Checked: false, Text: "- [ ] Buy milk"},
			},
		},
		{
			name: "one checked",
			src:  "- [x] Done\n",
			want: []model.NoteViewTaskItem{
				{Index: 0, Line: 1, Checked: true, Text: "- [x] Done"},
			},
		},
		{
			name: "checked uppercase X",
			src:  "- [X] Done\n",
			want: []model.NoteViewTaskItem{
				{Index: 0, Line: 1, Checked: true, Text: "- [X] Done"},
			},
		},
		{
			name: "mixed tasks",
			src:  "- [ ] Buy milk\n- [x] Done\n- [ ] Another\n",
			want: []model.NoteViewTaskItem{
				{Index: 0, Line: 1, Checked: false, Text: "- [ ] Buy milk"},
				{Index: 1, Line: 2, Checked: true, Text: "- [x] Done"},
				{Index: 2, Line: 3, Checked: false, Text: "- [ ] Another"},
			},
		},
		{
			name: "skips fenced code block with task-like content",
			src:  "```\n- [ ] not a task\n```\n\n- [ ] real task\n",
			want: []model.NoteViewTaskItem{
				{Index: 0, Line: 5, Checked: false, Text: "- [ ] real task"},
			},
		},
		{
			name: "skips tilde fenced code block",
			src:  "~~~\n- [ ] not a task\n~~~\n\n- [x] real\n",
			want: []model.NoteViewTaskItem{
				{Index: 0, Line: 5, Checked: true, Text: "- [x] real"},
			},
		},
		{
			name: "duplicate lines get distinct line numbers",
			src:  "- [ ] same\n- [ ] same\n",
			want: []model.NoteViewTaskItem{
				{Index: 0, Line: 1, Checked: false, Text: "- [ ] same"},
				{Index: 1, Line: 2, Checked: false, Text: "- [ ] same"},
			},
		},
		{
			name: "windows line endings",
			src:  "- [ ] one\r\n- [x] two\r\n",
			want: []model.NoteViewTaskItem{
				{Index: 0, Line: 1, Checked: false, Text: "- [ ] one"},
				{Index: 1, Line: 2, Checked: true, Text: "- [x] two"},
			},
		},
		{
			name: "frontmatter shifts line numbers",
			src:  "---\ntitle: Test\n---\n\n# Head\n\n- [ ] task after frontmatter\n",
			want: []model.NoteViewTaskItem{
				{Index: 0, Line: 7, Checked: false, Text: "- [ ] task after frontmatter"},
			},
		},
		{
			name: "task-like text outside list is not a task",
			src:  "some text [ ] not a task\n",
			want: nil,
		},
		{
			name: "asterisk and plus bullets",
			src:  "* [ ] asterisk\n\n+ [x] plus\n",
			want: []model.NoteViewTaskItem{
				{Index: 0, Line: 1, Checked: false, Text: "* [ ] asterisk"},
				{Index: 1, Line: 3, Checked: true, Text: "+ [x] plus"},
			},
		},
		{
			name: "nested tasks",
			src:  "- [ ] parent\n    - [x] child\n",
			want: []model.NoteViewTaskItem{
				{Index: 0, Line: 1, Checked: false, Text: "- [ ] parent"},
				{Index: 1, Line: 2, Checked: true, Text: "    - [x] child"},
			},
		},
		{
			name: "loose list with blank lines",
			src:  "- [ ] first\n\n- [x] second\n",
			want: []model.NoteViewTaskItem{
				{Index: 0, Line: 1, Checked: false, Text: "- [ ] first"},
				{Index: 1, Line: 3, Checked: true, Text: "- [x] second"},
			},
		},
		{
			name: "task inside blockquote",
			src:  "> - [ ] quoted task\n",
			want: []model.NoteViewTaskItem{
				{Index: 0, Line: 1, Checked: false, Text: "> - [ ] quoted task"},
			},
		},
		{
			name: "multi-line task item maps to marker line",
			src:  "- [ ] first line\n  continuation\n- [x] next\n",
			want: []model.NoteViewTaskItem{
				{Index: 0, Line: 1, Checked: false, Text: "- [ ] first line"},
				{Index: 1, Line: 3, Checked: true, Text: "- [x] next"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nv := loadSingleNote(t, tt.src)

			require.Equal(t, tt.want, nv.TaskList)
			require.Equal(t, len(tt.want) > 0, nv.HasTaskListItems())

			// Mapping invariant: the rendered HTML must contain exactly one
			// checkbox per extracted item, in the same document order, so the
			// frontend can map DOM checkbox i → TaskList[i].
			htmlCount := strings.Count(string(nv.HTML), `type="checkbox"`)
			require.Equal(t, len(nv.TaskList), htmlCount,
				"rendered checkbox count must equal extracted task count")
		})
	}
}

// TestTaskListHTMLOrderMatchesExtraction locks the invariant that checked
// states appear in the rendered HTML in the same order as TaskList entries.
func TestTaskListHTMLOrderMatchesExtraction(t *testing.T) {
	nv := loadSingleNote(t, "- [x] a\n- [ ] b\n\nText.\n\n1. [ ] c\n2. [x] d\n")

	html := string(nv.HTML)
	var htmlChecked []bool
	for _, part := range strings.Split(html, "<input")[1:] {
		tag := part[:strings.Index(part, ">")]
		if !strings.Contains(tag, `type="checkbox"`) {
			continue
		}
		htmlChecked = append(htmlChecked, strings.Contains(tag, "checked"))
	}

	var extracted []bool
	for _, item := range nv.TaskList {
		extracted = append(extracted, item.Checked)
	}

	require.Equal(t, extracted, htmlChecked)
}
