package model

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/yuin/goldmark/ast"
)

// ChartFenceLang is the info string of a fenced code block that defines a chart.
// Chosen to avoid collision with the Obsidian Charts (`chart`) and ECharts
// (`echarts`) community plugins.
const ChartFenceLang = "datachart"

// Chart data source kinds (the `data.source` discriminator).
const (
	ChartSourceFrontmatter = "frontmatter" // rows from a vault file referenced by a frontmatter [[link]] (client-fetched)
	ChartSourceURL         = "url"         // rows from an external HTTP-JSON endpoint (backend fetch + cache)
	ChartSourceInternal    = "internal"    // rows from a SELECT over trip2g's read replica (backend)
	ChartSourceInline      = "inline"      // rows bundled in the block
)

// ChartDataSource is the typed `data` object of a datachart block: it declares
// where the chart's rows come from. Exactly one source kind is used per chart.
type ChartDataSource struct {
	Source string          `json:"source"`         // frontmatter | url | internal | inline
	Ref    string          `json:"ref,omitempty"`  // frontmatter: key holding the [[link]]
	URL    string          `json:"url,omitempty"`  // url: HTTP-JSON endpoint
	Body   string          `json:"body,omitempty"` // url: optional request body (POSTed when present)
	SQL    string          `json:"sql,omitempty"`  // internal: SELECT against the read replica
	Rows   json.RawMessage `json:"rows,omitempty"` // inline: bundled rows
}

// CacheHash returns a stable backend cache key for sources fetched and cached
// server-side (url, internal). Client-fetched / bundled sources (frontmatter,
// inline) return "" — they are never cached server-side.
func (d ChartDataSource) CacheHash() string {
	var key string
	switch d.Source {
	case ChartSourceURL:
		key = "url\x00" + d.URL + "\x00" + d.Body
	case ChartSourceInternal:
		key = "internal\x00" + d.SQL
	default:
		return ""
	}
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// NoteViewChart is a dashboard chart parsed from a ```datachart fenced code block.
type NoteViewChart struct {
	Index  int             `json:"-"`               // position among chart blocks in the note (0-based)
	Hash   string          `json:"-"`               // backend cache key for url/internal; "" otherwise
	Title  string          `json:"title,omitempty"` // optional, for display
	Data   ChartDataSource `json:"data"`            // where the rows come from
	Config json.RawMessage `json:"config"`          // ECharts option object
}

// extractCharts walks the AST collecting all ```datachart fenced blocks into
// n.Charts, in document order. Invalid blocks are skipped with a warning.
func (n *NoteView) extractCharts() {
	n.Charts = nil

	if n.ast == nil {
		return
	}

	idx := 0
	_ = ast.Walk(n.ast, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		fcb, ok := node.(*ast.FencedCodeBlock)
		if !ok {
			return ast.WalkContinue, nil
		}

		if fcb.Info == nil {
			return ast.WalkContinue, nil
		}

		lang := string(fcb.Info.Segment.Value(n.Content))
		if lang != ChartFenceLang {
			return ast.WalkContinue, nil
		}

		var buf bytes.Buffer
		lines := fcb.Lines()
		for i := range lines.Len() {
			line := lines.At(i)
			buf.Write(line.Value(n.Content))
		}
		body := bytes.TrimSpace(buf.Bytes())

		var chart NoteViewChart
		if err := json.Unmarshal(body, &chart); err != nil {
			n.AddWarning(NoteWarningWarning, "invalid chart block: %s", err.Error())
			return ast.WalkContinue, nil
		}

		chart.Index = idx
		chart.Hash = chart.Data.CacheHash()
		n.Charts = append(n.Charts, chart)
		idx++

		return ast.WalkContinue, nil
	})
}
