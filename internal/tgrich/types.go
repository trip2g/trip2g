// Package tgrich holds the pure wire types, limits and validation for Telegram
// rich messages (`sendRichMessage`, Bot API 10.1+). It performs no network IO:
// the app layer owns the transport, per docs/dev/app_patterns.md.
package tgrich

import (
	"bytes"
	"encoding/json"
)

// BlockType is the wire discriminator of a block.
//
// The wire form is FLAT: the discriminator sits next to the variant's own
// fields rather than wrapping them. Measured probe responses (July 2026) came
// back as `{"type":"heading","text":"h1","size":1}` and
// `{"type":"paragraph","text":...}`; media nests only its own descriptor under
// a key named after the type, `{"type":"video","video":{...},"caption":{...}}`.
// The reference type names (RichBlockSectionHeading and friends) never appear
// on the wire.
//
// Variants marked "unprobed" below are documented but were never exercised
// against a live server, so their exact field names are the weakest part of
// this file. All JSON shaping lives in this one file so a correction stays a
// single edit.
type BlockType string

// Measured against a live bot, July 2026. The names below marked "measured"
// were accepted by the server; the earlier guesses they replace ("code",
// "quote", "collapsible") were all rejected with
// `can't parse InputRichBlock: type "..." is unsupported`.
const (
	BlockHeading   BlockType = "heading"   // measured
	BlockParagraph BlockType = "paragraph" // measured
	BlockList      BlockType = "list"      // measured
	BlockQuote     BlockType = "blockquote"
	BlockDetails   BlockType = "details" // measured; the collapsible section
	BlockPre       BlockType = "pre"     // measured; the fenced code block
	BlockTable     BlockType = "table"   // measured
	BlockDivider   BlockType = "divider" // unprobed
	BlockPhoto     BlockType = "photo"   // unprobed
	BlockVideo     BlockType = "video"   // unprobed
)

// Align is a table column alignment. An empty value means "server default".
type Align string

const (
	AlignLeft   Align = "left"
	AlignRight  Align = "right"
	AlignCenter Align = "center"
)

// RichText is one node of Telegram's rich-text tree, the block-level `text`
// field. A node with no marks, no link and no children marshals as a bare JSON
// string, which is what the probe responses showed for headings; anything else
// marshals as an object. Children concatenate in order and let a single field
// carry a run of differently formatted spans.
type RichText struct {
	Text      string
	Bold      bool
	Italic    bool
	Underline bool
	Strike    bool
	Marked    bool
	Code      bool
	// URL makes this node a link.
	URL string
	// Anchor makes this node an in-message anchor target, and nothing else:
	// the server echoes an anchor node back with its name and no text, so a
	// node carrying one renders no content of its own.
	Anchor   string
	Children []RichText
}

// Rich text is a tagged union on the wire, exactly like a block, and this is
// the single most error-prone part of the format: a text object without a
// "type" is parsed as a *block* and rejected with the block's own error
// message, `can't parse InputRichBlock: Can't find field "type"`, which points
// at entirely the wrong place.
//
// Measured forms:
//
//	"plain"                                    a bare string
//	["a", {...}, "b"]                          concatenation, as a JSON array
//	{"type":"bold","text":<rich text>}         a mark, nesting through `text`
//	{"type":"url","text":<rich text>,"url":…}  a link
//	{"type":"anchor","name":"…"}               an anchor target, and nothing else
//
// Marks nest rather than combine: bold italic is a bold wrapping an italic.
const (
	textTypeURL    = "url"
	textTypeAnchor = "anchor"
)

// textMarks lists the mark node types in the order they wrap, innermost first.
// The order is fixed so the same RichText always produces the same bytes: a
// canonical payload hash is worthless if the encoding wanders.
var textMarks = []struct { //nolint:gochecknoglobals // read-only lookup table of mark node types
	name string
	on   func(RichText) bool
}{
	{"code", func(t RichText) bool { return t.Code }},
	{"marked", func(t RichText) bool { return t.Marked }},
	{"strikethrough", func(t RichText) bool { return t.Strike }},
	{"underline", func(t RichText) bool { return t.Underline }},
	{"italic", func(t RichText) bool { return t.Italic }},
	{"bold", func(t RichText) bool { return t.Bold }},
}

