package tgtd_test

import (
	"testing"
	"trip2g/internal/tgrich"
	"trip2g/internal/tgtd"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/require"
)

func TestToPageBlocksHeadings(t *testing.T) {
	tests := []struct {
		name string
		size int
		want tg.PageBlockClass
	}{
		{"h1", 1, &tg.PageBlockHeading1{Text: &tg.TextPlain{Text: "t"}}},
		{"h2", 2, &tg.PageBlockHeading2{Text: &tg.TextPlain{Text: "t"}}},
		{"h3", 3, &tg.PageBlockHeading3{Text: &tg.TextPlain{Text: "t"}}},
		{"h4", 4, &tg.PageBlockHeading4{Text: &tg.TextPlain{Text: "t"}}},
		{"h5", 5, &tg.PageBlockHeading5{Text: &tg.TextPlain{Text: "t"}}},
		{"h6", 6, &tg.PageBlockHeading6{Text: &tg.TextPlain{Text: "t"}}},
		// The converter only emits 1..6, but a level outside that range must
		// still produce a heading rather than an error: clamping keeps a
		// malformed note publishable, and the level is the only thing lost.
		{"below range clamps to h1", 0, &tg.PageBlockHeading1{Text: &tg.TextPlain{Text: "t"}}},
		{"above range clamps to h6", 9, &tg.PageBlockHeading6{Text: &tg.TextPlain{Text: "t"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tgtd.ToPageBlocks([]tgrich.Block{
				tgrich.Heading(tt.size, tgrich.RichText{Text: "t"}),
			})

			require.NoError(t, err)
			require.Equal(t, []tg.PageBlockClass{tt.want}, got)
		})
	}
}

func TestToPageBlocksStructural(t *testing.T) {
	tests := []struct {
		name  string
		block tgrich.Block
		want  tg.PageBlockClass
	}{
		{
			name:  "paragraph",
			block: tgrich.Paragraph(tgrich.RichText{Text: "hello"}),
			want:  &tg.PageBlockParagraph{Text: &tg.TextPlain{Text: "hello"}},
		},
		{
			name:  "pre keeps its language tag",
			block: tgrich.Pre("x := 1", "go"),
			want:  &tg.PageBlockPreformatted{Text: &tg.TextPlain{Text: "x := 1"}, Language: "go"},
		},
		{
			name:  "divider",
			block: tgrich.Block{Type: tgrich.BlockDivider},
			want:  &tg.PageBlockDivider{},
		},
		{
			// The IR blockquote carries nested blocks, so the -Blocks variant is
			// the faithful target; the plain PageBlockBlockquote takes only a
			// single rich text and would flatten the contents.
			name: "blockquote nests its blocks",
			block: tgrich.Block{
				Type:   tgrich.BlockQuote,
				Blocks: []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "quoted"})},
			},
			want: &tg.PageBlockBlockquoteBlocks{
				Blocks:  []tg.PageBlockClass{&tg.PageBlockParagraph{Text: &tg.TextPlain{Text: "quoted"}}},
				Caption: &tg.TextEmpty{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tgtd.ToPageBlocks([]tgrich.Block{tt.block})

			require.NoError(t, err)
			require.Equal(t, []tg.PageBlockClass{tt.want}, got)
		})
	}
}

// Fold state is a real decision the AST walk made from the callout syntax
// (`> [!x]-` vs `> [!x]+`), so it has to survive the mapping in both directions.
func TestToPageBlocksDetailsFoldState(t *testing.T) {
	for _, open := range []bool{true, false} {
		block := tgrich.Block{
			Type:    tgrich.BlockDetails,
			Summary: &tgrich.RichText{Text: "Summary"},
			IsOpen:  open,
			Blocks:  []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "body"})},
		}

		got, err := tgtd.ToPageBlocks([]tgrich.Block{block})
		require.NoError(t, err)

		details, ok := got[0].(*tg.PageBlockDetails)
		require.True(t, ok)
		require.Equal(t, open, details.Open)
		require.Equal(t, &tg.TextPlain{Text: "Summary"}, details.Title)
		require.Len(t, details.Blocks, 1)
	}
}

