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
			want: `{"type":"bold","text":"hi"}`,
		},
		{
			// Marks nest rather than combine, innermost first, in a fixed order
			// so the same value always produces the same bytes.
			name: "every mark at once nests outwards",
			text: tgrich.RichText{
				Text: "x", Bold: true, Italic: true, Underline: true,
				Strike: true, Marked: true, Code: true,
			},
			want: `{"type":"bold","text":{"type":"italic","text":{"type":"underline",` +
				`"text":{"type":"strikethrough","text":{"type":"marked",` +
				`"text":{"type":"code","text":"x"}}}}}}`,
		},
		{
			name: "link run",
			text: tgrich.RichText{Text: "site", URL: "https://example.com"},
			want: `{"type":"url","text":"site","url":"https://example.com"}`,
		},
		{
			// A mark wraps the link, not the other way round.
			name: "bold link",
			text: tgrich.RichText{Text: "site", URL: "https://example.com", Bold: true},
			want: `{"type":"bold","text":{"type":"url","text":"site","url":"https://example.com"}}`,
		},
		{
			name: "children concatenate as a bare array",
			text: tgrich.RichText{Children: []tgrich.RichText{
				{Text: "a"},
				{Text: "b", Bold: true},
			}},
			want: `["a",{"type":"bold","text":"b"}]`,
		},
		{
			// An anchor is a target and carries no text of its own.
			name: "anchor",
			text: tgrich.RichText{Anchor: "section"},
			want: `{"type":"anchor","name":"section"}`,
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
			// There is no anchor field: measured, the server echoes back neither
			// anchor, name nor id on a heading.
			name:  "heading keeps the flat text/size shape",
			block: tgrich.Heading(2, tgrich.RichText{Text: "Title"}),
			want:  `{"type":"heading","text":"Title","size":2}`,
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
			// A list carries no ordering: measured, every spelling of it is
			// ignored and the server labels every item with a bullet.
			name: "list carries only its items",
			block: tgrich.Block{
				Type:  tgrich.BlockList,
				Items: []tgrich.ListItem{{Blocks: []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "a"})}}},
			},
			want: `{"type":"list","items":[{"blocks":[{"type":"paragraph","text":"a"}]}]}`,
		},
		{
			name: "quote nests blocks",
			block: tgrich.Block{
				Type:   tgrich.BlockQuote,
				Blocks: []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "quoted"})},
			},
			want: `{"type":"blockquote","blocks":[{"type":"paragraph","text":"quoted"}]}`,
		},
		{
			// Collapsed is the default, so a folded section sends no fold field
			// at all; is_open is the only spelling that expands one.
			name: "collapsed details carries a summary and no fold field",
			block: tgrich.Block{
				Type:    tgrich.BlockDetails,
				Summary: &tgrich.RichText{Text: "Note"},
				Blocks:  []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "body"})},
			},
			want: `{"type":"details","summary":"Note",` +
				`"blocks":[{"type":"paragraph","text":"body"}]}`,
		},
		{
			name: "expanded details carries is_open",
			block: tgrich.Block{
				Type:    tgrich.BlockDetails,
				Summary: &tgrich.RichText{Text: "Note"},
				IsOpen:  true,
				Blocks:  []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "body"})},
			},
			want: `{"type":"details","summary":"Note","is_open":true,` +
				`"blocks":[{"type":"paragraph","text":"body"}]}`,
		},
		{
			// The code rides in text, not in a field of its own.
			name:  "pre with a language",
			block: tgrich.Pre("fmt.Println()", "go"),
			want:  `{"type":"pre","text":"fmt.Println()","language":"go"}`,
		},
		{
			name:  "pre without a language omits it",
			block: tgrich.Pre("x", ""),
			want:  `{"type":"pre","text":"x"}`,
		},
		{
			// Cells are a row-major grid on the block itself. There is no column
			// descriptor, so alignment is per cell, and the header row is marked
			// by is_header on its cells rather than by a flag on the row.
			name: "table is a row-major cell grid",
			block: tgrich.Block{
				Type: tgrich.BlockTable,
				Cells: [][]tgrich.TableCell{
					{{Text: tgrich.RichText{Text: "a"}, IsHeader: true, Align: tgrich.AlignLeft}},
					{{Text: tgrich.RichText{Text: "1"}, Align: tgrich.AlignLeft}},
				},
			},
			want: `{"type":"table","cells":[` +
				`[{"text":"a","is_header":true,"align":"left"}],` +
				`[{"text":"1","align":"left"}]]}`,
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
				Caption: &tgrich.RichText{Text: "cap"},
			},
			want: `{"type":"video","video":{"url":"https://example.com/a.mp4"},"caption":"cap"}`,
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
