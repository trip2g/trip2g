package tgrich

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf16"
)

var (
	ErrNoContentSource          = errors.New("rich message has no content source")
	ErrUnsupportedContentSource = errors.New("rich message must carry blocks, not markdown or html")
	ErrUnknownBlockType         = errors.New("unknown block type")
	ErrBlockMissingField        = errors.New("block is missing a field its type requires")
	ErrBlockForbiddenField      = errors.New("block carries a field that does not belong to its type")
	ErrHeadingSize              = errors.New("heading size must be between 1 and 6")
	ErrMediaSource              = errors.New("media must carry exactly one of an https url or a file id")
	ErrInvalidLinkURL           = errors.New("link url must be absolute or an in-message anchor")
	ErrTextTooLong              = errors.New("rich message text is too long")
	ErrTooManyBlocks            = errors.New("rich message has too many blocks")
	ErrTooManyMedia             = errors.New("rich message has too many media")
	ErrTooDeep                  = errors.New("rich message nesting is too deep")
	ErrTooManyTableCols         = errors.New("rich message table has too many columns")
)

const maxHeadingSize = 6

// Validate checks the message against the limits before it reaches the server.
// This matters more than usual here: several server-side violations return
// ok:true and silently discard content (measured), so local rejection is the
// only reliable signal.
//
// Blocks are the only content source this path emits. The server accepts
// markdown and html too, and silently discards the losers when more than one is
// present, so carrying them at all is a way to lose a whole post to a typo.
func (m InputRichMessage) Validate(limits Limits) error {
	if m.Markdown != "" || m.HTML != "" {
		return ErrUnsupportedContentSource
	}

	if len(m.Blocks) == 0 {
		return ErrNoContentSource
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

// blockField is one field of Block, as a bit so a variant's whole field set is
// a single mask.
type blockField uint32

const (
	fieldText blockField = 1 << iota
	fieldSize
	fieldAnchor
	fieldOrdered
	fieldStart
	fieldItems
	fieldTitle
	fieldCollapsed
	fieldBlocks
	fieldCode
	fieldLanguage
	fieldColumns
	fieldRows
	fieldPhoto
	fieldVideo
	fieldCaption
)

// fieldNames is ordered by bit so an error can name the offending field.
var fieldNames = []struct {
	field blockField
	name  string
}{
	{fieldText, "text"},
	{fieldSize, "size"},
	{fieldAnchor, "anchor"},
	{fieldOrdered, "ordered"},
	{fieldStart, "start"},
	{fieldItems, "items"},
	{fieldTitle, "title"},
	{fieldCollapsed, "collapsed"},
	{fieldBlocks, "blocks"},
	{fieldCode, "code"},
	{fieldLanguage, "language"},
	{fieldColumns, "columns"},
	{fieldRows, "rows"},
	{fieldPhoto, "photo"},
	{fieldVideo, "video"},
	{fieldCaption, "caption"},
}

// blockSpec is what one variant may and must carry. A variant absent from
// blockSpecs is an unknown block type.
type blockSpec struct {
	allowed  blockField
	required blockField
}

var blockSpecs = map[BlockType]blockSpec{
	BlockHeading: {
		allowed:  fieldText | fieldSize | fieldAnchor,
		required: fieldText | fieldSize,
	},
	BlockParagraph: {
		allowed:  fieldText,
		required: fieldText,
	},
	BlockList: {
		allowed:  fieldOrdered | fieldStart | fieldItems,
		required: fieldItems,
	},
	BlockQuote: {
		allowed:  fieldTitle | fieldBlocks,
		required: fieldBlocks,
	},
	BlockCollapsible: {
		allowed:  fieldTitle | fieldCollapsed | fieldBlocks,
		required: fieldBlocks,
	},
	BlockCode: {
		allowed:  fieldCode | fieldLanguage,
		required: fieldCode,
	},
	BlockTable: {
		allowed:  fieldColumns | fieldRows,
		required: fieldRows,
	},
	BlockDivider: {},
	BlockPhoto: {
		allowed:  fieldPhoto | fieldCaption,
		required: fieldPhoto,
	},
	BlockVideo: {
		allowed:  fieldVideo | fieldCaption,
		required: fieldVideo,
	},
}

// present returns the set of fields the block actually carries.
//
//nolint:cyclop // one flat branch per field
func (b Block) present() blockField {
	var set blockField

	for _, check := range []struct {
		on    bool
		field blockField
	}{
		{b.Text != nil, fieldText},
		{b.Size != 0, fieldSize},
		{b.Anchor != "", fieldAnchor},
		{b.Ordered, fieldOrdered},
		{b.Start != 0, fieldStart},
		{len(b.Items) > 0, fieldItems},
		{b.Title != nil, fieldTitle},
		{b.Collapsed, fieldCollapsed},
		{len(b.Blocks) > 0, fieldBlocks},
		{b.Code != "", fieldCode},
		{b.Language != "", fieldLanguage},
		{len(b.Columns) > 0, fieldColumns},
		{len(b.Rows) > 0, fieldRows},
		{b.Photo != nil, fieldPhoto},
		{b.Video != nil, fieldVideo},
		{b.Caption != nil, fieldCaption},
	} {
		if check.on {
			set |= check.field
		}
	}

	return set
}

// Validate checks one block against its own variant: the fields that variant
// requires, the fields it forbids, and the rich text it carries. Checking only
// the discriminator is not enough — a paragraph carrying code and media fields
// at once, or a photo carrying no media at all, are both accepted by the server
// and render as something the author did not write.
//
// It does not recurse into nested blocks; the message walk does that, so
// nesting depth is counted in one place.
func (b Block) Validate() error {
	spec, known := blockSpecs[b.Type]
	if !known {
		return fmt.Errorf("%w: %q", ErrUnknownBlockType, b.Type)
	}

	present := b.present()

	if missing := spec.required &^ present; missing != 0 {
		return fmt.Errorf("%w: %s needs %s", ErrBlockMissingField, b.Type, names(missing))
	}

	if forbidden := present &^ spec.allowed; forbidden != 0 {
		return fmt.Errorf("%w: %s cannot carry %s", ErrBlockForbiddenField, b.Type, names(forbidden))
	}

	if b.Type == BlockHeading && b.Size > maxHeadingSize {
		return fmt.Errorf("%w: %d", ErrHeadingSize, b.Size)
	}

	if err := b.validateMedia(); err != nil {
		return err
	}

	return b.validateText()
}

func (b Block) validateMedia() error {
	for _, media := range []*Media{b.Photo, b.Video} {
		if media == nil {
			continue
		}
		if err := media.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (b Block) validateText() error {
	for _, text := range []*RichText{b.Text, b.Title} {
		if text == nil {
			continue
		}
		if err := text.validate(); err != nil {
			return err
		}
	}

	if b.Caption != nil {
		if err := b.Caption.Text.validate(); err != nil {
			return err
		}
	}

	for _, row := range b.Rows {
		for _, cell := range row.Cells {
			if err := cell.Text.validate(); err != nil {
				return err
			}
		}
	}

	return nil
}

// validate enforces exactly one media source. The Bot API ingests plain HTTPS
// URLs server-side; http was never documented nor measured, and a FileID
// replays an asset Telegram already holds.
func (m Media) validate() error {
	switch {
	case m.URL != "" && m.FileID != "":
		return fmt.Errorf("%w: both are set", ErrMediaSource)
	case m.URL == "" && m.FileID == "":
		return fmt.Errorf("%w: neither is set", ErrMediaSource)
	case m.URL != "" && !strings.HasPrefix(m.URL, "https://"):
		return fmt.Errorf("%w: %q is not https", ErrMediaSource, m.URL)
	}

	return nil
}

// validate walks the rich-text tree and checks every link. A relative or
// malformed url is not rejected by the server: it degrades to plain text or to
// something the author did not mean, which is exactly the class of failure this
// package exists to catch locally.
func (t RichText) validate() error {
	if t.URL != "" && !validLinkURL(t.URL) {
		return fmt.Errorf("%w: %q", ErrInvalidLinkURL, t.URL)
	}

	for _, child := range t.Children {
		if err := child.validate(); err != nil {
			return err
		}
	}

	return nil
}

// validLinkURL accepts an absolute url or an in-message anchor. Anchors are how
// a table of contents jumps to a heading; everything else needs a scheme,
// because Telegram has no base url to resolve against.
func validLinkURL(raw string) bool {
	if strings.HasPrefix(raw, "#") {
		return true
	}

	parsed, err := url.Parse(raw)

	return err == nil && parsed.Scheme != ""
}

func names(set blockField) string {
	var out []string
	for _, entry := range fieldNames {
		if set&entry.field != 0 {
			out = append(out, entry.name)
		}
	}

	return strings.Join(out, ", ")
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
		if err := block.Validate(); err != nil {
			return err
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
