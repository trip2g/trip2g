package markdownv2

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
	"trip2g/internal/mdloader/callout"
	"trip2g/internal/mdloader/highlight"
	"trip2g/internal/model"
	"unicode"
	"unicode/utf8"

	enclavecore "github.com/quailyquaily/goldmark-enclave/core"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"go.abhg.dev/goldmark/wikilink"
)

// ceEmojiURLPattern matches URLs like https://ce.trip2g.com/5460736117236048513.webp
var ceEmojiURLPattern = regexp.MustCompile(`^https://ce\.trip2g\.com/(\d+)\.webp$`)

// localCustomEmojiPattern matches local files like tg_ce_5460736117236048513.webp
// Matches at end of path to support any directory prefix (assets/tg_ce_*.webp, tg_ce_*.webp, etc.)
var localCustomEmojiPattern = regexp.MustCompile(`tg_ce_(\d+)\.webp$`)

// extractImageAltText extracts alt text from Image node's children.
// This replaces the deprecated node.Text(src) method for ast.Image nodes.
func extractImageAltText(node *ast.Image, src []byte) string {
	var sb strings.Builder
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if text, ok := child.(*ast.Text); ok {
			sb.Write(text.Value(src))
		}
	}
	return sb.String()
}

// extractCustomEmojiID extracts emoji ID from tg://emoji URL, ce.trip2g.com URL,
// or local tg_ce_*.webp path. Returns empty string if path doesn't match any pattern.
func extractCustomEmojiID(path string) string {
	if id, ok := strings.CutPrefix(path, "tg://emoji?id="); ok {
		return id
	}
	matches := ceEmojiURLPattern.FindStringSubmatch(path)
	if len(matches) == 2 {
		return matches[1]
	}
	matches = localCustomEmojiPattern.FindStringSubmatch(path)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

// stripSizeSuffix removes Obsidian size suffix from alt text (e.g., "🔥|20x20" -> "🔥").
func stripSizeSuffix(alt string) string {
	if idx := strings.Index(alt, "|"); idx != -1 {
		return alt[:idx]
	}
	return alt
}

type LinkResolverResult struct {
	URL       string
	Label     string
	PublishAt *time.Time
}

type LinkResolver func(string) (*LinkResolverResult, error)

type LinkResolverError struct {
	Target string
	Reason string
}

func (e *LinkResolverError) Error() string {
	return fmt.Sprintf("failed to resolve link '%s': %s", e.Target, e.Reason)
}

type unpublishedLink struct {
	label     string
	publishAt time.Time
}

// NodeHandlerFunc is called for AST nodes not handled by HTMLConverter.
// Write output via c.Write(). Return (status, true) to suppress the default warning+skip.
type NodeHandlerFunc func(c *HTMLConverter, n ast.Node, src []byte, entering bool) (ast.WalkStatus, bool)

type HTMLConverter struct {
	bufStack           []*strings.Builder
	linkResolver       LinkResolver
	skipClosingTag     map[ast.Node]bool
	unpublishedLinks   []unpublishedLink
	isUnpublishedLink  map[ast.Node]bool
	UnknownNodeHandler NodeHandlerFunc
}

func (c *HTMLConverter) SetLinkResolver(resolver LinkResolver) {
	c.linkResolver = resolver
}

// Write writes string to the current buffer (last in stack).
func (c *HTMLConverter) Write(s string) {
	if len(c.bufStack) > 0 {
		_, _ = c.bufStack[len(c.bufStack)-1].WriteString(s)
	}
}

// writePair writes the opening tag on entering and the closing tag on exit.
func (c *HTMLConverter) writePair(entering bool, open, closing string) {
	if entering {
		c.Write(open)
	} else {
		c.Write(closing)
	}
}

func (c *HTMLConverter) pushBuffer() {
	c.bufStack = append(c.bufStack, &strings.Builder{})
}

func (c *HTMLConverter) popBuffer() string {
	if len(c.bufStack) == 0 {
		return ""
	}
	lastIdx := len(c.bufStack) - 1
	result := c.bufStack[lastIdx].String()
	c.bufStack = c.bufStack[:lastIdx]
	return result
}

// currentBuffer returns the accumulated content of the buffer being written to.
func (c *HTMLConverter) currentBuffer() string {
	if len(c.bufStack) == 0 {
		return ""
	}
	return c.bufStack[len(c.bufStack)-1].String()
}

func (c *HTMLConverter) Process(nv *model.NoteView) ConverterResult {
	res := &ConverterResult{}
	src := nv.Content

	c.skipClosingTag = make(map[ast.Node]bool)
	c.isUnpublishedLink = make(map[ast.Node]bool)
	c.unpublishedLinks = nil
	c.bufStack = nil
	c.pushBuffer() // Initialize with root buffer

	_ = ast.Walk(nv.Ast(), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		return c.renderNode(n, src, entering, res), nil
	})

	c.writeUnpublishedFooter()

	content := c.bufStack[0].String()

	// Remove excessive blank lines (more than 2 newlines in a row)
	// This happens when media files are removed from paragraphs
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}

	// Trim leading/trailing whitespace (e.g., when cover image is removed)
	res.Content = strings.TrimSpace(content)

	return *res
}

