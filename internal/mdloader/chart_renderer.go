package mdloader

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"

	"trip2g/internal/model"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// chartRenderer renders ```datachart fenced blocks as chart containers and falls
// back to goldmark's default rendering for every other fenced code block.
type chartRenderer struct {
	resolver *myLinkResolver
}

func (r *chartRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.render)
}

func (r *chartRenderer) render(w util.BufWriter, src []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n, ok := node.(*ast.FencedCodeBlock)
	if !ok {
		return ast.WalkContinue, nil
	}

	lang := ""
	if n.Info != nil {
		lang = string(n.Info.Segment.Value(src))
	}

	if lang == model.ChartFenceLang {
		if entering {
			r.renderChart(w, src, n)
		}
		return ast.WalkSkipChildren, nil
	}

	// Default goldmark fenced-code rendering (replicated so non-datachart blocks
	// are unchanged — see goldmark renderer/html/html.go renderFencedCodeBlock).
	if entering {
		_, _ = w.WriteString("<pre><code")
		if language := n.Language(src); language != nil {
			_, _ = w.WriteString(` class="language-`)
			gmhtml.DefaultWriter.Write(w, language)
			_, _ = w.WriteString(`"`)
		}
		_ = w.WriteByte('>')
		lines := n.Lines()
		for i := range lines.Len() {
			line := lines.At(i)
			gmhtml.DefaultWriter.RawWrite(w, line.Value(src))
		}
	} else {
		_, _ = w.WriteString("</code></pre>\n")
	}
	return ast.WalkContinue, nil
}

// renderChart parses the block and emits a chart container plus a sibling JSON
// script the client widget reads. Resolution depends on data.source:
//   - inline:      rows are baked into the script (data present)
//   - frontmatter: the frontmatter [[link]] is resolved to a presigned URL and
//     emitted as data-src; the browser fetches it (data null)
//   - url/internal: backend-fetched + cached (not yet wired) → data null (loader)
func (r *chartRenderer) renderChart(w util.BufWriter, src []byte, n *ast.FencedCodeBlock) {
	var buf bytes.Buffer
	lines := n.Lines()
	for i := range lines.Len() {
		line := lines.At(i)
		_, _ = buf.Write(line.Value(src))
	}

	var chart model.NoteViewChart
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &chart); err != nil {
		// extractCharts already warned about this block; render nothing.
		return
	}

	var dataSrc, dataFormat string
	var rows json.RawMessage

	switch chart.Data.Source {
	case model.ChartSourceInline:
		rows = chart.Data.Rows
	case model.ChartSourceFrontmatter:
		dataSrc, dataFormat = r.resolveFrontmatterSrc(chart.Data.Ref)
	case model.ChartSourceURL, model.ChartSourceInternal:
		// backend job not wired yet → no data; widget shows a loader.
	}

	payload, err := json.Marshal(struct {
		Config json.RawMessage `json:"config"`
		Data   json.RawMessage `json:"data,omitempty"`
	}{Config: chart.Config, Data: rows})
	if err != nil {
		return
	}

	_, _ = w.WriteString(`<div class="chart"`)
	if dataSrc != "" {
		_, _ = w.WriteString(` data-src="`)
		_, _ = w.Write(util.EscapeHTML([]byte(dataSrc)))
		_, _ = w.WriteString(`"`)
	}
	if dataFormat != "" {
		_, _ = w.WriteString(` data-format="` + dataFormat + `"`)
	}
	_, _ = w.WriteString(`></div>`)
	_, _ = w.WriteString(`<script class="chart__data" type="application/json">`)
	_, _ = w.Write(payload)
	_, _ = w.WriteString(`</script>`)
}

// resolveFrontmatterSrc reads the named frontmatter key (which holds a vault
// [[link]]), resolves the link target to its presigned asset URL, and reports
// the data format from the file extension. Empty url → unresolved (no asset).
func (r *chartRenderer) resolveFrontmatterSrc(ref string) (string, string) {
	if ref == "" || r.resolver == nil || r.resolver.currentPage == nil {
		return "", ""
	}
	page := r.resolver.currentPage

	target := wikilinkTargetFromMeta(page.RawMeta[ref])
	if target == "" {
		return "", ""
	}

	if ar, ok := page.AssetReplaces[target]; ok && ar != nil {
		return ar.URL, formatFromExt(target)
	}
	// Frontmatter links may be resolved into ResolvedLinks during link extraction.
	if resolved, ok := page.ResolvedLinks[target]; ok {
		if ar, ok2 := page.AssetReplaces[resolved]; ok2 && ar != nil {
			return ar.URL, formatFromExt(resolved)
		}
	}
	return "", ""
}

// wikilinkTargetFromMeta extracts the `[[target]]` (or first of a list) from a
// frontmatter value, stripping the wrapping and any `|alias`.
func wikilinkTargetFromMeta(v any) string {
	switch val := v.(type) {
	case string:
		return stripWikilink(val)
	case []any:
		if len(val) > 0 {
			if s, ok := val[0].(string); ok {
				return stripWikilink(s)
			}
		}
	}
	return ""
}

func stripWikilink(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[[")
	s = strings.TrimSuffix(s, "]]")
	if i := strings.IndexByte(s, '|'); i != -1 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func formatFromExt(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".csv":
		return "csv"
	case ".json":
		return "json"
	default:
		return ""
	}
}
