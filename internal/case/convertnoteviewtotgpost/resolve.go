package convertnoteviewtotgpost

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"trip2g/internal/mdloader/highlight"
	"trip2g/internal/model"

	enclavecore "github.com/quailyquaily/goldmark-enclave/core"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

type Env interface {
}

func Resolve(ctx context.Context, env Env, nv *model.NoteView) (*model.TelegramPost, error) {
	tr := &transformer{}
	tr.process(nv)

	return &model.TelegramPost{
		Content:  tr.content,
		Warnings: tr.warnings,
	}, nil
}

type lineState int

const (
	midLine lineState = iota
	needNewline
	lineStart
)

type transformer struct {
	warnings []string
	content  string

	insideBlockquote bool
	state            lineState
}

func (res *transformer) process(nv *model.NoteView) {
	src := nv.Content
	res.state = lineStart

	var buf bytes.Buffer

	_ = ast.Walk(nv.Ast(), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		// Handle newlines for non-text nodes
		if _, ok := n.(*ast.Text); !ok && res.state == needNewline {
			buf.WriteString("\n")
			res.state = lineStart
		}

		switch node := n.(type) {
		case *ast.Document:
			// Nothing to do

		case *ast.Paragraph:
			if !entering {
				res.state = needNewline
			}

		case *ast.Text:
			if entering {
				if res.state == needNewline {
					buf.WriteString("\n")
					res.state = lineStart
				}

				if res.insideBlockquote && res.state == lineStart && !node.SoftLineBreak() {
					buf.WriteString(">")
					res.state = midLine
				}

				buf.Write(node.Segment.Value(src))
				if node.SoftLineBreak() {
					res.state = needNewline
				} else {
					res.state = midLine
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
			res.insideBlockquote = entering
			if !entering {
				res.state = needNewline
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
				res.warnings = append(res.warnings, msg)
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
					res.warnings = append(res.warnings, msg)
				}
			}

		case *ast.FencedCodeBlock:
			if entering {
				if res.state == needNewline {
					buf.WriteString("\n")
					res.state = lineStart
				}
				buf.WriteString("```")
				buf.Write(node.Language(src))
				buf.WriteString("\n")
				buf.Write(node.Text(src))
				buf.WriteString("```")
				res.state = needNewline
			}

		default:
			if entering {
				msg := fmt.Sprintf("unexpected markdown node: %s", n.Kind())
				res.warnings = append(res.warnings, msg)
			}
		}

		return ast.WalkContinue, nil
	})

	res.content = strings.TrimSuffix(buf.String(), "\n")
}
