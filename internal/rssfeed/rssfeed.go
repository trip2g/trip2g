// Package rssfeed generates RSS 2.0 feeds from note AST.
package rssfeed

import (
	"bytes"
	"encoding/xml"
	"strings"
	"time"

	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/goldmark/wikilink"

	"trip2g/internal/image"
	"trip2g/internal/model"
)

// RSSFeed represents an RSS 2.0 feed.
type RSSFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel RSSChannel `xml:"channel"`
}

// RSSChannel represents an RSS channel.
type RSSChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []RSSItem `xml:"item"`
}

// RSSItem represents an RSS item.
type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description,omitempty"`
	PubDate     string `xml:"pubDate,omitempty"`
	GUID        string `xml:"guid"`
}

// link is an extracted link from the AST.
type link struct {
	Title string
	URL   string // absolute URL or resolved permalink
}

// Generate creates an RSS feed from a note.
// publicURL is the site's public URL (e.g., "https://example.com").
// notes is used to resolve internal links and get metadata.
func Generate(note *model.NoteView, publicURL string, notes *model.NoteViews) ([]byte, error) {
	links := extractLinks(note)

	feedTitle := note.Title
	if note.RSSTitle != "" {
		feedTitle = note.RSSTitle
	}

	feedDesc := ""
	if note.RSSDescription != "" {
		feedDesc = note.RSSDescription
	} else if note.Description != nil {
		feedDesc = *note.Description
	}

	items := make([]RSSItem, 0, len(links))
	for _, l := range links {
		item := buildItem(l, publicURL, notes)
		items = append(items, item)
	}

	feed := RSSFeed{
		Version: "2.0",
		Channel: RSSChannel{
			Title:       feedTitle,
			Link:        publicURL + note.Permalink,
			Description: feedDesc,
			Items:       items,
		},
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)

	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")

	err := enc.Encode(feed)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// extractLinks walks the note AST and extracts all links.
func extractLinks(note *model.NoteView) []link {
	if note.Ast() == nil {
		return nil
	}

	var links []link

	_ = ast.Walk(note.Ast(), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n.Kind() {
		case wikilink.Kind:
			wl, ok := n.(*wikilink.Node)
			if !ok {
				return ast.WalkContinue, nil
			}

			// Skip embedded images/videos.
			if wl.Embed && image.IsMediaExtension(string(wl.Target)) {
				return ast.WalkContinue, nil
			}

			target := string(wl.Target)
			title := target

			// Use display text if different from target.
			if wl.ChildCount() > 0 {
				title = textContent(note.Content, wl)
			}

			// Resolve via ResolvedLinks map.
			resolved, ok := note.ResolvedLinks[target]
			if ok {
				links = append(links, link{Title: title, URL: resolved})
			}

		case ast.KindLink:
			l, ok := n.(*ast.Link)
			if !ok {
				return ast.WalkContinue, nil
			}

			dest := string(l.Destination)
			if dest == "" {
				return ast.WalkContinue, nil
			}

			// Skip image links.
			if image.IsMediaExtension(dest) {
				return ast.WalkContinue, nil
			}

			title := textContent(note.Content, l)
			if title == "" {
				title = dest
			}

			links = append(links, link{Title: title, URL: dest})
		}

		return ast.WalkContinue, nil
	})

	return links
}

// buildItem creates an RSS item from a link.
func buildItem(l link, publicURL string, notes *model.NoteViews) RSSItem {
	itemURL := l.URL
	isInternal := !strings.HasPrefix(l.URL, "http://") &&
		!strings.HasPrefix(l.URL, "https://") &&
		!strings.HasPrefix(l.URL, "//")

	if isInternal {
		itemURL = publicURL + l.URL
	}

	item := RSSItem{
		Title: l.Title,
		Link:  itemURL,
		GUID:  itemURL,
	}

	// For internal links, try to get metadata from the target note.
	if isInternal && notes != nil {
		enrichItem(&item, notes, l.URL)
	}

	return item
}

// textContent extracts plain text from an AST node's children.
// enrichItem adds metadata from the target note to an RSS item.
func enrichItem(item *RSSItem, notes *model.NoteViews, path string) {
	target := notes.GetByPath(path)
	if target == nil {
		return
	}

	if target.Description != nil {
		item.Description = *target.Description
	}

	if !target.CreatedAt.IsZero() {
		item.PubDate = target.CreatedAt.Format(time.RFC1123Z)
	}
}

// textContent extracts plain text from an AST node's children.
func textContent(src []byte, n ast.Node) string {
	var buf bytes.Buffer

	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		writeText(src, &buf, c)
	}

	return buf.String()
}

// writeText recursively writes text content of a node.
func writeText(src []byte, buf *bytes.Buffer, n ast.Node) {
	switch n := n.(type) {
	case *ast.Text:
		buf.Write(n.Segment.Value(src))
	case *ast.String:
		buf.Write(n.Value)
	default:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			writeText(src, buf, c)
		}
	}
}
