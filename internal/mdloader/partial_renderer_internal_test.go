package mdloader

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// failingEmphasis writes part of its output and then fails, like a renderer
// that errors mid-node.
type failingEmphasis struct{}

func (failingEmphasis) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindEmphasis, func(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			_, _ = w.WriteString("<em>half")
			return ast.WalkStop, errors.New("boom")
		}
		return ast.WalkContinue, nil
	})
}

func TestPartialRenderDropsFailedNodeWhole(t *testing.T) {
	md := goldmark.New(goldmark.WithRendererOptions(
		renderer.WithNodeRenderers(util.Prioritized(failingEmphasis{}, 100)),
	))
	// goldmark buffers output in 4 KB chunks and drops the unflushed tail on
	// error; a node larger than that leaks its first chunk.
	long := strings.Repeat("word ", 1000)
	content := []byte("before " + long + "*bad* after\n\nnext\n\n## Title\n\nx " + long + "*bad* y\n\nkept")
	pr := &PartialRenderer{md: md}
	pr.SetContent(md.Parser().Parse(text.NewReader(content)), content)

	intro := pr.Introduce().ContentHTML
	require.NotContains(t, intro, "half")
	require.NotContains(t, intro, "before")
	require.NotContains(t, intro, "word")
	require.Contains(t, intro, "next")

	section := pr.Sections(2)[0].ContentHTML
	require.NotContains(t, section, "half")
	require.NotContains(t, section, "word")
	require.Contains(t, section, "kept")
}
