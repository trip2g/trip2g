package tgrich

import (
	"errors"
	"fmt"
	"unicode/utf16"
)

var (
	ErrNoContentSource        = errors.New("rich message has no content source")
	ErrMultipleContentSources = errors.New("rich message must carry exactly one of blocks, markdown, html")
	ErrUnknownBlockType       = errors.New("unknown block type")
	ErrTextTooLong            = errors.New("rich message text is too long")
	ErrTooManyBlocks          = errors.New("rich message has too many blocks")
	ErrTooManyMedia           = errors.New("rich message has too many media")
	ErrTooDeep                = errors.New("rich message nesting is too deep")
	ErrTooManyTableCols       = errors.New("rich message table has too many columns")
)

var knownBlockTypes = map[BlockType]struct{}{
	BlockHeading: {}, BlockParagraph: {}, BlockList: {}, BlockQuote: {},
	BlockCollapsible: {}, BlockCode: {}, BlockTable: {}, BlockDivider: {},
	BlockPhoto: {}, BlockVideo: {},
}

// Validate checks the message against the limits before it reaches the server.
// This matters more than usual here: several server-side violations return
// ok:true and silently discard content (measured), so local rejection is the
// only reliable signal.
func (m InputRichMessage) Validate(limits Limits) error {
	sources := 0
	for _, present := range []bool{len(m.Blocks) > 0, m.Markdown != "", m.HTML != ""} {
		if present {
			sources++
		}
	}

	switch {
	case sources == 0:
		return ErrNoContentSource
	case sources > 1:
		return ErrMultipleContentSources
	}

	if len(m.Blocks) > limits.MaxBlocks {
		return fmt.Errorf("%w: %d > %d", ErrTooManyBlocks, len(m.Blocks), limits.MaxBlocks)
	}

	stats := blockStats{limits: limits}
	if err := stats.walk(m.Blocks, 1); err != nil {
		return err
	}

	if stats.textUnits > limits.TextLength {
		return fmt.Errorf("%w: %d > %d", ErrTextTooLong, stats.textUnits, limits.TextLength)
	}

	if stats.media > limits.MaxMedia {
		return fmt.Errorf("%w: %d > %d", ErrTooManyMedia, stats.media, limits.MaxMedia)
	}

	return nil
}

type blockStats struct {
	limits    Limits
	textUnits int
	media     int
}

//nolint:cyclop // flat per-variant accounting
func (s *blockStats) walk(blocks []Block, depth int) error {
	if len(blocks) > 0 && depth > s.limits.MaxDepth {
		return fmt.Errorf("%w: %d > %d", ErrTooDeep, depth, s.limits.MaxDepth)
	}

	for _, block := range blocks {
		if _, ok := knownBlockTypes[block.Type]; !ok {
			return fmt.Errorf("%w: %q", ErrUnknownBlockType, block.Type)
		}

		s.count(block.Text)
		s.count(block.Title)
		s.textUnits += utf16Len(block.Code)

		if block.Caption != nil {
			s.textUnits += utf16Len(block.Caption.Text.PlainText())
		}

		if block.Type == BlockPhoto || block.Type == BlockVideo {
			s.media++
		}

		if len(block.Columns) > s.limits.MaxTableCols {
			return fmt.Errorf("%w: %d > %d", ErrTooManyTableCols, len(block.Columns), s.limits.MaxTableCols)
		}

		for _, row := range block.Rows {
			if len(row.Cells) > s.limits.MaxTableCols {
				return fmt.Errorf("%w: %d > %d", ErrTooManyTableCols, len(row.Cells), s.limits.MaxTableCols)
			}
			for _, cell := range row.Cells {
				s.textUnits += utf16Len(cell.Text.PlainText())
			}
		}

		if err := s.walk(block.Blocks, depth+1); err != nil {
			return err
		}

		for _, item := range block.Items {
			if err := s.walk(item.Blocks, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *blockStats) count(text *RichText) {
	if text != nil {
		s.textUnits += utf16Len(text.PlainText())
	}
}

// utf16Len returns the length of s in UTF-16 code units, which is how Telegram
// counts message text.
func utf16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}
