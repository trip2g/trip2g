package model

import (
	"net/url"
	"path"
	"strings"

	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/goldmark/wikilink"
)

// NoteNavLink is a link to a note, with the nearest heading above it.
type NoteNavLink struct {
	Note        *NoteView
	Heading     string    // "" when no heading precedes the link
	HeadingNote *NoteView // the note the heading itself links to, if any
}

// NavLinks returns source's wikilinks and markdown links to notes in document
// order. Embeds, links to headings and unresolved links are skipped; a link inside a heading
// is not listed, it becomes HeadingNote of the links below.
func (nvs *NoteViews) NavLinks(source *NoteView) []NoteNavLink {
	if source == nil || source.ast == nil {
		return nil
	}

	var links []NoteNavLink
	var current NoteNavLink

	//nolint:errcheck,gosec // the callback never errors
	ast.Walk(source.ast, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if heading, ok := node.(*ast.Heading); ok {
			current = NoteNavLink{Heading: extractHeadingText(source.Content, heading)}
			for c := heading.FirstChild(); c != nil && current.HeadingNote == nil; c = c.NextSibling() {
				current.HeadingNote = nvs.navTarget(source, c)
			}
			return ast.WalkSkipChildren, nil
		}

		if note := nvs.navTarget(source, node); note != nil {
			link := current
			link.Note = note
			links = append(links, link)
		}

		return ast.WalkContinue, nil
	})

	return links
}

func (nvs *NoteViews) navTarget(source *NoteView, node ast.Node) *NoteView {
	switch link := node.(type) {
	case *wikilink.Node:
		if link.Embed || len(link.Fragment) > 0 {
			return nil
		}
		return nvs.ResolveWikilinkTarget(source, strings.TrimSuffix(string(link.Target), `\`))
	case *ast.Link:
		return nvs.markdownLinkTarget(source, string(link.Destination))
	}
	return nil
}

// markdownLinkTarget resolves a local link to a note: "/x" by permalink,
// "x.md" or "x" from source's folder first, then like a wikilink.
func (nvs *NoteViews) markdownLinkTarget(source *NoteView, dest string) *NoteView {
	u, err := url.Parse(dest)
	if err != nil || u.Scheme != "" || u.Host != "" || u.Fragment != "" || u.Path == "" {
		return nil
	}
	if strings.HasPrefix(u.Path, "/") {
		return nvs.Map[u.Path]
	}
	if ext := path.Ext(u.Path); ext != "" && ext != ".md" {
		return nil
	}

	target := strings.TrimSuffix(u.Path, ".md")
	if note := nvs.ResolveWikilinkTarget(source, "./"+target); note != nil {
		return note
	}
	return nvs.ResolveWikilinkTarget(source, target)
}
