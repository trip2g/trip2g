package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type canvasNode struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	File  string `json:"file,omitempty"`
	URL   string `json:"url,omitempty"`
	Color string `json:"color,omitempty"`
}

type canvasEdge struct {
	ID       string `json:"id"`
	FromNode string `json:"fromNode"`
	ToNode   string `json:"toNode"`
	Label    string `json:"label,omitempty"`
}

type canvasFile struct {
	Nodes []canvasNode `json:"nodes"`
	Edges []canvasEdge `json:"edges"`
}

// Canvas wraps an Obsidian .canvas file as a navigable graph for the bot.
// VaultRoot is the directory under which file-node `file` paths are resolved.
type Canvas struct {
	Path      string
	VaultRoot string

	nodesByID map[string]canvasNode
	outgoing  map[string][]canvasEdge
}

func loadCanvas(path, vaultRoot string) (*Canvas, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read canvas: %w", err)
	}
	var cf canvasFile
	if err := json.Unmarshal(raw, &cf); err != nil {
		return nil, fmt.Errorf("parse canvas: %w", err)
	}
	c := &Canvas{
		Path:      path,
		VaultRoot: vaultRoot,
		nodesByID: make(map[string]canvasNode, len(cf.Nodes)),
		outgoing:  make(map[string][]canvasEdge),
	}
	for _, n := range cf.Nodes {
		c.nodesByID[n.ID] = n
	}
	for _, e := range cf.Edges {
		c.outgoing[e.FromNode] = append(c.outgoing[e.FromNode], e)
	}
	return c, nil
}

// entry resolves the first screen of the canvas.
// Convention: a text node whose trimmed text equals "START" (case-insensitive)
// is the entry marker. The START node itself is not rendered:
//   - 1 outgoing edge → auto-open the target
//   - N outgoing edges → caller renders a menu rooted at START
//   - 0 START nodes → return "" (caller decides — usually surface an error)
func (c *Canvas) entry() string {
	var startID string
	for _, n := range c.nodesByID {
		if n.Type == "text" && strings.EqualFold(strings.TrimSpace(n.Text), "START") {
			startID = n.ID
			break
		}
	}
	if startID == "" {
		return ""
	}
	out := c.outgoing[startID]
	if len(out) == 1 {
		return out[0].ToNode
	}
	return startID
}

var frontmatterRe = regexp.MustCompile(`(?s)\A---\r?\n.*?\r?\n---\r?\n*`)

func stripFrontmatter(s string) string {
	return frontmatterRe.ReplaceAllString(s, "")
}

// nodeBody returns the message body for a node. File nodes are loaded from
// disk relative to VaultRoot. `.base` files are placeholdered out since
// Bases aren't supported yet.
func (c *Canvas) nodeBody(n canvasNode) string {
	switch n.Type {
	case "text":
		return n.Text
	case "file":
		if strings.HasSuffix(strings.ToLower(n.File), ".base") {
			return "Bases are not supported yet (file: " + n.File + ")"
		}
		raw, err := os.ReadFile(filepath.Join(c.VaultRoot, n.File))
		if err != nil {
			return fmt.Sprintf("(failed to load %s: %v)", n.File, err)
		}
		return strings.TrimSpace(stripFrontmatter(string(raw)))
	case "link":
		return "🌐 " + n.URL
	case "group":
		return "(group)"
	}
	return "(unsupported node type: " + n.Type + ")"
}

func (c *Canvas) edgesFrom(nodeID string) []canvasEdge {
	return c.outgoing[nodeID]
}

func (c *Canvas) node(id string) (canvasNode, bool) {
	n, ok := c.nodesByID[id]
	return n, ok
}

// edgeLabel falls back to the target node's display name when an edge has no
// explicit label (canvas authors sometimes leave them blank).
func (c *Canvas) edgeLabel(e canvasEdge) string {
	if strings.TrimSpace(e.Label) != "" {
		return e.Label
	}
	if target, ok := c.nodesByID[e.ToNode]; ok {
		return defaultNodeLabel(target)
	}
	return "→ " + e.ToNode
}

// renderBodyHTML converts a minimal subset of Markdown to Telegram HTML.
// Fenced code blocks (```...```) become <pre> so ASCII tables render
// monospaced. Lines starting with `#` are heading lines:
//   - h1 → bold UPPERCASE (strongest emphasis)
//   - h2 → bold
//   - h3 and deeper (h4/h5/h6) → bold italic, all collapsed to one tier
// Everything else is HTML-escaped verbatim — no other Markdown is interpreted.
func renderBodyHTML(s string) string {
	parts := strings.Split(s, "```")
	var out strings.Builder
	for i, p := range parts {
		if i%2 == 1 {
			// Inside a code fence. Strip a leading language hint line if present.
			if nl := strings.IndexByte(p, '\n'); nl >= 0 {
				first := strings.TrimSpace(p[:nl])
				if first != "" && !strings.ContainsAny(first, " \t") {
					p = p[nl+1:]
				}
			}
			out.WriteString("<pre>")
			out.WriteString(htmlEscape(p))
			out.WriteString("</pre>")
			continue
		}
		lines := strings.Split(p, "\n")
		for j, line := range lines {
			if j > 0 {
				out.WriteString("\n")
			}
			out.WriteString(renderLineHTML(line))
		}
	}
	return out.String()
}

func renderLineHTML(line string) string {
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level == 0 || level >= len(line) || line[level] != ' ' {
		return htmlEscape(line)
	}
	title := strings.TrimSpace(line[level+1:])
	if title == "" {
		return htmlEscape(line)
	}
	switch {
	case level == 1:
		return "<b>" + htmlEscape(strings.ToUpper(title)) + "</b>"
	case level == 2:
		return "<b>" + htmlEscape(title) + "</b>"
	default:
		return "<b><i>" + htmlEscape(title) + "</i></b>"
	}
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func defaultNodeLabel(n canvasNode) string {
	switch n.Type {
	case "file":
		base := filepath.Base(n.File)
		return strings.TrimSuffix(base, filepath.Ext(base))
	case "text":
		for _, line := range strings.Split(n.Text, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if len(line) > 40 {
				line = line[:40] + "…"
			}
			return line
		}
		return "(text)"
	case "link":
		return n.URL
	}
	return n.ID
}
