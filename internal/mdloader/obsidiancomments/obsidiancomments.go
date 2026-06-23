// Package obsidiancomments strips Obsidian-style %% comments %% from rendered output.
//
// Inline comments (%%text%%) are removed while surrounding text is preserved.
// Block comments (%% on its own line, content, %% on its own line) produce no output.
// Content inside fenced code blocks and inline code spans is left untouched.
package obsidiancomments

import (
	"bytes"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// ObsidianInlineComment is an inline AST node for %%...%%.
type ObsidianInlineComment struct {
	gast.BaseInline
}

// Dump implements gast.Node.Dump.
func (n *ObsidianInlineComment) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// KindObsidianInlineComment is the NodeKind for ObsidianInlineComment.
var KindObsidianInlineComment = gast.NewNodeKind("ObsidianInlineComment")

// Kind implements gast.Node.Kind.
func (n *ObsidianInlineComment) Kind() gast.NodeKind {
	return KindObsidianInlineComment
}

// ObsidianBlockComment is a block-level AST node for %%\n...\n%%.
type ObsidianBlockComment struct {
	gast.BaseBlock
}

// Dump implements gast.Node.Dump.
func (n *ObsidianBlockComment) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// KindObsidianBlockComment is the NodeKind for ObsidianBlockComment.
var KindObsidianBlockComment = gast.NewNodeKind("ObsidianBlockComment")

// Kind implements gast.Node.Kind.
func (n *ObsidianBlockComment) Kind() gast.NodeKind {
	return KindObsidianBlockComment
}

type inlineCommentParser struct{}

var defaultInlineCommentParser = &inlineCommentParser{}

// Trigger returns the trigger byte for the inline parser.
func (p *inlineCommentParser) Trigger() []byte {
	return []byte{'%'}
}

// Parse tries to parse %%...%% at the current position.
// Returns nil if not a valid comment (e.g. lone %%).
func (p *inlineCommentParser) Parse(parent gast.Node, block text.Reader, pc parser.Context) gast.Node {
	line, _ := block.PeekLine()

	// Need at least %%%% (4 bytes: open %%, close %%)
	if len(line) < 4 || line[0] != '%' || line[1] != '%' {
		return nil
	}

	// Find closing %% within the same line (no multi-line inline comments)
	rest := line[2:]
	idx := bytes.Index(rest, []byte("%%"))
	if idx < 0 {
		return nil
	}

	// Advance past the entire %%...%% span
	advance := 2 + idx + 2
	block.Advance(advance)

	return &ObsidianInlineComment{}
}

func (p *inlineCommentParser) CloseBlock(parent gast.Node, pc parser.Context) {
	// nothing to do
}

type blockCommentParser struct{}

var defaultBlockCommentParser = &blockCommentParser{}

func (p *blockCommentParser) Trigger() []byte {
	return []byte{'%'}
}

// Open checks whether the current line is a standalone %%.
func (p *blockCommentParser) Open(parent gast.Node, reader text.Reader, pc parser.Context) (gast.Node, parser.State) {
	line, _ := reader.PeekLine()
	trimmed := bytes.TrimRight(line, "\r\n")
	if string(trimmed) != "%%" {
		return nil, parser.NoChildren
	}
	// Consume up to EOL; the parser loop advances past the newline.
	reader.AdvanceToEOL()
	return &ObsidianBlockComment{}, parser.NoChildren
}

// Continue consumes lines until a closing standalone %% or EOF.
func (p *blockCommentParser) Continue(node gast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, _ := reader.PeekLine()
	trimmed := bytes.TrimRight(line, "\r\n")
	if string(trimmed) == "%%" {
		// Consume closing %% line and stop.
		reader.AdvanceToEOL()
		return parser.Close
	}
	// Consume the content line.
	reader.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}

// Close is called when the block closes.
func (p *blockCommentParser) Close(node gast.Node, reader text.Reader, pc parser.Context) {}

// CanInterruptParagraph returns true so a block comment can start mid-document.
func (p *blockCommentParser) CanInterruptParagraph() bool { return true }

// CanAcceptIndentedLine returns false (block comment must start at column 0).
func (p *blockCommentParser) CanAcceptIndentedLine() bool { return false }

// IsRaw returns true to prevent child parsing.
func (p *blockCommentParser) IsRaw() bool { return true }

type commentRenderer struct{}

// RegisterFuncs registers render functions that produce no output.
func (r *commentRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindObsidianInlineComment, r.renderNothing)
	reg.Register(KindObsidianBlockComment, r.renderNothing)
}

func (r *commentRenderer) renderNothing(
	w util.BufWriter, source []byte, n gast.Node, entering bool,
) (gast.WalkStatus, error) {
	return gast.WalkSkipChildren, nil
}

type obsidianComments struct{}

// ObsidianComments is the goldmark extension that strips Obsidian %% comments.
var ObsidianComments = &obsidianComments{}

func (e *obsidianComments) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithInlineParsers(
			util.Prioritized(defaultInlineCommentParser, 300),
		),
		parser.WithBlockParsers(
			util.Prioritized(defaultBlockCommentParser, 400),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(&commentRenderer{}, 300),
		),
	)
}
