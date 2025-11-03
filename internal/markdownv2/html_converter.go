package markdownv2

import (
	"fmt"
	"html"
	"strings"
	"time"
	"trip2g/internal/mdloader/highlight"
	"trip2g/internal/model"

	enclavecore "github.com/quailyquaily/goldmark-enclave/core"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"go.abhg.dev/goldmark/wikilink"
)

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

		case *enclavecore.Enclave:
			dest := string(node.Destination)
			if strings.HasPrefix(dest, "tg://emoji?id=") {
				emojiID := strings.TrimPrefix(dest, "tg://emoji?id=")
				if entering {
					c.Write(fmt.Sprintf(`<tg-emoji emoji-id="%s">`, html.EscapeString(emojiID)))
				} else {
					c.Write("</tg-emoji>")
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

					// Check if this is an unpublished link (no URL but has PublishAt)
					switch {
					case link.URL == "" && link.PublishAt != nil:
						// Get label from wikilink (fragment text or target)
						label := link.Label
						if label == "" {
							label = dest
						}

						// Add to unpublished links list
						c.unpublishedLinks = append(c.unpublishedLinks, unpublishedLink{
							label:     label,
							publishAt: *link.PublishAt,
						})

						// Mark this node as unpublished link
						c.isUnpublishedLink[n] = true

						// Write underlined text with label instead of link
						c.Write(fmt.Sprintf("<u>%s</u>", html.EscapeString(label)))

						// Skip children since we already wrote the label
						c.skipClosingTag[n] = true
						return ast.WalkSkipChildren, nil
					case link.URL != "":
						c.Write(fmt.Sprintf(`<a href="%s">`, html.EscapeString(link.URL)))
					default:
						// No URL and no PublishAt - skip
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

	res.Content = strings.Join(lines, "")

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
