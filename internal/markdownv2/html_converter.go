package markdownv2

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
	"trip2g/internal/mdloader/highlight"
	"trip2g/internal/model"

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

// extractCustomEmojiID extracts emoji ID from ce.trip2g.com URL or local tg_ce_*.webp path.
// Returns empty string if path doesn't match any pattern.
func extractCustomEmojiID(path string) string {
	// Try ce.trip2g.com URL pattern
	matches := ceEmojiURLPattern.FindStringSubmatch(path)
	if len(matches) == 2 {
		return matches[1]
	}
	// Try local tg_ce_*.webp pattern
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
	CommonConverter
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

//nolint:nestif,gocognit,gocyclo,cyclop,funlen // ast traversal always looks like this
func (c *HTMLConverter) Process(nv *model.NoteView) ConverterResult {
	res := ConverterResult{}
	src := nv.Content

	c.skipClosingTag = make(map[ast.Node]bool)
	c.isUnpublishedLink = make(map[ast.Node]bool)
	c.unpublishedLinks = nil
	c.bufStack = nil
	c.pushBuffer() // Initialize with root buffer

	_ = ast.Walk(nv.Ast(), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch node := n.(type) {
		case *ast.Document:
			// Nothing to do

		case *ast.Paragraph:
			if n.HasBlankPreviousLines() && entering {
				c.Write("\n\n")
			}

		case *ast.Text:
			if entering {
				text := string(node.Segment.Value(src))
				c.Write(html.EscapeString(text))
				if node.SoftLineBreak() {
					c.Write("\n")
				}
			}

		case *ast.Emphasis:
			var tag string
			if node.Level == 1 {
				if entering {
					tag = "<i>"
				} else {
					tag = "</i>"
				}
			} else {
				if entering {
					tag = "<b>"
				} else {
					tag = "</b>"
				}
			}
			c.Write(tag)

		case *extast.Strikethrough:
			var tag string
			if entering {
				tag = "<s>"
			} else {
				tag = "</s>"
			}
			c.Write(tag)

		case *ast.CodeSpan:
			var tag string
			if entering {
				tag = "<code>"
			} else {
				tag = "</code>"
			}
			c.Write(tag)

		case *highlight.HighlightAST:
			if entering {
				c.Write(`<span class="tg-spoiler">`)
			} else {
				c.Write("</span>")
			}

		case *ast.Blockquote:
			if entering {
				if n.HasBlankPreviousLines() {
					c.Write("\n\n")
				}
				c.pushBuffer()
			} else {
				content := strings.TrimSpace(c.popBuffer())

				// Blockquote ending with || means Telegram expandable quote
				if strings.HasSuffix(content, "||") {
					content = strings.TrimSuffix(content, "||")
					c.Write(fmt.Sprintf(`<blockquote expandable>%s</blockquote>`, content))
				} else {
					c.Write(fmt.Sprintf(`<blockquote>%s</blockquote>`, content))
				}
			}

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
			dest := string(node.Destination)
			var emojiID string

			if strings.HasPrefix(dest, "tg://emoji?id=") {
				emojiID = strings.TrimPrefix(dest, "tg://emoji?id=")
			} else if id := extractCustomEmojiID(dest); id != "" {
				emojiID = id
			}

			if emojiID != "" {
				if entering {
					alt := stripSizeSuffix(node.Alt)
					c.Write(fmt.Sprintf(`<tg-emoji emoji-id="%s">%s</tg-emoji>`,
						html.EscapeString(emojiID), html.EscapeString(alt)))
					return ast.WalkSkipChildren, nil
				}
			} else {
				// ast.Walk visits the node again with entering=false even after
				// WalkSkipChildren, so warn only once.
				if entering {
					msg := fmt.Sprintf("unsupported image source: %s", dest)
					res.Warnings = append(res.Warnings, msg)
				}
				return ast.WalkSkipChildren, nil
			}

		case *ast.Image:
			dest := string(node.Destination)
			var emojiID string

			if strings.HasPrefix(dest, "tg://emoji?id=") {
				emojiID = strings.TrimPrefix(dest, "tg://emoji?id=")
			} else if id := extractCustomEmojiID(dest); id != "" {
				emojiID = id
			}

			if emojiID != "" {
				if entering {
					alt := stripSizeSuffix(extractImageAltText(node, src))
					c.Write(fmt.Sprintf(`<tg-emoji emoji-id="%s">%s</tg-emoji>`,
						html.EscapeString(emojiID), html.EscapeString(alt)))
					return ast.WalkSkipChildren, nil
				}
			} else {
				if entering {
					msg := fmt.Sprintf("unsupported image source: %s", dest)
					res.Warnings = append(res.Warnings, msg)
				}
				return ast.WalkSkipChildren, nil
			}

		case *ast.RawHTML:
			if entering {
				tag := string(node.Segments.Value(src))
				if tag == "<u>" || tag == "</u>" {
					c.Write(tag)
				} else {
					msg := fmt.Sprintf("raw html tag is not supported: %s", tag)
					res.Warnings = append(res.Warnings, msg)
				}
			}

		case *ast.FencedCodeBlock:
			if entering {
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

		case *wikilink.Node:
			if c.linkResolver != nil && !node.Embed {
				if entering {
					dest := string(node.Target)
					link, err := c.linkResolver(dest)
					if err != nil {
						res.Warnings = append(res.Warnings, err.Error())
						c.skipClosingTag[n] = true
						return ast.WalkSkipChildren, nil
					}

					// Handle different link types
					switch {
					case link.URL != "":
						// Regular link with URL
						c.Write(fmt.Sprintf(`<a href="%s">`, html.EscapeString(link.URL)))

						// If Label is provided, use it instead of node children
						if link.Label != "" {
							c.Write(html.EscapeString(link.Label))
							c.Write("</a>")
							c.skipClosingTag[n] = true
							return ast.WalkSkipChildren, nil
						}
					case link.Label != "":
						// Unpublished link with label (with or without PublishAt)
						label := link.Label

						// If has PublishAt, add to footer list
						if link.PublishAt != nil {
							c.unpublishedLinks = append(c.unpublishedLinks, unpublishedLink{
								label:     label,
								publishAt: *link.PublishAt,
							})
						}

						// Mark this node as unpublished link
						c.isUnpublishedLink[n] = true

						// Write underlined text with label instead of link
						c.Write(fmt.Sprintf("<u>%s</u>", html.EscapeString(label)))

						// Skip children since we already wrote the label
						c.skipClosingTag[n] = true
						return ast.WalkSkipChildren, nil
					default:
						// No URL and no Label - skip
						c.skipClosingTag[n] = true
						return ast.WalkSkipChildren, nil
					}
				} else {
					if !c.skipClosingTag[n] {
						// Check if this was an unpublished link (underlined text)
						if c.isUnpublishedLink[n] {
							c.Write("</u>")
						} else {
							c.Write("</a>")
						}
					}
					delete(c.skipClosingTag, n)
					delete(c.isUnpublishedLink, n)
				}
			} else {
				// No link resolver or embed - skip this node
				return ast.WalkSkipChildren, nil
			}

		case *ast.List:
			if entering && n.HasBlankPreviousLines() {
				c.Write("\n")
			}

		case *ast.ListItem:
			if entering {
				if list, ok := node.Parent().(*ast.List); ok {
					// Start list on a new line when it follows a paragraph without blank line
					if node.PreviousSibling() == nil {
						cur := c.bufStack[len(c.bufStack)-1].String()
						if cur != "" && !strings.HasSuffix(cur, "\n") {
							c.Write("\n")
						}
					}

					if list.IsOrdered() {
						// For ordered lists, calculate item number based on child index
						itemNum := 1
						for prev := node.PreviousSibling(); prev != nil; prev = prev.PreviousSibling() {
							itemNum++
						}
						c.Write(fmt.Sprintf("%d. ", itemNum))
					} else {
						// For unordered lists, use dash
						c.Write("- ")
					}
				}
			} else if node.NextSibling() != nil {
				c.Write("\n")
			}

		case *ast.TextBlock:
			// TextBlock is a container for text within list items - just pass through

		default:
			if c.UnknownNodeHandler != nil {
				status, handled := c.UnknownNodeHandler(c, n, src, entering)
				if handled {
					return status, nil
				}
			}
			if entering {
				msg := fmt.Sprintf("unexpected markdown node: %s", n.Kind())
				res.Warnings = append(res.Warnings, msg)
			}

			return ast.WalkSkipChildren, nil
		}

		return ast.WalkContinue, nil
	})

	// Add unpublished links footer if any
	if len(c.unpublishedLinks) > 0 {
		c.Write("\n\n—————————\n🔜 Скоро выйдут:")
		for _, link := range c.unpublishedLinks {
			publishStr := formatPublishDate(link.publishAt)
			c.Write(fmt.Sprintf("\n• <u>%s</u> — %s", html.EscapeString(link.label), publishStr))
		}
		c.Write("\n\n📬 Подпишитесь, чтобы не пропустить")
	}

	content := c.bufStack[0].String()

	// Remove excessive blank lines (more than 2 newlines in a row)
	// This happens when media files are removed from paragraphs
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}

	// Trim leading/trailing whitespace (e.g., when cover image is removed)
	content = strings.TrimSpace(content)

	res.Content = content

	return res
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