// IsPlain reports whether the node carries text and nothing else.
func (t RichText) IsPlain() bool {
	return !t.Bold && !t.Italic && !t.Underline && !t.Strike && !t.Marked &&
		!t.Code && t.URL == "" && t.Anchor == "" && len(t.Children) == 0
}

// IsFormatted reports whether the node itself carries a mark or a link. A node
// that only holds children is not formatted: it is a concatenation, and the
// server is free to flatten it.
func (t RichText) IsFormatted() bool {
	return t.Bold || t.Italic || t.Underline || t.Strike || t.Marked ||
		t.Code || t.URL != ""
}

// IsEmpty reports whether the node would render nothing at all.
func (t RichText) IsEmpty() bool {
	return t.Text == "" && len(t.Children) == 0
}

func (t RichText) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.wire()) //nolint:wrapcheck // pure encoding
}

// wire builds the union form: the content first, then the link, then the marks
// wrapping outwards.
func (t RichText) wire() any {
	if t.Anchor != "" {
		return map[string]any{"type": textTypeAnchor, "name": t.Anchor}
	}

	var node any = t.Text

	if len(t.Children) > 0 {
		parts := make([]any, 0, len(t.Children)+1)
		if t.Text != "" {
			parts = append(parts, t.Text)
		}
		for _, child := range t.Children {
			parts = append(parts, child.wire())
		}

		node = parts
		if len(parts) == 1 {
			node = parts[0]
		}
	}

	if t.URL != "" {
		node = map[string]any{"type": textTypeURL, "text": node, "url": t.URL}
	}

	for _, mark := range textMarks {
		if mark.on(t) {
			node = map[string]any{"type": mark.name, "text": node}
		}
	}

	return node
}

// richTextWire is the object form, used only for decoding the server's echo.
type richTextWire struct {
	Type string   `json:"type"`
	Text RichText `json:"text"`
	URL  string   `json:"url"`
	Name string   `json:"name"`
}

func (t *RichText) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimLeft(data, " \t\r\n")

	switch {
	case bytes.HasPrefix(trimmed, []byte(`"`)):
		*t = RichText{}
		return json.Unmarshal(data, &t.Text) //nolint:wrapcheck // pure decoding

	case bytes.HasPrefix(trimmed, []byte(`[`)):
		*t = RichText{}
		return json.Unmarshal(data, &t.Children) //nolint:wrapcheck // pure decoding
	}

	var obj richTextWire
	if err := json.Unmarshal(data, &obj); err != nil {
		return err //nolint:wrapcheck // pure decoding
	}

	*t = obj.Text
	t.applyWireType(obj)

	return nil
}

// applyWireType folds one wire node's own type back onto the decoded child.
// The child already carries any marks below it, so a nested pair decodes into
// a single node with both marks set — which is the shape the converter emits
// and therefore the shape the echo comparison must see.
func (t *RichText) applyWireType(obj richTextWire) {
	switch obj.Type {
	case textTypeURL:
		t.URL = obj.URL
	case textTypeAnchor:
		*t = RichText{Anchor: obj.Name}
	case "code":
		t.Code = true
	case "marked":
		t.Marked = true
	case "strikethrough":
		t.Strike = true
	case "underline":
		t.Underline = true
	case "italic":
		t.Italic = true
	case "bold":
		t.Bold = true
	}
}

// PlainText returns the visible text of the node and its children.
func (t RichText) PlainText() string {
	var buf bytes.Buffer
	t.writePlain(&buf)
	return buf.String()
}

func (t RichText) writePlain(buf *bytes.Buffer) {
	buf.WriteString(t.Text)
	for _, child := range t.Children {
		child.writePlain(buf)
	}
}

// Media describes one photo or video. The Bot API ingests plain HTTPS URLs
// server-side; FileID replays an already-uploaded asset and is what an edit
// must resend to avoid dropping the media.
type Media struct {
	URL    string `json:"url,omitempty"`
	FileID string `json:"file_id,omitempty"`
}

// Caption is a media block's caption. Measured as an object with its own
// `text`, unlike the block-level `text` field which is the rich text directly.
type Caption struct {
	Text RichText `json:"text"`
}

