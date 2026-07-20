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
			name:    "markdown only",
			msg:     tgrich.InputRichMessage{Markdown: "hi"},
			wantErr: nil,
		},
		{
			name:    "html only",
			msg:     tgrich.InputRichMessage{HTML: "<b>hi</b>"},
			wantErr: nil,
		},
		{
			name: "blocks plus markdown",
			msg: tgrich.InputRichMessage{
				Blocks:   []tgrich.Block{para("x")},
				Markdown: "hi",
			},
			wantErr: tgrich.ErrMultipleContentSources,
		},
		{
			name:    "markdown plus html",
			msg:     tgrich.InputRichMessage{Markdown: "hi", HTML: "<b>hi</b>"},
			wantErr: tgrich.ErrMultipleContentSources,
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
