package markdownv2

import (
	"strings"
	"trip2g/internal/image"
	"trip2g/internal/mdloader/callout"
	"trip2g/internal/model"
	"trip2g/internal/tgrich"
	"unicode/utf16"

	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

// codeTabWidth expands tabs in fenced code: Telegram collapses a literal tab
// inside a code block to a single space (measured), and this is a Go codebase.
const codeTabWidth = 4

// LossKind names a thing the rich converter could not represent. It is a typed
// value on purpose: the classic converter only emits warning strings that mix
// conversion losses with length and policy warnings, which is why `auto` cannot
// be driven from them. A future `auto` predicate reads this set instead.
type LossKind string

const (
	// LossUnsupportedNode is an AST node with no rich-block equivalent.
	LossUnsupportedNode LossKind = "unsupported_node"
	// LossRawHTML is raw HTML beyond the supported <u> pair.
	LossRawHTML LossKind = "raw_html"
	// LossCustomEmoji is a Telegram custom emoji. Rich messages resolve the id
	// server-side and substitute the sticker set's own fallback, so the emoji
	// never arrives; a word in that slot is rejected outright.
	LossCustomEmoji LossKind = "custom_emoji"
	// LossEmbeddedWikiLink is an ![[embed]], which has no rich equivalent yet.
	LossEmbeddedWikiLink LossKind = "embedded_wikilink"
	// LossUnresolvedWikiLink is a [[link]] no resolver could turn into a URL.
	LossUnresolvedWikiLink LossKind = "unresolved_wikilink"
	// LossUnresolvedMedia is a local asset with no absolute URL. The Bot API
	// ingests plain HTTPS URLs, so resolving vault paths is app-layer work.
	LossUnresolvedMedia LossKind = "unresolved_media"
	// LossInlineMedia is media inside running text. Telegram accepts it, splits
	// the paragraph and drops the caption (measured), so it is refused here.
	LossInlineMedia LossKind = "inline_media"
)

// RichLoss is one unrepresentable node. Node is the AST node kind; Detail is
// the destination, target or identifier when there is one; Alt is the text the
// author wrote in the node's alt slot, which is what a reader would have seen.
// A loss set that reports only an opaque identifier cannot tell anyone what
// went missing.
type RichLoss struct {
	Kind   LossKind
	Node   string
	Detail string
	Alt    string
}

// RichConverterResult is the outcome of one note conversion.
type RichConverterResult struct {
	Blocks []tgrich.Block
	Losses []RichLoss
	// VisibleUTF16Length is the text budget the blocks consume, counted the way
	// Telegram counts it.
	VisibleUTF16Length int
	// Anchors holds the heading ids available as in-message link targets.
	// Dangling anchors degrade silently to an ordinary URL, so callers validate
	// their table of contents against this set.
	Anchors map[string]struct{}
}

// AssetResolver maps a vault-relative asset path to an absolute URL Telegram
// can fetch. Returning false records a LossUnresolvedMedia.
type AssetResolver func(path string) (string, bool)

// RichConverter turns a note AST into a typed rich-message block tree. It is a
// standalone sibling of HTMLConverter rather than a variant of it: the two
// share node knowledge but nothing else, and HTMLConverter stays untouched
// because the classic path, the navigation bot and canvas all still need it.
//
// The converter is pure — it performs no network IO and reaches nothing but
// the AST and its two resolvers.
type RichConverter struct {
	linkResolver  LinkResolver
	assetResolver AssetResolver

	src []byte
	res *RichConverterResult
}

func (c *RichConverter) SetLinkResolver(resolver LinkResolver) {
	c.linkResolver = resolver
}

func (c *RichConverter) SetAssetResolver(resolver AssetResolver) {
	c.assetResolver = resolver
}

func (c *RichConverter) Process(nv *model.NoteView) RichConverterResult {
	c.src = nv.Content
	c.res = &RichConverterResult{Anchors: make(map[string]struct{})}

	if doc := nv.Ast(); doc != nil {
		c.res.Blocks = c.blocks(doc)
	}

	c.res.VisibleUTF16Length = visibleLength(c.res.Blocks)

	return *c.res
}

func (c *RichConverter) loss(kind LossKind, node ast.Node, detail string) {
	c.res.Losses = append(c.res.Losses, RichLoss{
		Kind:   kind,
		Node:   node.Kind().String(),
		Detail: detail,
	})
}

// lossWithAlt records a loss that had visible alt text worth reporting.
func (c *RichConverter) lossWithAlt(kind LossKind, node ast.Node, detail, alt string) {
	c.res.Losses = append(c.res.Losses, RichLoss{
		Kind:   kind,
		Node:   node.Kind().String(),
		Detail: detail,
		Alt:    alt,
	})
}

// blocks converts every child of parent into block-level output.
func (c *RichConverter) blocks(parent ast.Node) []tgrich.Block {
	var out []tgrich.Block
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		out = append(out, c.block(child)...)
	}
	return out
}

