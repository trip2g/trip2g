package defaulttemplate

import "strings"

// ContentRefKind identifies the type of content block.
type ContentRefKind int

const (
	ContentRefSelfContent ContentRefKind = iota
	ContentRefMagazine
	ContentRefWikiLink // [[Title]] - resolve via ctx.Notes.ByPermalink
	ContentRefFile     // "path/file.md" - resolve via ctx.Notes.ByPath
	ContentRefNone
	ContentRefSimilar
	ContentRefInLinks
	ContentRefOutLinks
	ContentRefTOC
)

// ContentRef represents a content block reference in frontmatter.
type ContentRef struct {
	Kind  ContentRefKind
	Value string
}

// WidgetKind identifies the type of sidebar widget.
type WidgetKind int

const (
	WidgetTOC WidgetKind = iota
	WidgetInLinks
	WidgetOutLinks
	WidgetContent // wikilink or file path
	WidgetSimilar
)

// WidgetRef represents a sidebar widget reference.
type WidgetRef struct {
	Kind  WidgetKind
	Value string
}

// parseContentRef parses a content string into a ContentRef.
// Recognized values: "selfcontent", "magazine", "toc", "inlinks"/"backlinks",
// "outlinks", "similar", "[[Title]]", "path.md", false.
// The widget keywords mirror parseWidgetRef so a content: list can render the
// same blocks (backlinks, similar, TOC, ...) inline under the note body.
func parseContentRef(raw interface{}) ContentRef {
	s, ok := raw.(string)
	if !ok {
		// Check for bool false
		if b, isBool := raw.(bool); isBool && !b {
			return ContentRef{Kind: ContentRefNone}
		}
		return ContentRef{Kind: ContentRefNone}
	}

	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)

	switch lower {
	case "selfcontent", "self_content", "self":
		return ContentRef{Kind: ContentRefSelfContent}
	case "magazine":
		return ContentRef{Kind: ContentRefMagazine}
	case "toc":
		return ContentRef{Kind: ContentRefTOC}
	case "inlinks", "backlinks":
		return ContentRef{Kind: ContentRefInLinks}
	case "outlinks":
		return ContentRef{Kind: ContentRefOutLinks}
	case "similar":
		return ContentRef{Kind: ContentRefSimilar}
	case "false", "none":
		return ContentRef{Kind: ContentRefNone}
	}

	// Check for wikilink [[Title]]
	if strings.HasPrefix(s, "[[") && strings.HasSuffix(s, "]]") {
		title := strings.TrimPrefix(s, "[[")
		title = strings.TrimSuffix(title, "]]")
		title = strings.TrimSpace(title)
		return ContentRef{Kind: ContentRefWikiLink, Value: title}
	}

	// Treat as file path
	return ContentRef{Kind: ContentRefFile, Value: s}
}

// parseWidgetRef parses a widget string into a WidgetRef.
// Recognized values: "TOC", "toc", "inlinks", "backlinks", "outlinks",
// or a wikilink/file path for content widgets.
func parseWidgetRef(raw interface{}) (WidgetRef, bool) {
	s, ok := raw.(string)
	if !ok {
		return WidgetRef{}, false
	}

	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)

	switch lower {
	case "toc":
		return WidgetRef{Kind: WidgetTOC}, true
	case "inlinks", "backlinks":
		return WidgetRef{Kind: WidgetInLinks}, true
	case "outlinks":
		return WidgetRef{Kind: WidgetOutLinks}, true
	case "similar":
		return WidgetRef{Kind: WidgetSimilar}, true
	}

	// Check for wikilink [[Title]]
	if strings.HasPrefix(s, "[[") && strings.HasSuffix(s, "]]") {
		title := strings.TrimPrefix(s, "[[")
		title = strings.TrimSuffix(title, "]]")
		title = strings.TrimSpace(title)
		return WidgetRef{Kind: WidgetContent, Value: title}, true
	}

	// Treat as file path for content widget
	if strings.HasSuffix(lower, ".md") || strings.Contains(s, "/") {
		return WidgetRef{Kind: WidgetContent, Value: s}, true
	}

	return WidgetRef{}, false
}

// parseGlobSectionWidgets parses content: from a glob section note into widget refs.
// "self"/"selfcontent" becomes WidgetContent pointing to the section note itself.
func parseGlobSectionWidgets(raw interface{}, notePath string) []WidgetRef {
	var items []interface{}
	switch v := raw.(type) {
	case []interface{}:
		items = v
	case string:
		items = []interface{}{v}
	default:
		return nil
	}
	var widgets []WidgetRef
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			continue
		}
		lower := strings.ToLower(strings.TrimSpace(s))
		if lower == "self" || lower == "selfcontent" || lower == "self_content" {
			widgets = append(widgets, WidgetRef{Kind: WidgetContent, Value: notePath})
			continue
		}
		if w, wOk := parseWidgetRef(item); wOk {
			widgets = append(widgets, w)
		}
	}
	return widgets
}