func TestToPageBlocksList(t *testing.T) {
	checked := true
	unchecked := false

	got, err := tgtd.ToPageBlocks([]tgrich.Block{{
		Type: tgrich.BlockList,
		Items: []tgrich.ListItem{
			{Blocks: []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "plain"})}},
			{Blocks: []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "done"})}, Checked: &checked},
			{Blocks: []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "todo"})}, Checked: &unchecked},
		},
	}})
	require.NoError(t, err)

	list, ok := got[0].(*tg.PageBlockList)
	require.True(t, ok)
	require.Len(t, list.Items, 3)

	plain, ok := list.Items[0].(*tg.PageListItemBlocks)
	require.True(t, ok)
	require.False(t, plain.Checkbox, "a plain item must not become a checkbox")

	done, ok := list.Items[1].(*tg.PageListItemBlocks)
	require.True(t, ok)
	require.True(t, done.Checkbox)
	require.True(t, done.Checked)

	todo, ok := list.Items[2].(*tg.PageListItemBlocks)
	require.True(t, ok)
	require.True(t, todo.Checkbox, "an unchecked task is still a checkbox")
	require.False(t, todo.Checked)
}

func TestToPageBlocksTable(t *testing.T) {
	got, err := tgtd.ToPageBlocks([]tgrich.Block{{
		Type: tgrich.BlockTable,
		Cells: [][]tgrich.TableCell{
			{
				{Text: tgrich.RichText{Text: "L"}, IsHeader: true, Align: tgrich.AlignLeft},
				{Text: tgrich.RichText{Text: "C"}, IsHeader: true, Align: tgrich.AlignCenter},
				{Text: tgrich.RichText{Text: "R"}, IsHeader: true, Align: tgrich.AlignRight},
			},
			{{Text: tgrich.RichText{Text: "body"}}},
		},
	}})
	require.NoError(t, err)

	table, ok := got[0].(*tg.PageBlockTable)
	require.True(t, ok)
	require.Len(t, table.Rows, 2, "row-major cells become one PageTableRow each")

	head := table.Rows[0].Cells
	require.Len(t, head, 3)

	require.True(t, head[0].Header)
	require.False(t, head[0].AlignCenter)
	require.False(t, head[0].AlignRight, "left is the absence of both alignment flags")

	require.True(t, head[1].AlignCenter)
	require.False(t, head[1].AlignRight)

	require.True(t, head[2].AlignRight)
	require.False(t, head[2].AlignCenter)

	require.False(t, table.Rows[1].Cells[0].Header)
}

// Media rides as a pre-uploaded InputPhoto/InputDocument reference on MTProto,
// not as the HTTPS URL the Bot API ingests server-side. Until the upload step
// exists, a media block must fail loudly rather than send a post with a hole in
// it — the whole reason the block path was chosen over markdown.
func TestToPageBlocksRejectsMedia(t *testing.T) {
	for _, block := range []tgrich.Block{
		{Type: tgrich.BlockPhoto, Photo: &tgrich.Media{URL: "https://example.com/a.jpg"}},
		{Type: tgrich.BlockVideo, Video: &tgrich.Media{URL: "https://example.com/a.mp4"}},
	} {
		_, err := tgtd.ToPageBlocks([]tgrich.Block{block})
		require.ErrorIs(t, err, tgtd.ErrRichMediaUnsupported)
	}
}

