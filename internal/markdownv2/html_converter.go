package markdownv2

import (
	"fmt"
	"html"
	"strings"
	"trip2g/internal/mdloader/highlight"
	"trip2g/internal/model"

	enclavecore "github.com/quailyquaily/goldmark-enclave/core"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"go.abhg.dev/goldmark/wikilink"
)

type LinkResolver func(string) (string, error)

type LinkResolverError struct {
	Target string
	Reason string
}

func (e *LinkResolverError) Error() string {
	return fmt.Sprintf("failed to resolve link '%s': %s", e.Target, e.Reason)
}

type HTMLConverter struct {
	CommonConverter
	bufStack       []*strings.Builder
	linkResolver   LinkResolver
	skipClosingTag map[ast.Node]bool
}

func (c *HTMLConverter) SetLinkResolver(resolver LinkResolver) {
	c.linkResolver = resolver
}

// Write writes string to the current buffer (last in stack).
func (c *HTMLConverter) Write(s string) {
	if len(c.bufStack) > 0 {
		c.bufStack[len(c.bufStack)-1].WriteString(s)
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

func (c *HTMLConverter) Process(nv *model.NoteView) ConverterResult {
	res := ConverterResult{}
	src := nv.Content

	c.state = lineStart
	c.skipClosingTag = make(map[ast.Node]bool)
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
				code := string(node.Text(src))

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
					url, err := c.linkResolver(dest)
					if err != nil {
						res.Warnings = append(res.Warnings, err.Error())
						c.skipClosingTag[n] = true
						return ast.WalkSkipChildren, nil
					}

					c.Write(fmt.Sprintf(`<a href="%s">`, html.EscapeString(url)))
				} else {
					if !c.skipClosingTag[n] {
						c.Write("</a>")
					}
					delete(c.skipClosingTag, n)
				}
			} else {
				// No link resolver or embed - skip this node
				return ast.WalkSkipChildren, nil
			}

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

	res.Content = strings.Join(lines, "")

	return res
}
