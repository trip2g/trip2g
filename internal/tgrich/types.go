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

const (
	BlockHeading     BlockType = "heading"
	BlockParagraph   BlockType = "paragraph"
	BlockList        BlockType = "list"
	BlockQuote       BlockType = "quote"
	BlockCollapsible BlockType = "collapsible"
	BlockCode        BlockType = "code"
	BlockTable       BlockType = "table"
	BlockDivider     BlockType = "divider" // unprobed
	BlockPhoto       BlockType = "photo"
	BlockVideo       BlockType = "video"
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
	// URL makes this node a link. Anchors are ordinary URLs with a fragment.
	URL      string
	Children []RichText
}

// richTextJSON is the object form of RichText. Keep it beside RichText: the
// bare-string shortcut in MarshalJSON is the only other place the wire form of
// formatted text is decided.
type richTextJSON struct {
	Text      string     `json:"text,omitempty"`
	Bold      bool       `json:"bold,omitempty"`
	Italic    bool       `json:"italic,omitempty"`
	Underline bool       `json:"underline,omitempty"`
	Strike    bool       `json:"strike,omitempty"`
	Marked    bool       `json:"marked,omitempty"`
	Code      bool       `json:"code,omitempty"`
	URL       string     `json:"url,omitempty"`
	Children  []RichText `json:"children,omitempty"`
}

// IsPlain reports whether the node carries text and nothing else.
func (t RichText) IsPlain() bool {
	return !t.Bold && !t.Italic && !t.Underline && !t.Strike && !t.Marked &&
		!t.Code && t.URL == "" && len(t.Children) == 0
}

// IsEmpty reports whether the node would render nothing at all.
func (t RichText) IsEmpty() bool {
	return t.Text == "" && len(t.Children) == 0
}

func (t RichText) MarshalJSON() ([]byte, error) {
	if t.IsPlain() {
		return json.Marshal(t.Text) //nolint:wrapcheck // pure encoding
	}

	return json.Marshal(richTextJSON{
		Text: t.Text, Bold: t.Bold, Italic: t.Italic, Underline: t.Underline,
		Strike: t.Strike, Marked: t.Marked, Code: t.Code, URL: t.URL, Children: t.Children,
	}) //nolint:wrapcheck // pure encoding
}

func (t *RichText) UnmarshalJSON(data []byte) error {
	if bytes.HasPrefix(bytes.TrimLeft(data, " \t\r\n"), []byte(`"`)) {
		*t = RichText{}
		return json.Unmarshal(data, &t.Text) //nolint:wrapcheck // pure decoding
	}

	var obj richTextJSON
	if err := json.Unmarshal(data, &obj); err != nil {
		return err //nolint:wrapcheck // pure decoding
	}

	*t = RichText{
		Text: obj.Text, Bold: obj.Bold, Italic: obj.Italic, Underline: obj.Underline,
		Strike: obj.Strike, Marked: obj.Marked, Code: obj.Code, URL: obj.URL, Children: obj.Children,
	}

	return nil
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

// TableColumn describes one column. An empty Align leaves it to the server.
type TableColumn struct {
	Align Align `json:"align,omitempty"`
}

type TableCell struct {
	Text RichText `json:"text,omitempty"`
}

type TableRow struct {
	Header bool        `json:"header,omitempty"`
	Cells  []TableCell `json:"cells"`
}

// ListItem is one entry of a list block. Checked is nil for a plain item and
// set for a task-list checkbox. List items do not count towards the block
// limit (measured: a 501-item list is one block).
type ListItem struct {
	Blocks  []Block `json:"blocks,omitempty"`
	Checked *bool   `json:"checked,omitempty"`
}

// Block is one rich-message block in its flat wire form. Only the fields
// belonging to Type are populated; everything else is omitted.
type Block struct {
	Type BlockType `json:"type"`

	// heading, paragraph
	Text *RichText `json:"text,omitempty"`
	// heading: level 1..6
	Size int `json:"size,omitempty"`
	// heading: anchor target for in-message links
	Anchor string `json:"anchor,omitempty"`

	// list
	Ordered bool       `json:"ordered,omitempty"`
	Start   int        `json:"start,omitempty"`
	Items   []ListItem `json:"items,omitempty"`

	// quote, collapsible
	Title     *RichText `json:"title,omitempty"`
	Collapsed bool      `json:"collapsed,omitempty"`
	Blocks    []Block   `json:"blocks,omitempty"`

	// code
	Code     string `json:"code,omitempty"`
	Language string `json:"language,omitempty"`

	// table
	Columns []TableColumn `json:"columns,omitempty"`
	Rows    []TableRow    `json:"rows,omitempty"`

	// photo, video
	Photo   *Media   `json:"photo,omitempty"`
	Video   *Media   `json:"video,omitempty"`
	Caption *Caption `json:"caption,omitempty"`
}

// Heading builds a heading block. Anchor may be empty.
func Heading(size int, text RichText, anchor string) Block {
	return Block{Type: BlockHeading, Text: &text, Size: size, Anchor: anchor}
}

// Paragraph builds a paragraph block.
func Paragraph(text RichText) Block {
	return Block{Type: BlockParagraph, Text: &text}
}

// Code builds a fenced code block. Language may be empty.
func Code(code, language string) Block {
	return Block{Type: BlockCode, Code: code, Language: language}
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
