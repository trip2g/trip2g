package tgrich_test

import (
	"encoding/json"
	"strings"
	"testing"
	"trip2g/internal/tgrich"

	"github.com/stretchr/testify/require"
)

func TestRequestParams(t *testing.T) {
	req := tgrich.Request{
		ChatID: -1004487679938,
		RichMessage: tgrich.InputRichMessage{
			Blocks:              []tgrich.Block{tgrich.Heading(1, tgrich.RichText{Text: "Title"})},
			SkipEntityDetection: true,
		},
	}

	params, err := req.Params()
	require.NoError(t, err)

	require.Equal(t, "-1004487679938", params["chat_id"])
	require.NotContains(t, params, "disable_notification")

	// The nested object rides as a JSON string: MakeRequest form-encodes a
	// map[string]string, so there is nowhere else for it to go.
	var msg struct {
		Blocks              []tgrich.Block `json:"blocks"`
		SkipEntityDetection bool           `json:"skip_entity_detection"`
		Markdown            *string        `json:"markdown"`
		HTML                *string        `json:"html"`
	}
	require.NoError(t, json.Unmarshal([]byte(params["rich_message"]), &msg))

	require.True(t, msg.SkipEntityDetection)
	require.Nil(t, msg.Markdown, "markdown must not appear on the wire at all")
	require.Nil(t, msg.HTML, "html must not appear on the wire at all")
	require.Len(t, msg.Blocks, 1)
	require.Equal(t, tgrich.BlockHeading, msg.Blocks[0].Type)
}

// skip_entity_detection is never omitted: leaving it out means the server turns
// $USD into a cashtag and any bare 16-digit run into a bank card number.
func TestRequestParamsAlwaysCarriesSkipEntityDetection(t *testing.T) {
	req := tgrich.Request{ChatID: 1, RichMessage: tgrich.InputRichMessage{Blocks: []tgrich.Block{para("x")}}}

	params, err := req.Params()
	require.NoError(t, err)
	require.Contains(t, params["rich_message"], `"skip_entity_detection":false`)

	req.RichMessage.SkipEntityDetection = true
	params, err = req.Params()
	require.NoError(t, err)
	require.Contains(t, params["rich_message"], `"skip_entity_detection":true`)
}

func TestRequestParamsDisableNotification(t *testing.T) {
	req := tgrich.Request{
		ChatID:              1,
		RichMessage:         tgrich.InputRichMessage{Blocks: []tgrich.Block{para("x")}},
		DisableNotification: true,
	}

	params, err := req.Params()
	require.NoError(t, err)
	require.Equal(t, "true", params["disable_notification"])
}

func TestMethodName(t *testing.T) {
	require.Equal(t, "sendRichMessage", tgrich.Method)
}