// TableCell is one cell. Measured: cells are a flat row-major grid on the block
// itself (`cells: [[cell, cell], ...]`), not rows carrying their own cell list.
// A row object with a `cells` field is rejected outright.
//
// IsHeader is the header marker — `header` is silently ignored, `is_header` is
// echoed back and switches the server's default alignment to centre. Align and
// VAlign are always echoed, defaulted to left/middle when omitted.
type TableCell struct {
	Text     RichText `json:"text,omitempty"`
	IsHeader bool     `json:"is_header,omitempty"`
	Align    Align    `json:"align,omitempty"`
}

// ListItem is one entry of a list block. Checked is nil for a plain item and
// set for a task-list checkbox. List items do not count towards the block
// limit (measured: a 501-item list is one block).
//
// There is no ordered-list support on this path. Measured: `ordered`, `start`,
// `style`, `is_ordered`, `list_type` and `numbered` are all silently ignored,
// a per-item `label` is overwritten, and every item comes back labelled "•".
// An ordered list therefore renders as bullets.
type ListItem struct {
	Blocks  []Block `json:"blocks,omitempty"`
	Checked *bool   `json:"checked,omitempty"`
}

// Block is one rich-message block in its flat wire form. Only the fields
// belonging to Type are populated; everything else is omitted.
type Block struct {
	Type BlockType `json:"type"`

	// heading, paragraph, pre
	Text *RichText `json:"text,omitempty"`
	// heading: level 1..6
	Size int `json:"size,omitempty"`

	// list
	Items []ListItem `json:"items,omitempty"`

	// details: the always-visible summary line
	Summary *RichText `json:"summary,omitempty"`
	// details: fold state. Measured: `is_open` is the only spelling the server
	// echoes; open, collapsed, expanded, folded, is_collapsed, default_open and
	// closed are all accepted and silently ignored. Absent means collapsed, so
	// this field is the expanded case and never the other way round.
	IsOpen bool `json:"is_open,omitempty"`
	// blockquote, details
	Blocks []Block `json:"blocks,omitempty"`

	// pre: the language tag. The code itself rides in Text.
	Language string `json:"language,omitempty"`

	// table: row-major, header row included
	Cells [][]TableCell `json:"cells,omitempty"`

	// photo, video
	Photo   *Media    `json:"photo,omitempty"`
	Video   *Media    `json:"video,omitempty"`
	Caption *RichText `json:"caption,omitempty"`
}

// Heading builds a heading block.
//
// There is no anchor. Measured: `anchor`, `name` and `id` are all accepted and
// none is echoed back, so in-message anchor targets are not reachable from the
// bot path — and with them the table-of-contents answer to the visible fold.
func Heading(size int, text RichText) Block {
	return Block{Type: BlockHeading, Text: &text, Size: size}
}

// Paragraph builds a paragraph block.
func Paragraph(text RichText) Block {
	return Block{Type: BlockParagraph, Text: &text}
}

// Pre builds a fenced code block. Language may be empty.
func Pre(code, language string) Block {
	text := RichText{Text: code}
	return Block{Type: BlockPre, Text: &text, Language: language}
}

// InputRichMessage carries exactly one content source. The server does not
// enforce that (measured: markdown + html + blocks all return ok:true, with
// precedence blocks > markdown > html and the losers silently discarded), so
// an empty Blocks would swallow a populated Markdown. Validate enforces it
// locally instead.
type InputRichMessage struct {
	Blocks   []Block `json:"blocks,omitempty"`
	Markdown string  `json:"markdown,omitempty"`
	HTML     string  `json:"html,omitempty"`
	// SkipEntityDetection must be true for our content: auto-detection is on by
	// default and turns `$USD` into a cashtag and any bare 16-digit run into a
	// bank card number. Always sent, never omitted.
	SkipEntityDetection bool `json:"skip_entity_detection"`
}

// Request is the `sendRichMessage` request body.
type Request struct {
	ChatID              int64            `json:"chat_id"`
	RichMessage         InputRichMessage `json:"rich_message"`
	DisableNotification bool             `json:"disable_notification,omitempty"`
}

// EditRequest is the `editMessageText` request body for a message that is rich.
// It replaces the whole block list, which is also why media inside a rich post
// cannot be edited piecemeal.
type EditRequest struct {
	ChatID      int64            `json:"chat_id"`
	MessageID   int64            `json:"message_id"`
	RichMessage InputRichMessage `json:"rich_message"`
}
