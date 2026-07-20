package markdownv2

import (
	"trip2g/internal/mdloader/highlight"
	"trip2g/internal/tgrich"

	enclavecore "github.com/quailyquaily/goldmark-enclave/core"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"go.abhg.dev/goldmark/wikilink"
)

// mediaRef is a media node found while walking inline content. Media is hoisted
// out of the text run and decided by the caller, because a rich message carries
// it as its own block.
type mediaRef struct {
	node ast.Node
	dest string
	alt  string
}

// inlineStyle is the mark set inherited from the enclosing inline nodes.
type inlineStyle struct {
	bold      bool
	italic    bool
	strike    bool
	marked    bool
	code      bool
	underline bool
	url       string
}

// inlineBuilder accumulates the runs of one block-level node.
type inlineBuilder struct {
	c     *RichConverter
	runs  []tgrich.RichText
	media []mediaRef
	// underline is toggled by the raw <u> and </u> tags rather than inherited,
	// because they are siblings in the AST, not a container.
	underline bool
}

// inline renders the inline children of n, returning its text and any media it
// carried.
func (c *RichConverter) inline(n ast.Node) (tgrich.RichText, []mediaRef) {
	b := &inlineBuilder{c: c}
	b.children(n, inlineStyle{})

	return b.text(), b.media
}

func (b *inlineBuilder) children(n ast.Node, style inlineStyle) {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		b.node(child, style)
	}
}

func (b *inlineBuilder) node(n ast.Node, style inlineStyle) {
	switch node := n.(type) {
	case *ast.Text:
		b.write(string(node.Segment.Value(b.c.src)), style)
		if node.SoftLineBreak() || node.HardLineBreak() {
			b.write("\n", style)
		}

	case *ast.String:
		b.write(string(node.Value), style)

	case *ast.Emphasis:
		if node.Level == 1 {
			style.italic = true
		} else {
			style.bold = true
		}
		b.children(node, style)

	case *extast.Strikethrough:
		style.strike = true
		b.children(node, style)

	case *highlight.HighlightAST:
		// Real highlight, unlike the spoiler the classic HTML path emits.
		style.marked = true
		b.children(node, style)

	case *ast.CodeSpan:
		style.code = true
		b.children(node, style)

	case *ast.Link:
		style.url = string(node.Destination)
		b.children(node, style)

	case *ast.AutoLink:
		url := string(node.URL(b.c.src))
		style.url = url
		b.write(url, style)

	case *extast.TaskCheckBox:
		// Carried on the list item as Checked, never rendered into its text.

	case *enclavecore.Enclave:
		b.mediaNode(node, string(node.Destination), node.Alt)

	case *ast.Image:
		b.mediaNode(node, string(node.Destination), extractImageAltText(node, b.c.src))

	case *wikilink.Node:
		b.wikilink(node, style)

	case *ast.RawHTML:
		b.rawHTML(node)

	default:
		b.c.loss(LossUnsupportedNode, n, "")
	}
}

// write appends one run. Runs are never merged: a caller comparing output to
// source order needs the boundaries the markdown actually had.
func (b *inlineBuilder) write(text string, style inlineStyle) {
	if text == "" {
		return
	}

	b.runs = append(b.runs, tgrich.RichText{
		Text:      text,
		Bold:      style.bold,
		Italic:    style.italic,
		Underline: style.underline || b.underline,
		Strike:    style.strike,
		Marked:    style.marked,
		Code:      style.code,
		URL:       style.url,
	})
}

// mediaNode records real media for the caller and reports a custom emoji as a
// loss. A custom emoji is deliberately not emitted in any form: the server
// substitutes the sticker set's own fallback for the id, and a word in that
// slot is rejected outright with RICH_MESSAGE_EMOJI_INVALID.
func (b *inlineBuilder) mediaNode(n ast.Node, dest, alt string) {
	if emojiID := extractCustomEmojiID(dest); emojiID != "" {
		// Carry the alt text: it is the only human-readable trace of what the
		// reader lost, and the id alone cannot be reported to anyone usefully.
		b.c.lossWithAlt(LossCustomEmoji, n, emojiID, alt)
		return
	}

	b.media = append(b.media, mediaRef{node: n, dest: dest, alt: alt})
}

func (b *inlineBuilder) wikilink(node *wikilink.Node, style inlineStyle) {
	target := string(node.Target)

	if node.Embed {
		b.c.loss(LossEmbeddedWikiLink, node, target)
		return
	}

	if b.c.linkResolver == nil {
		b.c.loss(LossUnresolvedWikiLink, node, target)
		return
	}

	link, err := b.c.linkResolver(target)
	if err != nil || link == nil {
		b.c.loss(LossUnresolvedWikiLink, node, target)
		return
	}

	switch {
	case link.URL != "":
		style.url = link.URL
		if link.Label != "" {
			b.write(link.Label, style)
			return
		}
		b.children(node, style)

	case link.Label != "":
		// Not published yet: underline the label, exactly like the classic path.
		style.underline = true
		b.write(link.Label, style)

	default:
		b.c.loss(LossUnresolvedWikiLink, node, target)
	}
}

// rawHTML supports only the <u> pair, which maps onto a real underline mark.
// Everything else is a loss: rich blocks carry marks, not markup, so there is
// nowhere for a tag to go.
func (b *inlineBuilder) rawHTML(node *ast.RawHTML) {
	switch tag := string(node.Segments.Value(b.c.src)); tag {
	case tagUnderlineOpen:
		b.underline = true
	case tagUnderlineClose:
		b.underline = false
	default:
		b.c.loss(LossRawHTML, node, tag)
	}
}

// text collapses the accumulated runs. A single unmarked run becomes plain
// text so it marshals as a bare JSON string.
func (b *inlineBuilder) text() tgrich.RichText {
	switch {
	case len(b.runs) == 0:
		return tgrich.RichText{}
	case len(b.runs) == 1 && b.runs[0].IsPlain():
		return b.runs[0]
	default:
		return tgrich.RichText{Children: b.runs}
	}
}