func (c *RichConverter) block(n ast.Node) []tgrich.Block {
	switch node := n.(type) {
	case *ast.Heading:
		return c.heading(node)

	case *ast.Paragraph, *ast.TextBlock:
		return c.paragraph(n)

	case *ast.List:
		return []tgrich.Block{c.list(node)}

	case *callout.Node:
		return []tgrich.Block{c.callout(node)}

	case *ast.Blockquote:
		return []tgrich.Block{{Type: tgrich.BlockQuote, Blocks: c.blocks(node)}}

	case *ast.FencedCodeBlock:
		return []tgrich.Block{tgrich.Pre(codeText(node, c.src), string(node.Language(c.src)))}

	case *ast.CodeBlock:
		return []tgrich.Block{tgrich.Pre(codeText(node, c.src), "")}

	case *extast.Table:
		return []tgrich.Block{c.table(node)}

	case *ast.ThematicBreak:
		return []tgrich.Block{{Type: tgrich.BlockDivider}}

	case *ast.HTMLBlock:
		c.loss(LossRawHTML, node, "")
		return nil

	default:
		c.loss(LossUnsupportedNode, n, "")
		return nil
	}
}

// heading converts a heading, keeping its level.
//
// The loader's stamped id is recorded in the result's anchor set but is not
// sent: measured, the server accepts `anchor`, `name` and `id` on a heading and
// echoes none of them back, so there is no in-message anchor target to link to.
// Anchors stay in the result because a future table of contents still needs to
// know which ids exist before it can decide it cannot use them.
func (c *RichConverter) heading(node *ast.Heading) []tgrich.Block {
	if raw, ok := node.AttributeString("id"); ok {
		if id, isBytes := raw.([]byte); isBytes {
			c.res.Anchors[string(id)] = struct{}{}
		}
	}

	text, _ := c.inline(node)

	return []tgrich.Block{tgrich.Heading(node.Level, text)}
}

// paragraph converts a paragraph or list-item text block. Media alone on its
// own line becomes a media block with the alt text as its caption; media mixed
// into running text is a loss, because Telegram silently drops its caption and
// splits the paragraph around it.
func (c *RichConverter) paragraph(n ast.Node) []tgrich.Block {
	text, media := c.inline(n)

	if len(media) == 1 && text.IsEmpty() {
		if block, ok := c.mediaBlock(media[0]); ok {
			return []tgrich.Block{block}
		}
		return nil
	}

	for _, m := range media {
		c.loss(LossInlineMedia, m.node, m.dest)
	}

	if text.IsEmpty() {
		return nil
	}

	return []tgrich.Block{tgrich.Paragraph(text)}
}

func (c *RichConverter) mediaBlock(m mediaRef) (tgrich.Block, bool) {
	url, ok := c.mediaURL(m.dest)
	if !ok {
		c.loss(LossUnresolvedMedia, m.node, m.dest)
		return tgrich.Block{}, false
	}

	block := tgrich.Block{Photo: &tgrich.Media{URL: url}, Type: tgrich.BlockPhoto}
	if image.IsVideoExtension(url) {
		block = tgrich.Block{Video: &tgrich.Media{URL: url}, Type: tgrich.BlockVideo}
	}

	if alt := stripSizeSuffix(m.alt); alt != "" {
		block.Caption = &tgrich.RichText{Text: alt}
	}

	return block, true
}

// mediaURL turns a destination into something Telegram can fetch server-side.
// Only https passes through untouched: that is the only scheme documented and
// the only one measured. Anything else is the asset resolver's problem, and a
// loss if it has no answer.
func (c *RichConverter) mediaURL(dest string) (string, bool) {
	if strings.HasPrefix(dest, "https://") {
		return dest, true
	}
	if c.assetResolver == nil {
		return "", false
	}
	return c.assetResolver(dest)
}

