package defaulttemplate

import (
	"trip2g/internal/model"
	"trip2g/internal/templateviews"
	"trip2g/internal/yamlutil"
)

// NavLink is a prev/next link or a breadcrumb; a crumb with an empty Href is plain text.
type NavLink struct {
	Label string
	Href  string
}

// PageNav is the prev/next pager and the breadcrumbs of a page.
type PageNav struct {
	Prev        *NavLink
	Next        *NavLink
	Breadcrumbs []NavLink
}

// PageNav is derived from the first sidebar content note linking to the page,
// left sidebar first. Frontmatter prev, next and breadcrumbs override it.
func (ctx *Ctx) PageNav() PageNav {
	if ctx.Note == nil || ctx.Notes == nil {
		return PageNav{}
	}

	nav := ctx.sidebarPageNav()

	m := ctx.Note.M()
	if m.Has("prev") {
		nav.Prev = ctx.noteNavLink(m.Get("prev"))
	}
	if m.Has("next") {
		nav.Next = ctx.noteNavLink(m.Get("next"))
	}
	if m.Has("breadcrumbs") {
		nav.Breadcrumbs = ctx.breadcrumbs(m.Get("breadcrumbs"))
	}

	return nav
}

func (ctx *Ctx) sidebarPageNav() PageNav {
	for _, position := range []string{"left", "right"} {
		for _, w := range ctx.SidebarWidgets(position) {
			if w.Kind != WidgetContent {
				continue
			}
			sidebar := ctx.resolveWidgetNote(w.Value)
			if sidebar == nil {
				continue
			}
			links := templateviews.NoteViews(ctx.Notes).NavLinks(templateviews.NoteView(sidebar))
			if nav, found := findPageNav(links, ctx.Note.PathID()); found {
				return nav
			}
		}
	}
	return PageNav{}
}

// findPageNav takes the neighbours of the first link to the page and the heading above it.
func findPageNav(links []model.NoteNavLink, pathID int64) (PageNav, bool) {
	for i, link := range links {
		if link.Note.PathID != pathID {
			continue
		}

		var nav PageNav
		if i > 0 {
			nav.Prev = sidebarNavLink(links[i-1])
		}
		if i+1 < len(links) {
			nav.Next = sidebarNavLink(links[i+1])
		}
		if link.Heading != "" {
			crumb := NavLink{Label: link.Heading}
			if link.HeadingNote != nil {
				crumb.Href = link.HeadingNote.PermalinkEncoded()
			}
			nav.Breadcrumbs = []NavLink{crumb}
		}
		return nav, true
	}
	return PageNav{}, false
}

// sidebarNavLink labels a link with its sidebar text, falling back to the note title.
func sidebarNavLink(link model.NoteNavLink) *NavLink {
	label := link.Label
	if label == "" {
		label = link.Note.Title
	}
	return &NavLink{Label: label, Href: link.Note.PermalinkEncoded()}
}

// noteNavLink resolves a "[[Note]]" or path frontmatter value; false or a miss gives nil.
func (ctx *Ctx) noteNavLink(raw interface{}) *NavLink {
	ref := parseContentRef(raw)
	if ref.Kind != ContentRefWikiLink && ref.Kind != ContentRefFile {
		return nil
	}
	note := ctx.resolveWidgetNote(ref.Value)
	if note == nil {
		return nil
	}
	return &NavLink{Label: note.Title(), Href: note.PermalinkEncoded()}
}

// breadcrumbs parses a list of "[[Note]]", paths and {label, href} maps.
func (ctx *Ctx) breadcrumbs(raw interface{}) []NavLink {
	items, _ := raw.([]interface{})

	var crumbs []NavLink
	for _, item := range items {
		if m, ok := yamlutil.Normalize(item).(map[string]interface{}); ok {
			label, _ := m["label"].(string)
			href, _ := m["href"].(string)
			crumbs = append(crumbs, NavLink{Label: label, Href: href})
		} else if link := ctx.noteNavLink(item); link != nil {
			crumbs = append(crumbs, *link)
		}
	}
	return crumbs
}

// resolveWidgetNote resolves a sidebar content widget value: by wikilink first, then by file path.
func (ctx *Ctx) resolveWidgetNote(value string) *templateviews.Note {
	if note := ctx.resolveNoteRef(ContentRef{Kind: ContentRefWikiLink, Value: value}); note != nil {
		return note
	}
	return ctx.resolveNoteRef(ContentRef{Kind: ContentRefFile, Value: value})
}
