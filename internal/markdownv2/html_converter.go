package markdownv2

import (
	"bytes"
	"fmt"
	"html"
	"strings"
	"trip2g/internal/mdloader/highlight"
	"trip2g/internal/model"

	enclavecore "github.com/quailyquaily/goldmark-enclave/core"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

type HTMLConverter struct {
	CommonConverter
	blockquoteContent strings.Builder
	inBlockquote      bool
}

func (c *HTMLConverter) Process(nv *model.NoteView) ConverterResult {
	res := ConverterResult{}
	src := nv.Content

	c.state = lineStart
	c.inBlockquote = false
	c.blockquoteContent.Reset()

	var buf bytes.Buffer
	var lines []string

	_ = ast.Walk(nv.Ast(), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch node := n.(type) {
		case *ast.Document:
			// Nothing to do

		case *ast.Paragraph:
			if !c.inBlockquote {
				if !entering {
					// End of paragraph, add current line to lines slice
					if buf.Len() > 0 {
						lines = append(lines, buf.String())
						buf.Reset()
					}
				}
			}

		case *ast.Text:
			if entering {
				text := string(node.Segment.Value(src))
				escapedText := html.EscapeString(text)

				if c.inBlockquote {
					c.blockquoteContent.WriteString(escapedText)
					if node.SoftLineBreak() {
						c.blockquoteContent.WriteString("\\n")
					}
				} else {
					buf.WriteString(escapedText)
					if node.SoftLineBreak() {
						// Add current line to lines and start new line
						lines = append(lines, buf.String())
						buf.Reset()
					}
				}
			}

		case *ast.Emphasis:
			tag := ""
			if emphasis := node.Level; emphasis == 1 {
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

			if c.inBlockquote {
				c.blockquoteContent.WriteString(tag)
			} else {
				buf.WriteString(tag)
			}

		case *extast.Strikethrough:
			tag := ""
			if entering {
				tag = "<s>"
			} else {
				tag = "</s>"
			}

			if c.inBlockquote {
				c.blockquoteContent.WriteString(tag)
			} else {
				buf.WriteString(tag)
			}

		case *ast.CodeSpan:
			tag := ""
			if entering {
				tag = "<code>"
			} else {
				tag = "</code>"
			}

			if c.inBlockquote {
				c.blockquoteContent.WriteString(tag)
			} else {
				buf.WriteString(tag)
			}

		case *highlight.HighlightAST:
			if c.inBlockquote {
				if entering {
					c.blockquoteContent.WriteString(`<span class="tg-spoiler">`)
				} else {
					c.blockquoteContent.WriteString("</span>")
				}
			} else {
				if entering {
					buf.WriteString(`<span class="tg-spoiler">`)
				} else {
					buf.WriteString("</span>")
				}
			}

		case *ast.Blockquote:
			if entering {
				c.inBlockquote = true
				c.blockquoteContent.Reset()
			} else {
				c.inBlockquote = false
				content := c.blockquoteContent.String()

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
			linkHTML := ""
			if entering {
				linkHTML = fmt.Sprintf(`<a href="%s">`, html.EscapeString(string(node.Destination)))
			} else {
				linkHTML = "</a>"
			}

			if c.inBlockquote {
				c.blockquoteContent.WriteString(linkHTML)
			} else {
				buf.WriteString(linkHTML)
			}

		case *enclavecore.Enclave:
			dest := string(node.Destination)
			if strings.HasPrefix(dest, "tg://emoji?id=") {
				emojiID := strings.TrimPrefix(dest, "tg://emoji?id=")
				emojiHTML := ""
				if entering {
					emojiHTML = fmt.Sprintf(`<tg-emoji emoji-id="%s">`, html.EscapeString(emojiID))
				} else {
					emojiHTML = "</tg-emoji>"
				}

				if c.inBlockquote {
					c.blockquoteContent.WriteString(emojiHTML)
				} else {
					buf.WriteString(emojiHTML)
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
					if c.inBlockquote {
						c.blockquoteContent.WriteString(tag)
					} else {
						buf.WriteString(tag)
					}
				} else {
					msg := fmt.Sprintf("raw html tag is not supported: %s", tag)
					res.Warnings = append(res.Warnings, msg)
				}
			}

		case *ast.FencedCodeBlock:
			if entering {
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

		default:
			if entering {
				msg := fmt.Sprintf("unexpected markdown node: %s", n.Kind())
				res.Warnings = append(res.Warnings, msg)
			}

			return ast.WalkSkipChildren, nil
		}

		return ast.WalkContinue, nil
	})

	// Add any remaining content in buffer
	if buf.Len() > 0 {
		lines = append(lines, buf.String())
	}

	res.Content = strings.Join(lines, "\n")

	return res
}
