package markdownv2

import (
	"bytes"
	"fmt"
	"strings"
	"trip2g/internal/mdloader/highlight"
	"trip2g/internal/model"

	enclavecore "github.com/quailyquaily/goldmark-enclave/core"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

type lineState int

const (
	midLine lineState = iota
	needNewline
	lineStart
)

type ConverterResult struct {
	Warnings []string
	Content  string
	Assets   []string
}

type CommonConverter struct {
	insideBlockquote bool
	state            lineState
}

func (c *CommonConverter) Process(nv *model.NoteView) ConverterResult {
	res := ConverterResult{}
	src := nv.Content

	c.state = lineStart

	var buf bytes.Buffer

	_ = ast.Walk(nv.Ast(), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		// Handle newlines for non-text nodes
		if _, ok := n.(*ast.Text); !ok && c.state == needNewline {
			buf.WriteString("\n")
			c.state = lineStart
		}

		switch node := n.(type) {
		case *ast.Document:
			// Nothing to do

		case *ast.Paragraph:
			if !entering {
				c.state = needNewline
			}

		case *ast.Text:
			if entering {
				if c.state == needNewline {
					buf.WriteString("\n")
					c.state = lineStart
				}

				if c.insideBlockquote && c.state == lineStart && !node.SoftLineBreak() {
					buf.WriteString(">")
					c.state = midLine
				}

				buf.Write(node.Segment.Value(src))
				if node.SoftLineBreak() {
					c.state = needNewline
				} else {
					c.state = midLine
				}
			}

		case *ast.Emphasis:
			if emphasis := node.Level; emphasis == 1 {
				buf.WriteString("_")
			} else {
				buf.WriteString("*")
			}

		case *extast.Strikethrough:
			buf.WriteString("~")

		case *ast.CodeSpan:
			buf.WriteString("`")

		case *highlight.HighlightAST:
			buf.WriteString("||")

		case *ast.Blockquote:
			c.insideBlockquote = entering
			if !entering {
				c.state = needNewline
			}

		case *ast.Link:
			if entering {
				buf.WriteString("[")
			} else {
				buf.WriteString("](")
				buf.Write(node.Destination)
				buf.WriteString(")")
			}

		case *enclavecore.Enclave:
			dest := string(node.Destination)
			if !strings.HasPrefix(dest, "tg://emoji?id=") {
				msg := fmt.Sprintf("unsupported image source: %s", dest)
				res.Warnings = append(res.Warnings, msg)
				return ast.WalkSkipChildren, nil
			}

			if entering {
				buf.WriteString("![")
			} else {
				buf.WriteString("](")
				buf.WriteString(dest)
				buf.WriteString(")")
			}

		case *ast.RawHTML:
			if entering {
				tag := string(node.Segments.Value(src))
				if tag == "<u>" || tag == "</u>" {
					buf.WriteString("__")
				} else {
					msg := fmt.Sprintf("raw html tag is not supported: %s", tag)
					res.Warnings = append(res.Warnings, msg)
				}
			}

		case *ast.FencedCodeBlock:
			if entering {
				if c.state == needNewline {
					buf.WriteString("\n")
					c.state = lineStart
				}
				buf.WriteString("```")
				buf.Write(node.Language(src))
				buf.WriteString("\n")
				buf.Write(node.Text(src))
				buf.WriteString("```")
				c.state = needNewline
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

	res.Content = strings.TrimSuffix(buf.String(), "\n")

	return res
}
