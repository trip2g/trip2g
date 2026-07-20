package tgrich_test

import (
	"strings"
	"testing"
	"trip2g/internal/tgrich"

	"github.com/stretchr/testify/require"
)

// para builds a paragraph carrying s.
func para(s string) tgrich.Block {
	return tgrich.Paragraph(tgrich.RichText{Text: s})
}

// nest wraps a paragraph in depth-1 quote levels, so the returned block tree
// has exactly `depth` levels.
func nest(depth int) tgrich.Block {
	b := para("x")
	for range depth - 1 {
		b = tgrich.Block{Type: tgrich.BlockQuote, Blocks: []tgrich.Block{b}}
	}
	return b
}

func repeatBlocks(n int) []tgrich.Block {
	blocks := make([]tgrich.Block, n)
	for i := range blocks {
		blocks[i] = para("x")
	}
	return blocks
}

func repeatMedia(n int) []tgrich.Block {
	blocks := make([]tgrich.Block, n)
	for i := range blocks {
		blocks[i] = tgrich.Block{Type: tgrich.BlockPhoto, Photo: &tgrich.Media{URL: "https://e.com/a.png"}}
	}
	return blocks
}

func TestInputRichMessageValidateContentSource(t *testing.T) {
	tests := []struct {
		name    string
		msg     tgrich.InputRichMessage
		wantErr error
	}{
		{
			name:    "no content source at all",
			msg:     tgrich.InputRichMessage{},
			wantErr: tgrich.ErrNoContentSource,
		},
		{
			name:    "blocks only",
			msg:     tgrich.InputRichMessage{Blocks: []tgrich.Block{para("x")}},
			wantErr: nil,
		},
		{
			name:    "markdown only is refused, this path is blocks-only",
			msg:     tgrich.InputRichMessage{Markdown: "hi"},
			wantErr: tgrich.ErrUnsupportedContentSource,
		},
		{
			name:    "html only is refused, this path is blocks-only",
			msg:     tgrich.InputRichMessage{HTML: "<b>hi</b>"},
			wantErr: tgrich.ErrUnsupportedContentSource,
		},
		{
			name: "blocks plus markdown",
			msg: tgrich.InputRichMessage{
				Blocks:   []tgrich.Block{para("x")},
				Markdown: "hi",
			},
			wantErr: tgrich.ErrUnsupportedContentSource,
		},
		{
			name:    "markdown plus html",
			msg:     tgrich.InputRichMessage{Markdown: "hi", HTML: "<b>hi</b>"},
			wantErr: tgrich.ErrUnsupportedContentSource,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate(tgrich.DefaultLimits())
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestInputRichMessageValidateLimits(t *testing.T) {
	limits := tgrich.DefaultLimits()

	tests := []struct {
		name    string
		blocks  []tgrich.Block
		wantErr error
	}{
		{
			name:    "text at the length limit",
			blocks:  []tgrich.Block{para(strings.Repeat("a", limits.TextLength))},
			wantErr: nil,
		},
		{
			name:    "text one unit past the length limit",
			blocks:  []tgrich.Block{para(strings.Repeat("a", limits.TextLength+1))},
			wantErr: tgrich.ErrTextTooLong,
		},
		{
			name:    "block count at the limit",
			blocks:  repeatBlocks(limits.MaxBlocks),
			wantErr: nil,
		},
		{
			name:    "block count one past the limit",
			blocks:  repeatBlocks(limits.MaxBlocks + 1),
			wantErr: tgrich.ErrTooManyBlocks,
		},
		{
			name:    "media count at the limit",
			blocks:  repeatMedia(limits.MaxMedia),
			wantErr: nil,
		},
		{
			name:    "media count one past the limit",
			blocks:  repeatMedia(limits.MaxMedia + 1),
			wantErr: tgrich.ErrTooManyMedia,
		},
		{
			name:    "depth at the limit",
			blocks:  []tgrich.Block{nest(limits.MaxDepth)},
			wantErr: nil,
		},
		{
			name:    "depth one past the limit",
			blocks:  []tgrich.Block{nest(limits.MaxDepth + 1)},
			wantErr: tgrich.ErrTooDeep,
		},
		{
			name:    "table columns at the limit",
			blocks:  []tgrich.Block{tableWithCols(limits.MaxTableCols)},
			wantErr: nil,
		},
		{
			name:    "table columns one past the limit",
			blocks:  []tgrich.Block{tableWithCols(limits.MaxTableCols + 1)},
			wantErr: tgrich.ErrTooManyTableCols,
		},
		{
			name:    "unknown block type",
			blocks:  []tgrich.Block{{Type: "nonsense"}},
			wantErr: tgrich.ErrUnknownBlockType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tgrich.InputRichMessage{Blocks: tt.blocks, SkipEntityDetection: true}
			err := msg.Validate(limits)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func tableWithCols(n int) tgrich.Block {
	cols := make([]tgrich.TableColumn, n)
	cells := make([]tgrich.TableCell, n)
	for i := range cols {
		cells[i] = tgrich.TableCell{Text: tgrich.RichText{Text: "c"}}
	}
	return tgrich.Block{
		Type:    tgrich.BlockTable,
		Columns: cols,
		Rows:    []tgrich.TableRow{{Header: true, Cells: cells}},
	}
}

// List items do not count towards the block limit: a 501-item list is one
// block (measured, see docs/dev/telegram_rich.md).
func TestListItemsDoNotCountAsBlocks(t *testing.T) {
	limits := tgrich.DefaultLimits()

	items := make([]tgrich.ListItem, limits.MaxBlocks+1)
	for i := range items {
		items[i] = tgrich.ListItem{Blocks: []tgrich.Block{para("x")}}
	}

	msg := tgrich.InputRichMessage{Blocks: []tgrich.Block{{Type: tgrich.BlockList, Items: items}}}
	require.NoError(t, msg.Validate(limits))
}

func TestTextLengthCountsUTF16Units(t *testing.T) {
	limits := tgrich.Limits{TextLength: 2, MaxBlocks: 10, MaxMedia: 10, MaxDepth: 10, MaxTableCols: 10}

	// One astral-plane rune is two UTF-16 units.
	require.NoError(t, tgrich.InputRichMessage{Blocks: []tgrich.Block{para("😀")}}.Validate(limits))
	require.ErrorIs(t,
		tgrich.InputRichMessage{Blocks: []tgrich.Block{para("😀a")}}.Validate(limits),
		tgrich.ErrTextTooLong)
}

func TestDefaultLimitsMatchAppConfig(t *testing.T) {
	l := tgrich.DefaultLimits()
	require.Equal(t, 32768, l.TextLength)
	require.Equal(t, 500, l.MaxBlocks)
	require.Equal(t, 50, l.MaxMedia)
	require.Equal(t, 16, l.MaxDepth)
	require.Equal(t, 20, l.MaxTableCols)
}

func TestLimitsFromAppConfig(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]interface{}
		want   tgrich.Limits
	}{
		{
			name:   "nil config keeps every default",
			config: nil,
			want:   tgrich.DefaultLimits(),
		},
		{
			name:   "config predating rich messages keeps every default",
			config: map[string]interface{}{"caption_length_limit_default": float64(1024)},
			want:   tgrich.DefaultLimits(),
		},
		{
			name: "advertised values win",
			config: map[string]interface{}{
				tgrich.KeyTextLength:   float64(4096),
				tgrich.KeyMaxBlocks:    float64(100),
				tgrich.KeyMaxMedia:     float64(10),
				tgrich.KeyMaxDepth:     float64(4),
				tgrich.KeyMaxTableCols: float64(8),
			},
			want: tgrich.Limits{TextLength: 4096, MaxBlocks: 100, MaxMedia: 10, MaxDepth: 4, MaxTableCols: 8},
		},
		{
			name: "non-numeric and non-positive entries fall back",
			config: map[string]interface{}{
				tgrich.KeyTextLength: "lots",
				tgrich.KeyMaxBlocks:  float64(0),
				tgrich.KeyMaxMedia:   float64(-1),
			},
			want: tgrich.DefaultLimits(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tgrich.LimitsFromAppConfig(tt.config))
		})
	}
}

// Per-variant validation. Before this existed a paragraph could carry code and
// media fields at once, and a photo could carry no media at all: the walk only
// checked that the discriminator was a known string.
func TestBlockValidateVariants(t *testing.T) {
	photo := &tgrich.Media{URL: "https://e.com/a.png"}
	body := []tgrich.Block{para("x")}
	text := tgrich.RichText{Text: "x"}

	tests := []struct {
		name    string
		block   tgrich.Block
		wantErr error
	}{
		{
			name:  "heading with size and anchor",
			block: tgrich.Heading(2, text, "intro"),
		},
		{
			name:    "heading without size",
			block:   tgrich.Block{Type: tgrich.BlockHeading, Text: &text},
			wantErr: tgrich.ErrBlockMissingField,
		},
		{
			name:    "heading without text",
			block:   tgrich.Block{Type: tgrich.BlockHeading, Size: 1},
			wantErr: tgrich.ErrBlockMissingField,
		},
		{
			name:    "heading below the level range",
			block:   tgrich.Block{Type: tgrich.BlockHeading, Text: &text, Size: 0},
			wantErr: tgrich.ErrBlockMissingField,
		},
		{
			name:    "heading past the level range",
			block:   tgrich.Block{Type: tgrich.BlockHeading, Text: &text, Size: 7},
			wantErr: tgrich.ErrHeadingSize,
		},
		{
			name:  "paragraph",
			block: para("x"),
		},
		{
			name:    "paragraph without text",
			block:   tgrich.Block{Type: tgrich.BlockParagraph},
			wantErr: tgrich.ErrBlockMissingField,
		},
		{
			name:    "paragraph carrying code",
			block:   tgrich.Block{Type: tgrich.BlockParagraph, Text: &text, Code: "x := 1"},
			wantErr: tgrich.ErrBlockForbiddenField,
		},
		{
			name:    "paragraph carrying media",
			block:   tgrich.Block{Type: tgrich.BlockParagraph, Text: &text, Photo: photo},
			wantErr: tgrich.ErrBlockForbiddenField,
		},
		{
			name:    "paragraph carrying a heading size",
			block:   tgrich.Block{Type: tgrich.BlockParagraph, Text: &text, Size: 1},
			wantErr: tgrich.ErrBlockForbiddenField,
		},
		{
			name:  "list with items",
			block: tgrich.Block{Type: tgrich.BlockList, Ordered: true, Start: 3, Items: []tgrich.ListItem{{Blocks: body}}},
		},
		{
			name:    "list without items",
			block:   tgrich.Block{Type: tgrich.BlockList},
			wantErr: tgrich.ErrBlockMissingField,
		},
		{
			name:  "quote with a title",
			block: tgrich.Block{Type: tgrich.BlockQuote, Title: &text, Blocks: body},
		},
		{
			name:    "quote without body blocks",
			block:   tgrich.Block{Type: tgrich.BlockQuote, Title: &text},
			wantErr: tgrich.ErrBlockMissingField,
		},
		{
			name:    "quote claiming to be collapsed",
			block:   tgrich.Block{Type: tgrich.BlockQuote, Blocks: body, Collapsed: true},
			wantErr: tgrich.ErrBlockForbiddenField,
		},
		{
			name:  "collapsible",
			block: tgrich.Block{Type: tgrich.BlockCollapsible, Title: &text, Blocks: body, Collapsed: true},
		},
		{
			name:    "collapsible without body blocks",
			block:   tgrich.Block{Type: tgrich.BlockCollapsible, Title: &text},
			wantErr: tgrich.ErrBlockMissingField,
		},
		{
			name:  "code with a language",
			block: tgrich.Code("x := 1", "go"),
		},
		{
			name:    "code without code",
			block:   tgrich.Code("", "go"),
			wantErr: tgrich.ErrBlockMissingField,
		},
		{
			name:    "code carrying text",
			block:   tgrich.Block{Type: tgrich.BlockCode, Code: "x := 1", Text: &text},
			wantErr: tgrich.ErrBlockForbiddenField,
		},
		{
			name:  "table",
			block: tableWithCols(2),
		},
		{
			name:    "table without rows",
			block:   tgrich.Block{Type: tgrich.BlockTable, Columns: []tgrich.TableColumn{{Align: tgrich.AlignLeft}}},
			wantErr: tgrich.ErrBlockMissingField,
		},
		{
			name:  "divider",
			block: tgrich.Block{Type: tgrich.BlockDivider},
		},
		{
			name:    "divider carrying text",
			block:   tgrich.Block{Type: tgrich.BlockDivider, Text: &text},
			wantErr: tgrich.ErrBlockForbiddenField,
		},
		{
			name:  "photo by url with a caption",
			block: tgrich.Block{Type: tgrich.BlockPhoto, Photo: photo, Caption: &tgrich.Caption{Text: text}},
		},
		{
			name:  "photo by file id",
			block: tgrich.Block{Type: tgrich.BlockPhoto, Photo: &tgrich.Media{FileID: "AgACAgIAA"}},
		},
		{
			name:    "photo with no media at all",
			block:   tgrich.Block{Type: tgrich.BlockPhoto},
			wantErr: tgrich.ErrBlockMissingField,
		},
		{
			name:    "photo with an empty media descriptor",
			block:   tgrich.Block{Type: tgrich.BlockPhoto, Photo: &tgrich.Media{}},
			wantErr: tgrich.ErrMediaSource,
		},
		{
			name:    "photo carrying both a url and a file id",
			block:   tgrich.Block{Type: tgrich.BlockPhoto, Photo: &tgrich.Media{URL: "https://e.com/a.png", FileID: "AgACAgIAA"}},
			wantErr: tgrich.ErrMediaSource,
		},
		{
			name:    "photo carrying a video descriptor",
			block:   tgrich.Block{Type: tgrich.BlockPhoto, Photo: photo, Video: &tgrich.Media{URL: "https://e.com/a.mp4"}},
			wantErr: tgrich.ErrBlockForbiddenField,
		},
		{
			name:  "video",
			block: tgrich.Block{Type: tgrich.BlockVideo, Video: &tgrich.Media{URL: "https://e.com/a.mp4"}},
		},
		{
			name:    "video with no media at all",
			block:   tgrich.Block{Type: tgrich.BlockVideo},
			wantErr: tgrich.ErrBlockMissingField,
		},
		{
			name:    "unknown type",
			block:   tgrich.Block{Type: "nonsense"},
			wantErr: tgrich.ErrUnknownBlockType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.block.Validate()
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// A malformed block anywhere in the tree must fail the whole message, not just
// at the top level.
func TestValidateReachesNestedBlocks(t *testing.T) {
	bad := tgrich.Block{Type: tgrich.BlockPhoto}

	tests := []struct {
		name  string
		block tgrich.Block
	}{
		{
			name:  "inside a quote",
			block: tgrich.Block{Type: tgrich.BlockQuote, Blocks: []tgrich.Block{bad}},
		},
		{
			name:  "inside a collapsible",
			block: tgrich.Block{Type: tgrich.BlockCollapsible, Blocks: []tgrich.Block{bad}},
		},
		{
			name:  "inside a list item",
			block: tgrich.Block{Type: tgrich.BlockList, Items: []tgrich.ListItem{{Blocks: []tgrich.Block{bad}}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tgrich.InputRichMessage{Blocks: []tgrich.Block{tt.block}, SkipEntityDetection: true}
			require.ErrorIs(t, msg.Validate(tgrich.DefaultLimits()), tgrich.ErrBlockMissingField)
		})
	}
}

// Rich text is a tree, so a bad link hides arbitrarily deep inside a block's
// text, a title, a caption or a table cell.
func TestValidateRichTextLinks(t *testing.T) {
	tests := []struct {
		name    string
		text    tgrich.RichText
		wantErr error
	}{
		{
			name: "plain text",
			text: tgrich.RichText{Text: "x"},
		},
		{
			name: "https link",
			text: tgrich.RichText{Text: "x", URL: "https://example.com/a"},
		},
		{
			name: "http link",
			text: tgrich.RichText{Text: "x", URL: "http://example.com/a"},
		},
		{
			name: "in-message anchor",
			text: tgrich.RichText{Text: "x", URL: "#intro"},
		},
		{
			name: "tg deep link",
			text: tgrich.RichText{Text: "x", URL: "tg://user?id=1"},
		},
		{
			name:    "relative link",
			text:    tgrich.RichText{Text: "x", URL: "/notes/a"},
			wantErr: tgrich.ErrInvalidLinkURL,
		},
		{
			name:    "scheme-relative link",
			text:    tgrich.RichText{Text: "x", URL: "//example.com/a"},
			wantErr: tgrich.ErrInvalidLinkURL,
		},
		{
			name: "bad link nested in a child run",
			text: tgrich.RichText{Children: []tgrich.RichText{
				{Text: "ok"},
				{Text: "bad", URL: "notaurl"},
			}},
			wantErr: tgrich.ErrInvalidLinkURL,
		},
	}

	for _, tt := range tests {
		for _, where := range []struct {
			name  string
			build func(tgrich.RichText) tgrich.Block
		}{
			{"paragraph text", func(t tgrich.RichText) tgrich.Block { return tgrich.Paragraph(t) }},
			{"collapsible title", func(t tgrich.RichText) tgrich.Block {
				return tgrich.Block{Type: tgrich.BlockCollapsible, Title: &t, Blocks: []tgrich.Block{para("x")}}
			}},
			{"photo caption", func(t tgrich.RichText) tgrich.Block {
				return tgrich.Block{
					Type:    tgrich.BlockPhoto,
					Photo:   &tgrich.Media{URL: "https://e.com/a.png"},
					Caption: &tgrich.Caption{Text: t},
				}
			}},
			{"table cell", func(t tgrich.RichText) tgrich.Block {
				return tgrich.Block{
					Type:    tgrich.BlockTable,
					Columns: []tgrich.TableColumn{{}},
					Rows:    []tgrich.TableRow{{Cells: []tgrich.TableCell{{Text: t}}}},
				}
			}},
		} {
			t.Run(tt.name+" in "+where.name, func(t *testing.T) {
				err := where.build(tt.text).Validate()
				if tt.wantErr == nil {
					require.NoError(t, err)
					return
				}
				require.ErrorIs(t, err, tt.wantErr)
			})
		}
	}
}

// Media URLs must be https: only https was documented and measured.
func TestMediaURLMustBeHTTPS(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		{name: "https", url: "https://e.com/a.png"},
		{name: "http", url: "http://e.com/a.png", wantErr: tgrich.ErrMediaSource},
		{name: "vault relative path", url: "assets/a.png", wantErr: tgrich.ErrMediaSource},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := tgrich.Block{Type: tgrich.BlockPhoto, Photo: &tgrich.Media{URL: tt.url}}
			err := block.Validate()
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}
