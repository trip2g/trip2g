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

type HTMLConverter struct {
	CommonConverter
	bufStack          []*strings.Builder
	linkResolver      LinkResolver
	skipClosingTag    map[ast.Node]bool
	unpublishedLinks  []unpublishedLink
	isUnpublishedLink map[ast.Node]bool
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

	c.state = lineStart
	c.skipClosingTag = make(map[ast.Node]bool)
	c.isUnpublishedLink = make(map[ast.Node]bool)
	c.unpublishedLinks = nil
	c.bufStack = nil
	c.pushBuffer() // Initialize with root buffer

	var lines []string

	_ = ast.Walk(nv.Ast(), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch node := n.(type) {
		case *ast.Document:
			// Nothing to do

		case *ast.Paragraph:
			if n.HasBlankPreviousLines() && entering {
				lines = append(lines, "\n\n")
			}

			if !entering && len(c.bufStack) == 1 {
				// End of paragraph in root buffer, add current line to lines slice
				current := c.bufStack[0]
				if current.Len() > 0 {
					lines = append(lines, current.String())
					current.Reset()
				}
			}

		case *ast.Text:
			if entering {
				text := string(node.Segment.Value(src))
				escapedText := html.EscapeString(text)

				c.Write(escapedText)
				if node.SoftLineBreak() {
					if len(c.bufStack) == 1 {
						// In root buffer: add current line to lines and start new line
						lines = append(lines, c.bufStack[0].String(), "\n")
						c.bufStack[0].Reset()
					} else {
						// In nested buffer (blockquote): just add newline
						c.Write("\n")
					}
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
				c.pushBuffer()
			} else {
				content := c.popBuffer()

				// Check if blockquote ends with spoiler (||)
				isExpandable := strings.HasSuffix(content, "||")
				if isExpandable {
					content = strings.TrimSuffix(content, "||")
					lines = append(lines, fmt.Sprintf(`<blockquote expandable>%s</blockquote>`, content))
				} else {
					lines = append(lines, fmt.Sprintf(`<blockquote>%s</blockquote>`, content))
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
				msg := fmt.Sprintf("unsupported image source: %s", dest)
				res.Warnings = append(res.Warnings, msg)
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
				msg := fmt.Sprintf("unsupported image source: %s", dest)
				res.Warnings = append(res.Warnings, msg)
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
				lines = append(lines, "\n")

				language := string(node.Language(src))
				code := string(node.Lines().Value(src))

				var codeHTML string
				if language != "" {
					codeHTML = fmt.Sprintf(`<pre><code class="language-%s">%s</code></pre>`,
						html.EscapeString(language),
						html.EscapeString(strings.TrimSuffix(code, "\n")))
				} else {
					codeHTML = fmt.Sprintf("<pre>%s</pre>",
						html.EscapeString(strings.TrimSuffix(code, "\n")))
				}

				lines = append(lines, codeHTML)
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
				lines = append(lines, "\n")
			}

		case *ast.ListItem:
			if entering {
				// Get the parent list
				parent := node.Parent()
				if list, ok := parent.(*ast.List); ok {
					// Add newline before first list item if previous content exists
					// This handles the case when list follows a paragraph without blank line
					if node.PreviousSibling() == nil {
						// Flush buffer if not empty
						if len(c.bufStack) == 1 && c.bufStack[0].Len() > 0 {
							lines = append(lines, c.bufStack[0].String())
							c.bufStack[0].Reset()
						}
						// Add newline if lines exist and last element is not already a newline
						if len(lines) > 0 && lines[len(lines)-1] != "\n" && lines[len(lines)-1] != "\n\n" {
							lines = append(lines, "\n")
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
			} else if len(c.bufStack) == 1 {
				// End of list item - flush to lines
				current := c.bufStack[0]
				if current.Len() > 0 {
					lines = append(lines, current.String())
					current.Reset()
					// Add newline if not the last item
					if node.NextSibling() != nil {
						lines = append(lines, "\n")
					}
				}
			}

		case *ast.TextBlock:
			// TextBlock is a container for text within list items - just pass through

		default:
			if entering {
				msg := fmt.Sprintf("unexpected markdown node: %s", n.Kind())
				res.Warnings = append(res.Warnings, msg)
			}

			return ast.WalkSkipChildren, nil
		}

		return ast.WalkContinue, nil
	})

	// Add any remaining content in root buffer
	if len(c.bufStack) > 0 && c.bufStack[0].Len() > 0 {
		lines = append(lines, c.bufStack[0].String())
	}

	// Add unpublished links footer if any
	if len(c.unpublishedLinks) > 0 {
		lines = append(lines, "\n\n—————————\n🔜 Скоро выйдут:")
		for _, link := range c.unpublishedLinks {
			// Format date: "5 ноября, 14:30"
			publishStr := formatPublishDate(link.publishAt)
			lines = append(lines, fmt.Sprintf("\n• <u>%s</u> — %s", html.EscapeString(link.label), publishStr))
		}
		lines = append(lines, "\n\n📬 Подпишитесь, чтобы не пропустить")
	}

	content := strings.Join(lines, "")

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
