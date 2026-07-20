package tgtd

import (
	"errors"
	"fmt"
	"trip2g/internal/tgrich"

	"github.com/gotd/td/tg"
)

// The Bot API and MTProto are two encodings of one block model: the Bot API
// takes flat JSON blocks, MTProto takes Instant View's PageBlock/RichText tree.
// tgrich.Block is the shared intermediate form, produced by the single AST walk
// in markdownv2.RichConverter — the walk that decides callout fold state, list
// checkbox state, table alignment and what counts as a conversion loss.
//
// Mapping the IR is therefore preferred to a second AST walk: those decisions
// stay in one place and cannot drift between the two senders. The mapper lives
// here rather than in tgrich so that tgrich keeps no dependency on gotd, and so
// MTProto encoding sits at the MTProto boundary next to convert.go.

// ErrRichMediaUnsupported reports a media block on the account path.
//
// MTProto references pre-uploaded media by id out of InputRichMessage.Photos
// and .Documents; it does not ingest the HTTPS URLs the Bot API accepts. Until
// that upload step exists, a media block fails loudly instead of sending a post
// with a silent hole in it.
var ErrRichMediaUnsupported = errors.New("rich media is not supported on the account path")

// ToPageBlocks maps the rich block IR onto MTProto's page-block tree.
func ToPageBlocks(blocks []tgrich.Block) ([]tg.PageBlockClass, error) {
	if len(blocks) == 0 {
		return nil, nil
	}

	out := make([]tg.PageBlockClass, 0, len(blocks))

	for _, block := range blocks {
		mapped, err := toPageBlock(block)
		if err != nil {
			return nil, err
		}

		out = append(out, mapped)
	}

	return out, nil
}

func toPageBlock(block tgrich.Block) (tg.PageBlockClass, error) {
	switch block.Type {
	case tgrich.BlockHeading:
		return heading(block), nil

	case tgrich.BlockParagraph:
		return &tg.PageBlockParagraph{Text: blockText(block.Text)}, nil

	case tgrich.BlockPre:
		return &tg.PageBlockPreformatted{Text: blockText(block.Text), Language: block.Language}, nil

	case tgrich.BlockDivider:
		return &tg.PageBlockDivider{}, nil

	case tgrich.BlockQuote:
		nested, err := ToPageBlocks(block.Blocks)
		if err != nil {
			return nil, err
		}

		return &tg.PageBlockBlockquoteBlocks{Blocks: nested, Caption: &tg.TextEmpty{}}, nil

	case tgrich.BlockDetails:
		return details(block)

	case tgrich.BlockList:
		return list(block)

	case tgrich.BlockTable:
		return table(block), nil

	case tgrich.BlockPhoto, tgrich.BlockVideo:
		return nil, fmt.Errorf("%w: block type %q", ErrRichMediaUnsupported, block.Type)
	}

	return nil, fmt.Errorf("unsupported rich block type %q", block.Type)
}

// heading clamps the level into 1..6. The converter only emits that range, but
// a level outside it must still publish: the level is the only thing lost.
func heading(block tgrich.Block) tg.PageBlockClass {
	text := blockText(block.Text)

	switch {
	case block.Size <= 1:
		return &tg.PageBlockHeading1{Text: text}
	case block.Size == 2:
		return &tg.PageBlockHeading2{Text: text}
	case block.Size == 3:
		return &tg.PageBlockHeading3{Text: text}
	case block.Size == 4:
		return &tg.PageBlockHeading4{Text: text}
	case block.Size == 5:
		return &tg.PageBlockHeading5{Text: text}
	}

	return &tg.PageBlockHeading6{Text: text}
}

func details(block tgrich.Block) (tg.PageBlockClass, error) {
	nested, err := ToPageBlocks(block.Blocks)
	if err != nil {
		return nil, err
	}

	out := &tg.PageBlockDetails{Blocks: nested, Title: blockText(block.Summary)}
	// Open is a flag field: setting it through the struct alone would not mark
	// the bit, and the fold state would silently invert.
	out.SetOpen(block.IsOpen)

	return out, nil
}