// list converts a list. Ordering is deliberately not carried: measured, every
// spelling of it is ignored and the server labels every item "•", so an ordered
// list renders as bullets and claiming otherwise in the type would be a lie.
func (c *RichConverter) list(node *ast.List) tgrich.Block {
	block := tgrich.Block{Type: tgrich.BlockList}

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		item, ok := child.(*ast.ListItem)
		if !ok {
			c.loss(LossUnsupportedNode, child, "")
			continue
		}

		block.Items = append(block.Items, tgrich.ListItem{
			Blocks:  c.blocks(item),
			Checked: c.itemCheckbox(item),
		})
	}

	return block
}

// itemCheckbox reports the GFM task state of a list item. The checkbox is the
// first inline child of the item's text block and is skipped during inline
// rendering, so it never reaches the item's text.
func (c *RichConverter) itemCheckbox(item *ast.ListItem) *bool {
	first := item.FirstChild()
	if first == nil {
		return nil
	}

	checkbox, ok := first.FirstChild().(*extast.TaskCheckBox)
	if !ok {
		return nil
	}

	return &checkbox.IsChecked
}

// callout maps an Obsidian callout. A foldable callout is the native source of
// a details block, and its fold state carries across directly. A plain callout
// has nowhere to put its title — a blockquote takes only blocks — so it becomes
// a details block that is open by default, which keeps the title visible.
func (c *RichConverter) callout(node *callout.Node) tgrich.Block {
	title := node.Title
	if title == "" {
		title = capitalize(node.CalloutType)
	}

	block := tgrich.Block{
		Type:    tgrich.BlockDetails,
		Summary: &tgrich.RichText{Text: title},
		Blocks:  c.blocks(node),
		IsOpen:  true,
	}

	if node.Foldable {
		block.IsOpen = node.Expanded
	}

	return block
}

// table converts a GFM table into the row-major cell grid the server takes.
// Alignment is per cell here, not per column: the wire form has no column
// descriptor at all, so the column's alignment is stamped onto each of its
// cells.
func (c *RichConverter) table(node *extast.Table) tgrich.Block {
	block := tgrich.Block{Type: tgrich.BlockTable}

	for row := node.FirstChild(); row != nil; row = row.NextSibling() {
		_, header := row.(*extast.TableHeader)

		var out []tgrich.TableCell
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			text, media := c.inline(cell)
			for _, m := range media {
				c.loss(LossInlineMedia, m.node, m.dest)
			}

			out = append(out, tgrich.TableCell{
				Text:     text,
				IsHeader: header,
				Align:    columnAlign(node, len(out)),
			})
		}

		block.Cells = append(block.Cells, out)
	}

	return block
}

// columnAlign returns the alignment declared for a column index, if any.
func columnAlign(node *extast.Table, column int) tgrich.Align {
	if column >= len(node.Alignments) {
		return ""
	}

	return alignOf(node.Alignments[column])
}

func alignOf(a extast.Alignment) tgrich.Align {
	switch a {
	case extast.AlignLeft:
		return tgrich.AlignLeft
	case extast.AlignRight:
		return tgrich.AlignRight
	case extast.AlignCenter:
		return tgrich.AlignCenter
	case extast.AlignNone:
		return ""
	}
	return ""
}

// codeText returns the block's source with tabs expanded and the trailing
// newline removed.
func codeText(node ast.Node, src []byte) string {
	code := strings.TrimSuffix(string(node.Lines().Value(src)), "\n")
	return strings.ReplaceAll(code, "\t", strings.Repeat(" ", codeTabWidth))
}

// visibleLength counts the text a block tree renders, in UTF-16 code units.
func visibleLength(blocks []tgrich.Block) int {
	total := 0
	for _, block := range blocks {
		if block.Text != nil {
			total += utf16Len(block.Text.PlainText())
		}
		if block.Summary != nil {
			total += utf16Len(block.Summary.PlainText())
		}
		if block.Caption != nil {
			total += utf16Len(block.Caption.PlainText())
		}

		for _, row := range block.Cells {
			for _, cell := range row {
				total += utf16Len(cell.Text.PlainText())
			}
		}

		total += visibleLength(block.Blocks)
		for _, item := range block.Items {
			total += visibleLength(item.Blocks)
		}
	}
	return total
}

func utf16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}
