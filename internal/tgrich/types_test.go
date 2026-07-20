package tgrich_test

import (
	"encoding/json"
	"testing"
	"trip2g/internal/tgrich"

	"github.com/stretchr/testify/require"
)

func TestRichTextMarshal(t *testing.T) {
	tests := []struct {
		name string
		text tgrich.RichText
		want string
	}{
		{
			name: "plain text marshals as a bare string",
			text: tgrich.RichText{Text: "h1"},
			want: `"h1"`,
		},
		{
			name: "empty text marshals as an empty string",
			text: tgrich.RichText{},
			want: `""`,
		},
		{
			name: "bold run",
			text: tgrich.RichText{Text: "hi", Bold: true},
			want: `{"text":"hi","bold":true}`,
		},
		{
			name: "every mark at once",
			text: tgrich.RichText{
				Text: "x", Bold: true, Italic: true, Underline: true,
				Strike: true, Marked: true, Code: true,
			},
			want: `{"text":"x","bold":true,"italic":true,"underline":true,` +
				`"strike":true,"marked":true,"code":true}`,
		},
		{
			name: "link run",
			text: tgrich.RichText{Text: "site", URL: "https://example.com"},
			want: `{"text":"site","url":"https://example.com"}`,
		},
		{
			name: "children concatenation drops the empty text field",
			text: tgrich.RichText{Children: []tgrich.RichText{
				{Text: "a"},
				{Text: "b", Bold: true},
			}},
			want: `{"children":["a",{"text":"b","bold":true}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.text)
			require.NoError(t, err)
			require.JSONEq(t, tt.want, string(got))
		})
	}
}

func TestBlockMarshal(t *testing.T) {
	checked := true

	tests := []struct {
		name  string
		block tgrich.Block
		want  string
	}{
		{
			name:  "heading keeps the flat text/size shape",
			block: tgrich.Heading(2, tgrich.RichText{Text: "Title"}, "title"),
			want:  `{"type":"heading","text":"Title","size":2,"anchor":"title"}`,
		},
		{
			name:  "heading without an anchor omits it",
			block: tgrich.Heading(1, tgrich.RichText{Text: "Title"}, ""),
			want:  `{"type":"heading","text":"Title","size":1}`,
		},
		{
			name:  "paragraph",
			block: tgrich.Paragraph(tgrich.RichText{Text: "body"}),
			want:  `{"type":"paragraph","text":"body"}`,
		},
		{
			name: "unordered list with a task item",
			block: tgrich.Block{
				Type: tgrich.BlockList,
				Items: []tgrich.ListItem{
					{Blocks: []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "todo"})}, Checked: &checked},
				},
			},
			want: `{"type":"list","items":[{"blocks":[{"type":"paragraph","text":"todo"}],"checked":true}]}`,
		},
		{
			name: "ordered list carries ordered and start",
			block: tgrich.Block{
				Type:    tgrich.BlockList,
				Ordered: true,
				Start:   3,
				Items:   []tgrich.ListItem{{Blocks: []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "a"})}}},
			},
			want: `{"type":"list","ordered":true,"start":3,` +
				`"items":[{"blocks":[{"type":"paragraph","text":"a"}]}]}`,
		},
		{
			name: "quote nests blocks",
			block: tgrich.Block{
				Type:   tgrich.BlockQuote,
				Blocks: []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "quoted"})},
			},
			want: `{"type":"quote","blocks":[{"type":"paragraph","text":"quoted"}]}`,
		},
		{
			name: "collapsed collapsible carries title and fold state",
			block: tgrich.Block{
				Type:      tgrich.BlockCollapsible,
				Title:     &tgrich.RichText{Text: "Note"},
				Collapsed: true,
				Blocks:    []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "body"})},
			},
			want: `{"type":"collapsible","title":"Note","collapsed":true,` +
				`"blocks":[{"type":"paragraph","text":"body"}]}`,
		},
		{
			name:  "code with a language",
			block: tgrich.Code("fmt.Println()", "go"),
			want:  `{"type":"code","code":"fmt.Println()","language":"go"}`,
		},
		{
			name:  "code without a language omits it",
			block: tgrich.Code("x", ""),
			want:  `{"type":"code","code":"x"}`,
		},
		{
			name: "table with per-column alignment",
			block: tgrich.Block{
				Type:    tgrich.BlockTable,
				Columns: []tgrich.TableColumn{{Align: tgrich.AlignLeft}, {Align: tgrich.AlignRight}, {}},
				Rows: []tgrich.TableRow{
					{Header: true, Cells: []tgrich.TableCell{{Text: tgrich.RichText{Text: "a"}}}},
					{Cells: []tgrich.TableCell{{Text: tgrich.RichText{Text: "1"}}}},
				},
			},
			want: `{"type":"table","columns":[{"align":"left"},{"align":"right"},{}],` +
				`"rows":[{"header":true,"cells":[{"text":"a"}]},{"cells":[{"text":"1"}]}]}`,
		},
		{
			name:  "divider carries nothing but its type",
			block: tgrich.Block{Type: tgrich.BlockDivider},
			want:  `{"type":"divider"}`,
		},
		{
			name: "photo nests the media under its own type name",
			block: tgrich.Block{
				Type:  tgrich.BlockPhoto,
				Photo: &tgrich.Media{URL: "https://example.com/a.png"},
			},
			want: `{"type":"photo","photo":{"url":"https://example.com/a.png"}}`,
		},
		{
			name: "video with a caption",
			block: tgrich.Block{
				Type:    tgrich.BlockVideo,
				Video:   &tgrich.Media{URL: "https://example.com/a.mp4"},
				Caption: &tgrich.Caption{Text: tgrich.RichText{Text: "cap"}},
			},
			want: `{"type":"video","video":{"url":"https://example.com/a.mp4"},"caption":{"text":"cap"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.block)
			require.NoError(t, err)
			require.JSONEq(t, tt.want, string(got))
		})
	}
}

func TestRequestMarshal(t *testing.T) {
	req := tgrich.Request{
		ChatID: 42,
		RichMessage: tgrich.InputRichMessage{
			Blocks:              []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "hi"})},
			SkipEntityDetection: true,
		},
	}

	got, err := json.Marshal(req)
	require.NoError(t, err)
	require.JSONEq(t, `{"chat_id":42,"rich_message":{`+
		`"blocks":[{"type":"paragraph","text":"hi"}],"skip_entity_detection":true}}`, string(got))
}
