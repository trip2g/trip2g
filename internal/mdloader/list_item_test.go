package mdloader_test

import (
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestListItemTextAndHTML(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []model.NoteViewListItem
	}{
		{
			name: "quotes and punctuation mid-line",
			src:  "- Where do we see \"perfect bodies\" and \"perfect faces\" every day?\n",
			want: []model.NoteViewListItem{{
				Text: `Where do we see "perfect bodies" and "perfect faces" every day?`,
				HTML: `Where do we see &quot;perfect bodies&quot; and &quot;perfect faces&quot; every day?`,
			}},
		},
		{
			name: "dash mid-line keeps the tail",
			src:  "- plastic surgery — a medical operation to change how your body or face looks\n",
			want: []model.NoteViewListItem{{
				Text: "plastic surgery — a medical operation to change how your body or face looks",
				HTML: "plastic surgery — a medical operation to change how your body or face looks",
			}},
		},
		{
			name: "emphasis, strong and code",
			src:  "- half *em* and **strong** and `code`\n",
			want: []model.NoteViewListItem{{
				Text: "half em and strong and code",
				HTML: "half <em>em</em> and <strong>strong</strong> and <code>code</code>",
			}},
		},
		{
			name: "link item",
			src:  "- [there](https://example.com)\n",
			want: []model.NoteViewListItem{{
				Text: "there",
				URL:  "https://example.com",
				HTML: `<a href="https://example.com">there</a>`,
			}},
		},
		{
			name: "link inside text",
			src:  "- moved [there](https://example.com) today\n",
			want: []model.NoteViewListItem{{
				Text: "moved there today",
				URL:  "https://example.com",
				HTML: `moved <a href="https://example.com">there</a> today`,
			}},
		},
		{
			name: "wikilink resolves as on the page",
			src:  "- see [[Other|the other]]\n",
			want: []model.NoteViewListItem{{
				Text: "see the other",
				HTML: `see <a class="paywall paywall:core" data-pid="0" href="/other">the other</a>`,
			}},
		},
		{
			name: "obsidian task mark is not part of the content",
			src:  "- [/] half **done**\n",
			want: []model.NoteViewListItem{{
				Text:     "half done",
				HTML:     "half <strong>done</strong>",
				Task:     "done",
				TaskMark: "/",
			}},
		},
		{
			name: "task mark the page escapes",
			src:  "- [>] moved [there](https://example.com)\n",
			want: []model.NoteViewListItem{{
				Text:     "moved there",
				URL:      "https://example.com",
				HTML:     `moved <a href="https://example.com">there</a>`,
				Task:     "done",
				TaskMark: ">",
			}},
		},
		{
			name: "checkbox is not part of the content",
			src:  "- [ ] open *now*\n",
			want: []model.NoteViewListItem{{
				Text:     "open now",
				HTML:     "open <em>now</em>",
				Task:     "todo",
				TaskMark: " ",
			}},
		},
		{
			name: "bare task mark",
			src:  "- [/]\n",
			want: []model.NoteViewListItem{{
				Task:     "done",
				TaskMark: "/",
			}},
		},
		{
			name: "nested list stays in children",
			src:  "- parent *p*\n  - child **c**\n",
			want: []model.NoteViewListItem{{
				Text: "parent p",
				HTML: "parent <em>p</em>",
				Children: []model.NoteViewListItem{{
					Text: "child c",
					HTML: "child <strong>c</strong>",
				}},
			}},
		},
		{
			name: "continuation line",
			src:  "- two\n  lines\n",
			want: []model.NoteViewListItem{{
				Text: "two lines",
				HTML: "two<br>\nlines",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pages, err := mdloader.Load(mdloader.Options{
				Sources: []mdloader.SourceFile{
					{Path: "note.md", Content: []byte(tt.src)},
					{Path: "Other.md", Content: []byte("Other note.")},
				},
				Log: &logger.TestLogger{},
			})
			require.NoError(t, err)

			list, ok := pages.PathMap["note.md"].PartialRenderer.FirstList().(*model.NoteViewList)
			require.True(t, ok)
			require.Equal(t, tt.want, list.Items)
		})
	}
}
