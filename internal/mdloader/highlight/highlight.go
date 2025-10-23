package highlight

import (
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// A HighlightAST struct represents a strikethrough of GFM text.
type HighlightAST struct {
	gast.BaseInline
}

// Dump implements Node.Dump.
func (n *HighlightAST) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// KindHighlightAST is a NodeKind of the HighlightAST node.
var KindHighlight = gast.NewNodeKind("Highlight")

// Kind implements Node.Kind.
func (n *HighlightAST) Kind() gast.NodeKind {
	return KindHighlight
}

// NewHighlightAST returns a new HighlightAST node.
func NewHighlightAST() *HighlightAST {
	return &HighlightAST{}
}

type highligthDelimiterProcessor struct {
}

func (p *highligthDelimiterProcessor) IsDelimiter(b byte) bool {
	return b == '='
}

func (p *highligthDelimiterProcessor) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}

func (p *highligthDelimiterProcessor) OnMatch(consumes int) gast.Node {
	return NewHighlightAST()
}

var defaultHighlightDelimiterProcessor = &highligthDelimiterProcessor{}

type highligthParser struct {
}

var defaultHighlightParser = &highligthParser{}

// NewHighlightParser return a new InlineParser that parses
// highligth expressions.
func NewHighlightParser() parser.InlineParser {
	return defaultHighlightParser
}

func (s *highligthParser) Trigger() []byte {
	return []byte{'='}
}

func (s *highligthParser) Parse(parent gast.Node, block text.Reader, pc parser.Context) gast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := parser.ScanDelimiter(line, before, 1, defaultHighlightDelimiterProcessor)
	if node == nil || node.OriginalLength > 2 || before == '~' {
		return nil
	}

	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

func (s *highligthParser) CloseBlock(parent gast.Node, pc parser.Context) {
	// nothing to do
}

// HighlightHTMLRenderer is a renderer.NodeRenderer implementation that
// renders Highlight nodes.
type HighlightHTMLRenderer struct {
	html.Config
}

// NewHighlightHTMLRenderer returns a new HighlightHTMLRenderer.
func NewHighlightHTMLRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &HighlightHTMLRenderer{
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

// RegisterFuncs implements renderer.NodeRenderer.RegisterFuncs.
func (r *HighlightHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindHighlight, r.renderHighlight)
}

// HighlightAttributeFilter defines attribute names which dd elements can have.
var HighlightAttributeFilter = html.GlobalAttributeFilter

func (r *HighlightHTMLRenderer) renderHighlight(
	w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		if n.Attributes() != nil {
			_, _ = w.WriteString("<mark")
			html.RenderAttributes(w, n, HighlightAttributeFilter)
			_ = w.WriteByte('>')
		} else {
			_, _ = w.WriteString("<mark>")
		}
	} else {
		_, _ = w.WriteString("</mark>")
	}
	return gast.WalkContinue, nil
}

type highligth struct {
}

// Highlight is an extension that allow you to use highligth expression like '~~text~~' .
var Highlight = &highligth{}

func (e *highligth) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(NewHighlightParser(), 500),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewHighlightHTMLRenderer(), 500),
	))
}