func list(block tgrich.Block) (tg.PageBlockClass, error) {
	items := make([]tg.PageListItemClass, 0, len(block.Items))

	for _, item := range block.Items {
		nested, err := ToPageBlocks(item.Blocks)
		if err != nil {
			return nil, err
		}

		out := &tg.PageListItemBlocks{Blocks: nested}

		// A nil Checked is a plain bullet; a set one is a task-list checkbox,
		// including when it is false — an unchecked task is still a checkbox.
		if item.Checked != nil {
			out.SetCheckbox(true)
			out.SetChecked(*item.Checked)
		}

		items = append(items, out)
	}

	return &tg.PageBlockList{Items: items}, nil
}

func table(block tgrich.Block) tg.PageBlockClass {
	rows := make([]tg.PageTableRow, 0, len(block.Cells))

	for _, row := range block.Cells {
		cells := make([]tg.PageTableCell, 0, len(row))

		for _, cell := range row {
			out := tg.PageTableCell{Text: ToRichText(cell.Text)}

			if cell.IsHeader {
				out.SetHeader(true)
			}

			// Left is the absence of both flags, so it needs no case.
			switch cell.Align {
			case tgrich.AlignCenter:
				out.SetAlignCenter(true)
			case tgrich.AlignRight:
				out.SetAlignRight(true)
			}

			cells = append(cells, out)
		}

		rows = append(rows, tg.PageTableRow{Cells: cells})
	}

	return &tg.PageBlockTable{Rows: rows}
}

// blockText maps a block's optional text field, which is absent on the block
// types that carry no text of their own.
func blockText(text *tgrich.RichText) tg.RichTextClass {
	if text == nil {
		return &tg.TextEmpty{}
	}

	return ToRichText(*text)
}

// richTextMarks lists the mark wrappers innermost first, matching the order
// tgrich.RichText marshals to JSON. The order is fixed so that the same input
// always produces the same tree on both transports.
var richTextMarks = []struct {
	on   func(tgrich.RichText) bool
	wrap func(tg.RichTextClass) tg.RichTextClass
}{
	{func(t tgrich.RichText) bool { return t.Code }, func(c tg.RichTextClass) tg.RichTextClass { return &tg.TextFixed{Text: c} }},
	{func(t tgrich.RichText) bool { return t.Marked }, func(c tg.RichTextClass) tg.RichTextClass { return &tg.TextMarked{Text: c} }},
	{func(t tgrich.RichText) bool { return t.Strike }, func(c tg.RichTextClass) tg.RichTextClass { return &tg.TextStrike{Text: c} }},
	{func(t tgrich.RichText) bool { return t.Underline }, func(c tg.RichTextClass) tg.RichTextClass { return &tg.TextUnderline{Text: c} }},
	{func(t tgrich.RichText) bool { return t.Italic }, func(c tg.RichTextClass) tg.RichTextClass { return &tg.TextItalic{Text: c} }},
	{func(t tgrich.RichText) bool { return t.Bold }, func(c tg.RichTextClass) tg.RichTextClass { return &tg.TextBold{Text: c} }},
}

// ToRichText maps one rich-text node onto MTProto's RichText tree: content
// first, then the link, then the marks wrapping outwards.
func ToRichText(text tgrich.RichText) tg.RichTextClass {
	if text.Anchor != "" {
		return &tg.TextAnchor{Name: text.Anchor, Text: &tg.TextEmpty{}}
	}

	node := content(text)

	if text.URL != "" {
		node = &tg.TextURL{Text: node, URL: text.URL}
	}

	for _, mark := range richTextMarks {
		if mark.on(text) {
			node = mark.wrap(node)
		}
	}

	return node
}

// content builds the node's own text and children, concatenating when both are
// present so a single field can carry a run of differently formatted spans.
func content(text tgrich.RichText) tg.RichTextClass {
	if len(text.Children) == 0 {
		if text.Text == "" {
			return &tg.TextEmpty{}
		}

		return &tg.TextPlain{Text: text.Text}
	}

	parts := make([]tg.RichTextClass, 0, len(text.Children)+1)

	if text.Text != "" {
		parts = append(parts, &tg.TextPlain{Text: text.Text})
	}

	for _, child := range text.Children {
		parts = append(parts, ToRichText(child))
	}

	if len(parts) == 1 {
		return parts[0]
	}

	return &tg.TextConcat{Texts: parts}
}
