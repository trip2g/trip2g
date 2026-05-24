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

// frontmatterBlockRe captures the YAML inner block so we can scan for keys
// like `image:` without pulling in a full YAML parser.
var frontmatterBlockRe = regexp.MustCompile(`(?s)\A---\r?\n(.*?)\r?\n---\r?\n*`)

// frontmatterImageRe pulls the value of an `image:` key from a YAML block.
// Accepts quoted or unquoted values, single-line only (good enough for demos).
var frontmatterImageRe = regexp.MustCompile(`(?m)^image:\s*"?([^"\r\n]+?)"?\s*$`)

// embedRe matches an Obsidian wikilink embed `![[target]]` with optional
// trailing newline so removing it doesn't leave double blank lines.
var embedRe = regexp.MustCompile(`!\[\[([^\]\n]+)\]\]\n?`)

// imageExtRe checks whether a wikilink target points at a media file.
var imageExtRe = regexp.MustCompile(`(?i)\.(png|jpg|jpeg|gif|webp|bmp|svg)$`)

func stripFrontmatter(s string) string {
	return frontmatterRe.ReplaceAllString(s, "")
}

// extractFirstImage returns the first image reference for a note and the
// body with frontmatter and the consumed image embed stripped. The
// frontmatter `image:` key wins over any `![[…]]` embed in the body.
// Non-image embeds (e.g., `![[other.md]]`) are left in place.
func extractFirstImage(content string) (image, body string) {
	if m := frontmatterBlockRe.FindStringSubmatch(content); len(m) > 1 {
		if im := frontmatterImageRe.FindStringSubmatch(m[1]); len(im) > 1 {
			image = strings.TrimSpace(im[1])
		}
	}

	body = strings.TrimSpace(stripFrontmatter(content))

	if image != "" {
		return image, body
	}

	loc := embedRe.FindAllStringSubmatchIndex(body, -1)
	for _, m := range loc {
		target := body[m[2]:m[3]]
		if !imageExtRe.MatchString(target) {
			continue
		}
		image = target
		body = strings.TrimSpace(body[:m[0]] + body[m[1]:])
		break
	}
	return image, body
}

// nodeBody returns the message body for a node. File nodes are loaded from
// disk relative to VaultRoot. `.base` files are placeholdered out since
// Bases aren't supported yet.
func (c *Canvas) nodeBody(n canvasNode) string {
	text, _ := c.nodeContent(n)
	return text
}

// nodeContent returns the message body plus an optional resolved local image
// path. File nodes that carry an `image:` frontmatter or a `![[…]]` image
// embed surface the first match here; the bot can then route the render
// through sendPhoto+caption instead of plain sendMessage.
func (c *Canvas) nodeContent(n canvasNode) (text, media string) {
	switch n.Type {
	case "text":
		return n.Text, ""
	case "file":
		if strings.HasSuffix(strings.ToLower(n.File), ".base") {
			return "Bases are not supported yet (file: " + n.File + ")", ""
		}
		raw, err := os.ReadFile(filepath.Join(c.VaultRoot, n.File))
		if err != nil {
			return fmt.Sprintf("(failed to load %s: %v)", n.File, err), ""
		}
		image, body := extractFirstImage(string(raw))
		body = strings.TrimSpace(body)
		if image != "" {
			return body, c.resolveImagePath(n.File, image)
		}
		return body, ""
	case "link":
		return "🌐 " + n.URL, ""
	case "group":
		return "(group)", ""
	}
	return "(unsupported node type: " + n.Type + ")", ""
}

// resolveImagePath turns an image reference from a note's frontmatter or
// `![[…]]` embed into an absolute filesystem path the bot can upload.
// Refs starting with "/" are resolved against the vault root; everything
// else relative to the note's directory (Obsidian default).
func (c *Canvas) resolveImagePath(noteFile, imageRef string) string {
	if strings.HasPrefix(imageRef, "/") {
		return filepath.Join(c.VaultRoot, strings.TrimPrefix(imageRef, "/"))
	}
	return filepath.Join(c.VaultRoot, filepath.Dir(noteFile), imageRef)
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
		return applyInlineMarkdown(htmlEscape(line))
	}
	title := strings.TrimSpace(line[level+1:])
	if title == "" {
		return htmlEscape(line)
	}
	switch {
	case level == 1:
		return "<b>" + applyInlineMarkdown(htmlEscape(strings.ToUpper(title))) + "</b>"
	case level == 2:
		return "<b>" + applyInlineMarkdown(htmlEscape(title)) + "</b>"
	default:
		return "<b><i>" + applyInlineMarkdown(htmlEscape(title)) + "</i></b>"
	}
}

// inlineMarkdown patterns are applied to text that has already been
// HTML-escaped. Order matters: inline code first (so its contents are not
// re-processed as bold/italic/link), bold before italic (since `**` would
// otherwise be eaten as two italic boundaries), links last.
var (
	inlineCodeRe = regexp.MustCompile("`([^`\n]+)`")
	boldRe       = regexp.MustCompile(`\*\*([^*\n]+?)\*\*`)
	italicRe     = regexp.MustCompile(`\*([^*\n]+?)\*`)
	mdLinkRe     = regexp.MustCompile(`\[([^\]\n]+)\]\(([^)\s\n]+)\)`)
)

func applyInlineMarkdown(escaped string) string {
	escaped = inlineCodeRe.ReplaceAllString(escaped, "<code>$1</code>")
	escaped = boldRe.ReplaceAllString(escaped, "<b>$1</b>")
	escaped = italicRe.ReplaceAllString(escaped, "<i>$1</i>")
	escaped = mdLinkRe.ReplaceAllString(escaped, `<a href="$2">$1</a>`)
	return escaped
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