//nolint:gocyclo,cyclop,gocognit // flat per-node dispatch
func (c *HTMLConverter) renderNode(n ast.Node, src []byte, entering bool, res *ConverterResult) ast.WalkStatus {
	switch node := n.(type) {
	case *ast.Document:
		// Nothing to do

	case *ast.Paragraph:
		if entering && n.HasBlankPreviousLines() {
			c.Write("\n\n")
		}

	case *ast.Text:
		if entering {
			c.Write(html.EscapeString(string(node.Segment.Value(src))))
			if node.SoftLineBreak() {
				c.Write("\n")
			}
		}

	case *ast.Emphasis:
		if node.Level == 1 {
			c.writePair(entering, "<i>", "</i>")
		} else {
			c.writePair(entering, "<b>", "</b>")
		}

	case *extast.Strikethrough:
		c.writePair(entering, "<s>", "</s>")

	case *ast.CodeSpan:
		c.writePair(entering, "<code>", "</code>")

	case *highlight.HighlightAST:
		c.writePair(entering, `<span class="tg-spoiler">`, "</span>")

	case *ast.Blockquote:
		c.renderBlockquote(n, entering)

	case *callout.Node:
		c.renderCallout(node, entering)

	case *ast.Link:
		if entering {
			c.Write(fmt.Sprintf(`<a href="%s">`, html.EscapeString(string(node.Destination))))
		} else {
			c.Write("</a>")
		}

	case *ast.AutoLink:
		if entering {
			url := string(node.URL(src))
			c.Write(fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(url), html.EscapeString(url)))
		}

	case *enclavecore.Enclave:
		return c.renderCustomEmoji(string(node.Destination), node.Alt, entering, res)

	case *ast.Image:
		return c.renderCustomEmoji(string(node.Destination), extractImageAltText(node, src), entering, res)

	case *ast.RawHTML:
		if entering {
			tag := string(node.Segments.Value(src))
			if tag == tagUnderlineOpen || tag == tagUnderlineClose {
				c.Write(tag)
			} else {
				res.Warnings = append(res.Warnings, fmt.Sprintf("raw html tag is not supported: %s", tag))
			}
		}

	case *ast.FencedCodeBlock:
		if entering {
			c.renderFencedCodeBlock(node, src)
		}

	case *wikilink.Node:
		return c.renderWikilink(node, entering, res)

	case *ast.List:
		if entering && n.HasBlankPreviousLines() {
			c.Write("\n")
		}

	case *ast.ListItem:
		c.renderListItem(node, entering)

	case *ast.TextBlock:
		// TextBlock is a container for text within list items - just pass through

	default:
		if c.UnknownNodeHandler != nil {
			status, handled := c.UnknownNodeHandler(c, n, src, entering)
			if handled {
				return status
			}
		}
		if entering {
			res.Warnings = append(res.Warnings, fmt.Sprintf("unexpected markdown node: %s", n.Kind()))
		}

		return ast.WalkSkipChildren
	}

	return ast.WalkContinue
}

// renderBlockquote collects the quote content in its own buffer and wraps it
// into <blockquote> in the parent buffer on exit.
func (c *HTMLConverter) renderBlockquote(n ast.Node, entering bool) {
	if entering {
		if n.HasBlankPreviousLines() {
			c.Write("\n\n")
		}
		c.pushBuffer()
		return
	}

	content := strings.TrimSpace(c.popBuffer())

	// Blockquote ending with || means Telegram expandable quote
	if inner, ok := strings.CutSuffix(content, "||"); ok {
		c.writeQuote(inner, true)
	} else {
		c.writeQuote(content, false)
	}
}

// renderCallout renders an Obsidian callout as a blockquote with a bold title
// line, collecting the body in its own buffer like renderBlockquote does.
// A collapsed callout (`[!type]-`) maps to an expandable quote: Telegram shows
// its first line — the title — and hides the body behind an expand control.
func (c *HTMLConverter) renderCallout(node *callout.Node, entering bool) {
	if entering {
		// The parser builds a fresh node, so HasBlankPreviousLines is always
		// false here; separate from preceding content by what was written.
		if cur := c.currentBuffer(); cur != "" && !strings.HasSuffix(cur, "\n\n") {
			c.Write("\n\n")
		}
		c.pushBuffer()
		return
	}

	body := strings.TrimSpace(c.popBuffer())

	title := node.Title
	if title == "" {
		title = capitalize(node.CalloutType)
	}

	content := fmt.Sprintf("<b>%s</b>", html.EscapeString(title))
	if body != "" {
		content += "\n" + body
	}

	c.writeQuote(content, node.Foldable && !node.Expanded)
}

// writeQuote wraps content into a Telegram blockquote, expandable or not.
func (c *HTMLConverter) writeQuote(content string, expandable bool) {
	if expandable {
		c.Write(fmt.Sprintf(`<blockquote expandable>%s</blockquote>`, content))
	} else {
		c.Write(fmt.Sprintf(`<blockquote>%s</blockquote>`, content))
	}
}

