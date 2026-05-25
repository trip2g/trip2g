package handletgcanvasupdate

import (
	"regexp"
	"strings"
)

// renderBodyHTML converts a minimal subset of Markdown to Telegram HTML.
// Used for text-type canvas nodes (not file nodes which go through HTMLConverter).
// Fenced code blocks become <pre>. Headings become bold. Inline markdown applied.
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
		return applyInlineMarkdown(line)
	}
	title := strings.TrimSpace(line[level+1:])
	if title == "" {
		return htmlEscape(line)
	}
	switch level {
	case 1:
		return "<b>" + applyInlineMarkdown(strings.ToUpper(title)) + "</b>"
	case 2:
		return "<b>" + applyInlineMarkdown(title) + "</b>"
	default:
		return "<b><i>" + applyInlineMarkdown(title) + "</i></b>"
	}
}

var (
	inlineCodeRe = regexp.MustCompile("`([^`\n]+)`")
	boldRe       = regexp.MustCompile(`\*\*([^*\n]+?)\*\*`)
	italicRe     = regexp.MustCompile(`\*([^*\n]+?)\*`)
	mdLinkRe     = regexp.MustCompile(`\[([^\]\n]+)\]\(([^)\s\n]+)\)`)
)

// applyInlineMarkdown converts a minimal subset of Markdown to Telegram HTML.
// Takes RAW (unescaped) text and is responsible for both escaping non-link text
// and emitting properly-escaped `<a href="...">...</a>` for `[text](url)` links.
// Doing links first on raw text is important so HTML-escape doesn't double-encode
// `&` inside URLs (e.g. `?a=1&b=2`) before the link regex sees them.
func applyInlineMarkdown(raw string) string {
	var b strings.Builder
	last := 0
	for _, m := range mdLinkRe.FindAllStringSubmatchIndex(raw, -1) {
		// m layout: [matchStart, matchEnd, textStart, textEnd, urlStart, urlEnd]
		b.WriteString(applyInlineNoLinks(htmlEscape(raw[last:m[0]])))
		text := htmlEscape(raw[m[2]:m[3]])
		url := htmlEscape(raw[m[4]:m[5]])
		b.WriteString(`<a href="`)
		b.WriteString(url)
		b.WriteString(`">`)
		b.WriteString(text)
		b.WriteString(`</a>`)
		last = m[1]
	}
	b.WriteString(applyInlineNoLinks(htmlEscape(raw[last:])))
	return b.String()
}

// applyInlineNoLinks runs inline-code / bold / italic on text that is already
// HTML-escaped and known to contain no markdown link syntax.
func applyInlineNoLinks(escaped string) string {
	escaped = inlineCodeRe.ReplaceAllString(escaped, "<code>$1</code>")
	escaped = boldRe.ReplaceAllString(escaped, "<b>$1</b>")
	escaped = italicRe.ReplaceAllString(escaped, "<i>$1</i>")
	return escaped
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