func TestToPageBlocksRejectsUnknownType(t *testing.T) {
	_, err := tgtd.ToPageBlocks([]tgrich.Block{{Type: "wat"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "wat")
}

func TestToRichText(t *testing.T) {
	tests := []struct {
		name string
		in   tgrich.RichText
		want tg.RichTextClass
	}{
		{
			name: "empty",
			in:   tgrich.RichText{},
			want: &tg.TextEmpty{},
		},
		{
			name: "plain",
			in:   tgrich.RichText{Text: "a"},
			want: &tg.TextPlain{Text: "a"},
		},
		{
			name: "bold",
			in:   tgrich.RichText{Text: "a", Bold: true},
			want: &tg.TextBold{Text: &tg.TextPlain{Text: "a"}},
		},
		{
			name: "code maps to the fixed-width node",
			in:   tgrich.RichText{Text: "a", Code: true},
			want: &tg.TextFixed{Text: &tg.TextPlain{Text: "a"}},
		},
		{
			name: "marked",
			in:   tgrich.RichText{Text: "a", Marked: true},
			want: &tg.TextMarked{Text: &tg.TextPlain{Text: "a"}},
		},
		{
			name: "strike",
			in:   tgrich.RichText{Text: "a", Strike: true},
			want: &tg.TextStrike{Text: &tg.TextPlain{Text: "a"}},
		},
		{
			name: "underline",
			in:   tgrich.RichText{Text: "a", Underline: true},
			want: &tg.TextUnderline{Text: &tg.TextPlain{Text: "a"}},
		},
		{
			// Marks nest rather than combine, and the order is fixed so the same
			// RichText always produces the same tree — matching the JSON encoder
			// in tgrich, which wraps bold outermost.
			name: "bold italic nests with bold outermost",
			in:   tgrich.RichText{Text: "a", Bold: true, Italic: true},
			want: &tg.TextBold{Text: &tg.TextItalic{Text: &tg.TextPlain{Text: "a"}}},
		},
		{
			name: "link",
			in:   tgrich.RichText{Text: "a", URL: "https://example.com"},
			want: &tg.TextURL{Text: &tg.TextPlain{Text: "a"}, URL: "https://example.com"},
		},
		{
			// A mark outside a link, matching the JSON encoder's order: the link
			// is applied first, then the marks wrap outwards.
			name: "bold link",
			in:   tgrich.RichText{Text: "a", URL: "https://example.com", Bold: true},
			want: &tg.TextBold{Text: &tg.TextURL{Text: &tg.TextPlain{Text: "a"}, URL: "https://example.com"}},
		},
		{
			name: "anchor carries its name and no content",
			in:   tgrich.RichText{Anchor: "intro"},
			want: &tg.TextAnchor{Name: "intro", Text: &tg.TextEmpty{}},
		},
		{
			name: "children concatenate",
			in: tgrich.RichText{Children: []tgrich.RichText{
				{Text: "a"},
				{Text: "b", Bold: true},
			}},
			want: &tg.TextConcat{Texts: []tg.RichTextClass{
				&tg.TextPlain{Text: "a"},
				&tg.TextBold{Text: &tg.TextPlain{Text: "b"}},
			}},
		},
		{
			name: "own text leads its children",
			in: tgrich.RichText{
				Text:     "a",
				Children: []tgrich.RichText{{Text: "b"}},
			},
			want: &tg.TextConcat{Texts: []tg.RichTextClass{
				&tg.TextPlain{Text: "a"},
				&tg.TextPlain{Text: "b"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tgtd.ToRichText(tt.in))
		})
	}
}

// Every RichTextClass field on a page block is required, not optional: a nil
// one fails to encode and takes the whole message with it, and the failure
// surfaces only at send time as an opaque codec error naming a field index.
// This walks a document exercising every block type and asserts the tree
// encodes, which is the only check that covers fields nothing else reads.
func TestToPageBlocksEncodes(t *testing.T) {
	checked := true

	blocks := []tgrich.Block{
		tgrich.Heading(1, tgrich.RichText{Text: "Title"}),
		tgrich.Paragraph(tgrich.RichText{Text: "body", Bold: true}),
		tgrich.Pre("code", "go"),
		{Type: tgrich.BlockDivider},
		{
			Type:   tgrich.BlockQuote,
			Blocks: []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "quoted"})},
		},
		{
			Type:    tgrich.BlockDetails,
			Summary: &tgrich.RichText{Text: "Summary"},
			Blocks:  []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "hidden"})},
		},
		{
			Type: tgrich.BlockList,
			Items: []tgrich.ListItem{
				{Blocks: []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "one"})}},
				{Blocks: []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Text: "two"})}, Checked: &checked},
			},
		},
		{
			Type: tgrich.BlockTable,
			Cells: [][]tgrich.TableCell{
				{{Text: tgrich.RichText{Text: "h"}, IsHeader: true, Align: tgrich.AlignCenter}},
				{{Text: tgrich.RichText{Text: "c"}}},
			},
		},
	}

	mapped, err := tgtd.ToPageBlocks(blocks)
	require.NoError(t, err)
	require.Len(t, mapped, len(blocks))

	message := &tg.InputRichMessage{Blocks: mapped}

	buf := &bin.Buffer{}
	require.NoError(t, message.Encode(buf), "the whole block tree must encode")
	require.NotEmpty(t, buf.Buf)
}
