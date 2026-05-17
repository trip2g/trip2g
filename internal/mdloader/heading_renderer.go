package mdloader

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// headingRenderer wraps heading sections in <div data-header="plain text">.
//
// The div is opened when the heading is entered and closed when a heading of
// equal or smaller level appears (or the document ends). This produces nested
// sections: h3 content sits inside the h2 div, which sits inside the h1 div.
//
// Wrapping only happens during full document renders (when ast.KindDocument is
// visited). Partial renders of individual nodes (used by PartialRenderer) get
// plain <hN> tags to avoid shared-state corruption.
//
// Registered at priority 200 to override the default goldmark HTML renderer
// (priority 1000) for ast.KindHeading and ast.KindDocument.
type headingRenderer struct {
	stack      []int // levels of currently open section divs
	inDocument bool  // true only during a full document render
}

func (r *headingRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHeading, r.renderHeading)
	reg.Register(ast.KindDocument, r.renderDocument)
}

func (r *headingRenderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	heading, ok := node.(*ast.Heading)
	if !ok {
		return ast.WalkContinue, nil
	}

	level := heading.Level
	tag := fmt.Sprintf("h%d", level)

	if !r.inDocument {
		// Partial render (individual node) — plain heading, no section div.
		if entering {
			_, _ = fmt.Fprintf(w, "<%s>", tag)
		} else {
			_, _ = fmt.Fprintf(w, "</%s>\n", tag)
		}
		return ast.WalkContinue, nil
	}

	if entering {
		// Close all open sections with level >= current (same or shallower).
		for len(r.stack) > 0 && r.stack[len(r.stack)-1] >= level {
			_, _ = w.WriteString(`</div>`)
			r.stack = r.stack[:len(r.stack)-1]
		}
		text := headingPlainText(source, heading)
		idAttr := ""
		if rawID, hasID := heading.AttributeString("id"); hasID {
			if idBytes, isBytes := rawID.([]byte); isBytes {
				idAttr = ` id="` + string(idBytes) + `"`
			}
		}
		_, _ = fmt.Fprintf(w, `<div data-header="%s" data-level="%d"><%s%s>`, escapeAttr(text), level, tag, idAttr)
		r.stack = append(r.stack, level)
	} else {
		_, _ = fmt.Fprintf(w, `</%s>`, tag)
	}

	return ast.WalkContinue, nil
}

func headingPlainText(source []byte, heading *ast.Heading) string {
	var sb strings.Builder
	for child := heading.FirstChild(); child != nil; child = child.NextSibling() {
		writeNodeText(source, &sb, child)
	}
	return strings.TrimSpace(sb.String())
}

func escapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, `"`, "&#34;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// renderDocument resets the stack on document start and closes remaining
// open section divs when the document ends.
func (r *headingRenderer) renderDocument(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.stack = r.stack[:0]
		r.inDocument = true
	} else {
		for len(r.stack) > 0 {
			_, _ = w.WriteString(`</div>`)
			r.stack = r.stack[:len(r.stack)-1]
		}
		r.inDocument = false
	}
	return ast.WalkContinue, nil
}