func TestDecodeSendResult(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantID  int64
		wantErr bool
	}{
		{
			name:   "message with an echoed rich message",
			raw:    `{"message_id":42,"chat":{"id":-100},"rich_message":{"blocks":[{"type":"paragraph","text":"hi"}]}}`,
			wantID: 42,
		},
		{
			name:   "message without an echo",
			raw:    `{"message_id":7,"chat":{"id":-100}}`,
			wantID: 7,
		},
		{
			name:    "not a message at all",
			raw:     `"nope"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tgrich.DecodeSendResult(json.RawMessage(tt.raw))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantID, res.MessageID)
		})
	}
}

func TestDecodeSendResultKeepsTheEchoedBlocks(t *testing.T) {
	// The echo comes back in the wire union: a bare string, an array for a
	// concatenation, and a typed object for a mark.
	raw := `{"message_id":42,"rich_message":{"blocks":[
		{"type":"heading","text":"Title","size":1},
		{"type":"paragraph","text":[{"type":"bold","text":"a"},"b"]}
	]}}`

	res, err := tgrich.DecodeSendResult(json.RawMessage(raw))
	require.NoError(t, err)

	require.Len(t, res.Blocks, 2)
	require.Equal(t, tgrich.BlockHeading, res.Blocks[0].Type)
	require.Equal(t, "ab", res.Blocks[1].Text.PlainText())
}

func TestMeasure(t *testing.T) {
	tests := []struct {
		name   string
		blocks []tgrich.Block
		want   tgrich.Stats
	}{
		{
			name:   "empty",
			blocks: nil,
			want:   tgrich.Stats{},
		},
		{
			name:   "one plain paragraph is one block and one run",
			blocks: []tgrich.Block{para("hello")},
			want:   tgrich.Stats{Blocks: 1, TextUnits: 5},
		},
		{
			// Only the marked children count: the server merges adjacent plain
			// runs, so counting those would report loss on almost every message.
			name: "only formatted children count as runs",
			blocks: []tgrich.Block{tgrich.Paragraph(tgrich.RichText{Children: []tgrich.RichText{
				{Text: "a", Bold: true},
				{Text: "b"},
				{Text: "c", Italic: true},
			}})},
			want: tgrich.Stats{Blocks: 1, FormattedRuns: 2, TextUnits: 3},
		},
		{
			// The code rides in the block's text, so it is a run like any other.
			name:   "pre text counts, and the block counts once",
			blocks: []tgrich.Block{tgrich.Pre("x := 1", "go")},
			want:   tgrich.Stats{Blocks: 1, TextUnits: 6},
		},
		{
			name: "list items count as blocks of their own",
			blocks: []tgrich.Block{{Type: tgrich.BlockList, Items: []tgrich.ListItem{
				{Blocks: []tgrich.Block{para("a")}},
				{Blocks: []tgrich.Block{para("b")}},
			}}},
			want: tgrich.Stats{Blocks: 3, TextUnits: 2},
		},
		{
			name: "nested blocks and summaries are reached",
			blocks: []tgrich.Block{{
				Type:    tgrich.BlockDetails,
				Summary: &tgrich.RichText{Text: "more"},
				Blocks:  []tgrich.Block{para("inside")},
			}},
			want: tgrich.Stats{Blocks: 2, TextUnits: 10},
		},
		{
			name: "table cells count",
			blocks: []tgrich.Block{{
				Type: tgrich.BlockTable,
				Cells: [][]tgrich.TableCell{{
					{Text: tgrich.RichText{Text: "ab"}, IsHeader: true},
					{Text: tgrich.RichText{Text: "cd"}, IsHeader: true},
				}},
			}},
			want: tgrich.Stats{Blocks: 1, TextUnits: 4},
		},
		{
			name:   "an astral rune is two text units",
			blocks: []tgrich.Block{para("😀")},
			want:   tgrich.Stats{Blocks: 1, TextUnits: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tgrich.Measure(tt.blocks))
		})
	}
}

// The server discards content past a run-cost ceiling and still answers ok:true.
// Comparing what came back against what went out is the only way to notice.
func TestVerifyEcho(t *testing.T) {
	sent := []tgrich.Block{
		tgrich.Heading(1, tgrich.RichText{Text: "Title"}),
		para("hello"),
		para("world"),
	}

	tests := []struct {
		name    string
		echo    []tgrich.Block
		wantErr bool
	}{
		{
			name: "identical echo",
			echo: sent,
			// no error
		},
		{
			name: "no echo at all is not a truncation signal",
			echo: nil,
		},
		{
			name:    "a dropped block",
			echo:    sent[:2],
			wantErr: true,
		},
		{
			name: "truncated text in a kept block",
			echo: []tgrich.Block{
				tgrich.Heading(1, tgrich.RichText{Text: "Title"}),
				para("hello"),
				para("wor"),
			},
			wantErr: true,
		},
		{
			name: "dropped runs",
			echo: []tgrich.Block{
				tgrich.Heading(1, tgrich.RichText{Text: "Title"}),
				para("hello"),
				tgrich.Paragraph(tgrich.RichText{Children: []tgrich.RichText{{Text: "world"}}}),
			},
			wantErr: false, // same run count: one child run, one plain run
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tgrich.VerifyEcho(sent, tt.echo)
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tgrich.ErrContentDiscarded)
		})
	}
}

func TestVerifyEchoNamesWhatWasLost(t *testing.T) {
	sent := []tgrich.Block{para("hello"), para("world")}

	err := tgrich.VerifyEcho(sent, []tgrich.Block{para("hello")})

	require.ErrorIs(t, err, tgrich.ErrContentDiscarded)
	require.True(t, strings.Contains(err.Error(), "blocks"), "error should name the counts: %v", err)
}