// capitalize upper-cases the first rune of s, matching the web callout renderer.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}

// renderCustomEmoji renders a Telegram custom emoji, or warns on an unsupported
// image source. Warns only on entering: ast.Walk visits the node again with
// entering=false even after WalkSkipChildren.
func (c *HTMLConverter) renderCustomEmoji(dest, alt string, entering bool, res *ConverterResult) ast.WalkStatus {
	emojiID := extractCustomEmojiID(dest)
	if !entering {
		return ast.WalkSkipChildren
	}

	if emojiID == "" {
		res.Warnings = append(res.Warnings, fmt.Sprintf("unsupported image source: %s", dest))
		return ast.WalkSkipChildren
	}

	c.Write(fmt.Sprintf(`<tg-emoji emoji-id="%s">%s</tg-emoji>`,
		html.EscapeString(emojiID), html.EscapeString(stripSizeSuffix(alt))))
	return ast.WalkSkipChildren
}

func (c *HTMLConverter) renderFencedCodeBlock(node *ast.FencedCodeBlock, src []byte) {
	c.Write("\n")

	language := string(node.Language(src))
	code := html.EscapeString(strings.TrimSuffix(string(node.Lines().Value(src)), "\n"))

	if language != "" {
		c.Write(fmt.Sprintf(`<pre><code class="language-%s">%s</code></pre>`,
			html.EscapeString(language), code))
	} else {
		c.Write(fmt.Sprintf("<pre>%s</pre>", code))
	}
}

func (c *HTMLConverter) renderWikilink(node *wikilink.Node, entering bool, res *ConverterResult) ast.WalkStatus {
	if c.linkResolver == nil || node.Embed {
		return ast.WalkSkipChildren
	}

	if !entering {
		if !c.skipClosingTag[node] {
			if c.isUnpublishedLink[node] {
				c.Write(tagUnderlineClose)
			} else {
				c.Write("</a>")
			}
		}
		delete(c.skipClosingTag, node)
		delete(c.isUnpublishedLink, node)
		return ast.WalkContinue
	}

	link, err := c.linkResolver(string(node.Target))
	if err != nil {
		res.Warnings = append(res.Warnings, err.Error())
		c.skipClosingTag[node] = true
		return ast.WalkSkipChildren
	}
	if link == nil {
		// Resolver returned nothing - drop the link like the no-URL/no-Label case
		c.skipClosingTag[node] = true
		return ast.WalkSkipChildren
	}

	switch {
	case link.URL != "":
		c.Write(fmt.Sprintf(`<a href="%s">`, html.EscapeString(link.URL)))

		// If Label is provided, use it instead of node children
		if link.Label != "" {
			c.Write(html.EscapeString(link.Label))
			c.Write("</a>")
			c.skipClosingTag[node] = true
			return ast.WalkSkipChildren
		}
		return ast.WalkContinue

	case link.Label != "":
		// Unpublished link: render as underlined label, list in the footer if it has a date
		if link.PublishAt != nil {
			c.unpublishedLinks = append(c.unpublishedLinks, unpublishedLink{
				label:     link.Label,
				publishAt: *link.PublishAt,
			})
		}

		c.isUnpublishedLink[node] = true
		c.Write(fmt.Sprintf("<u>%s</u>", html.EscapeString(link.Label)))
		c.skipClosingTag[node] = true
		return ast.WalkSkipChildren

	default:
		// No URL and no Label - skip
		c.skipClosingTag[node] = true
		return ast.WalkSkipChildren
	}
}

func (c *HTMLConverter) renderListItem(node *ast.ListItem, entering bool) {
	if !entering {
		if node.NextSibling() != nil {
			c.Write("\n")
		}
		return
	}

	list, ok := node.Parent().(*ast.List)
	if !ok {
		return
	}

	// Start list on a new line when it follows a paragraph without blank line
	if node.PreviousSibling() == nil {
		if cur := c.currentBuffer(); cur != "" && !strings.HasSuffix(cur, "\n") {
			c.Write("\n")
		}
	}

	if list.IsOrdered() {
		itemNum := max(list.Start, 1)
		for prev := node.PreviousSibling(); prev != nil; prev = prev.PreviousSibling() {
			itemNum++
		}
		c.Write(fmt.Sprintf("%d. ", itemNum))
	} else {
		c.Write("- ")
	}
}

func (c *HTMLConverter) writeUnpublishedFooter() {
	if len(c.unpublishedLinks) == 0 {
		return
	}
	c.Write("\n\n—————————\n🔜 Скоро выйдут:")
	for _, link := range c.unpublishedLinks {
		c.Write(fmt.Sprintf("\n• <u>%s</u> — %s", html.EscapeString(link.label), formatPublishDate(link.publishAt)))
	}
	c.Write("\n\n📬 Подпишитесь, чтобы не пропустить")
}

func formatPublishDate(t time.Time) string {
	// Format: "5 ноября, 14:30"
	months := []string{
		"января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря",
	}

	month := months[t.Month()-1]
	return fmt.Sprintf("%d %s, %02d:%02d", t.Day(), month, t.Hour(), t.Minute())
}
