package callout

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// icons maps a callout type to its display glyph. Aliases share an icon.
var icons = map[string]string{
	"note":      "📝",
	"abstract":  "📋",
	"summary":   "📋",
	"tldr":      "📋",
	"info":      "ℹ️",
	"todo":      "✅",
	"tip":       "💡",
	"hint":      "💡",
	"important": "💡",
	"success":   "✔️",
	"check":     "✔️",
	"done":      "✔️",
	"question":  "❓",
	"help":      "❓",
	"faq":       "❓",
	"warning":   "⚠️",
	"caution":   "⚠️",
	"attention": "⚠️",
	"failure":   "✖️",
	"fail":      "✖️",
	"missing":   "✖️",
	"danger":    "🔥",
	"error":     "🔥",
	"bug":       "🐛",
	"example":   "📖",
	"quote":     "💬",
	"cite":      "💬",
}

const defaultIcon = "📌"

// htmlRenderer renders Callout nodes to HTML.
type htmlRenderer struct{}

// newRenderer returns a NodeRenderer for callout nodes.
func newRenderer() renderer.NodeRenderer {
	return &htmlRenderer{}
}

// RegisterFuncs implements renderer.NodeRenderer.
func (r *htmlRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(Kind, r.render)
}

func (r *htmlRenderer) render(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n, ok := node.(*Node)
	if !ok {
		return ast.WalkContinue, nil
	}

	if entering {
		r.renderOpen(w, n)
	} else {
		r.renderClose(w, n)
	}

	return ast.WalkContinue, nil
}

func (r *htmlRenderer) renderOpen(w util.BufWriter, n *Node) {
	icon := icons[n.CalloutType]
	if icon == "" {
		icon = defaultIcon
	}

	title := n.Title
	if title == "" {
		title = capitalize(n.CalloutType)
	}

	class := "callout callout--" + n.CalloutType

	if n.Foldable {
		if n.Expanded {
			_, _ = w.WriteString(`<details open class="`)
		} else {
			_, _ = w.WriteString(`<details class="`)
		}
		_, _ = w.WriteString(class)
		_, _ = w.WriteString(`">` + "\n")
		_, _ = w.WriteString(`<summary class="callout__header">`)
	} else {
		_, _ = w.WriteString(`<div class="`)
		_, _ = w.WriteString(class)
		_, _ = w.WriteString(`">` + "\n")
		_, _ = w.WriteString(`<div class="callout__header">`)
	}

	_, _ = w.WriteString(`<span class="callout__icon">`)
	_, _ = w.WriteString(icon)
	_, _ = w.WriteString(`</span>`)
	_, _ = w.WriteString(`<span class="callout__title">`)
	_, _ = w.Write(util.EscapeHTML([]byte(title)))
	_, _ = w.WriteString(`</span>`)

	if n.Foldable {
		_, _ = w.WriteString(`</summary>` + "\n")
	} else {
		_, _ = w.WriteString(`</div>` + "\n")
	}

	_, _ = w.WriteString(`<div class="callout__body">` + "\n")
}

func (r *htmlRenderer) renderClose(w util.BufWriter, n *Node) {
	_, _ = w.WriteString(`</div>` + "\n")
	if n.Foldable {
		_, _ = w.WriteString(`</details>` + "\n")
	} else {
		_, _ = w.WriteString(`</div>` + "\n")
	}
}

// capitalize upper-cases the first rune of s, leaving the rest unchanged.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
