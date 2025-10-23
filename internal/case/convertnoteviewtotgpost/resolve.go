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

type transformer struct {
	warnings []string
	content  string

	insideBlockquote bool
	startOfLine      bool

	newLine bool
}

func (res *transformer) process(nv *model.NoteView) {
	src := nv.Content
	res.startOfLine = true

	var buf bytes.Buffer
	nv.Ast().Dump(src, 2)

	_ = ast.Walk(nv.Ast(), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		_, ok := n.(*ast.Text)
		if !ok && res.newLine {
			buf.WriteString("\n")
			res.newLine = false
			res.startOfLine = true
		}

		emphasis, ok := n.(*ast.Emphasis)
		if ok {
			if emphasis.Level == 1 {
				buf.WriteString("_")
			} else {
				buf.WriteString("*")
			}

			return ast.WalkContinue, nil
		}

		_, ok = n.(*extast.Strikethrough)
		if ok {
			buf.WriteString("~")
			return ast.WalkContinue, nil
		}

		_, ok = n.(*ast.CodeSpan)
		if ok {
			buf.WriteString("`")
			return ast.WalkContinue, nil
		}

		// ==highlight== to ||spoiler||
		_, ok = n.(*highlight.HighlightAST)
		if ok {
			buf.WriteString("||")
			return ast.WalkContinue, nil
		}

		_, ok = n.(*ast.Blockquote)
		if ok {
			res.insideBlockquote = entering
			if !entering {
				res.newLine = true
				res.startOfLine = true
			}
			return ast.WalkContinue, nil
		}

		link, ok := n.(*ast.Link)
		if ok {
			if entering {
				buf.WriteString("[")
			} else {
				buf.WriteString("](")
				buf.Write(link.Destination)
				buf.WriteString(")")
			}

			return ast.WalkContinue, nil
		}

		image, ok := n.(*enclavecore.Enclave)
		if ok {
			dest := string(image.Destination)
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

			return ast.WalkContinue, nil
		}

		if !entering {
			_, ok := n.(*ast.Paragraph)
			if ok {
				res.newLine = true
				res.startOfLine = true
			}

			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Document:
			// ok

		case *ast.Paragraph:
			// ok

		case *ast.Text:
			// это для ссылки плохо срабатывает
			if res.newLine {
				buf.WriteString("\n")
				res.newLine = false
				res.startOfLine = true
			}

			if res.insideBlockquote && res.startOfLine && !node.SoftLineBreak() {
				buf.WriteString(">")
				res.startOfLine = false
			}

			buf.Write(node.Segment.Value(src))
			if node.SoftLineBreak() {
				res.newLine = true
			} else {
				res.startOfLine = false
			}

		case *ast.RawHTML:
			tag := string(node.Segments.Value(src))
			if tag == "<u>" || tag == "</u>" {
				buf.WriteString("__")
				return ast.WalkContinue, nil
			}

			msg := fmt.Sprintf("raw html tag is not supported: %s", tag)
			res.warnings = append(res.warnings, msg)

		case *ast.FencedCodeBlock:
			if res.newLine {
				buf.WriteString("\n")
				res.newLine = false
				res.startOfLine = true
			}
			buf.WriteString("```")
			buf.Write(node.Language(src))
			buf.WriteString("\n")
			buf.Write(node.Text(src))
			buf.WriteString("```")
			res.newLine = true
			res.startOfLine = true

		default:
			msg := fmt.Sprintf("unexpected markdown node: %s", n.Kind())
			res.warnings = append(res.warnings, msg)
		}

		return ast.WalkContinue, nil
	})

	res.content = strings.TrimSuffix(buf.String(), "\n")
}
