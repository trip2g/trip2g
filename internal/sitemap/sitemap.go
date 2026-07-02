// Package sitemap generates sitemap.xml from published notes.
package sitemap

import (
	"bytes"
	"encoding/xml"
	"strings"
	"time"

	"trip2g/internal/model"
)

const (
	xmlns      = "http://www.sitemaps.org/schemas/sitemap/0.9"
	xmlnsXHTML = "http://www.w3.org/1999/xhtml"
)

type urlset struct {
	XMLName    xml.Name   `xml:"urlset"`
	XMLNS      string     `xml:"xmlns,attr"`
	XMLNSXHTML string     `xml:"xmlns:xhtml,attr,omitempty"`
	URLs       []urlEntry `xml:"url"`
}

type urlEntry struct {
	Loc        string      `xml:"loc"`
	LastMod    string      `xml:"lastmod,omitempty"`
	Alternates []xhtmlLink `xml:"xhtml:link,omitempty"`
}

// xhtmlLink is an <xhtml:link rel="alternate" hreflang="..." href="..."/>
// language alternate for a URL, per the sitemap hreflang protocol.
type xhtmlLink struct {
	Rel      string `xml:"rel,attr"`
	HrefLang string `xml:"hreflang,attr"`
	Href     string `xml:"href,attr"`
}

// hreflangAlternates builds the language-alternate links for a note from its
// LangGroup (hub + all language versions). Every URL in a language set lists the
// full set plus x-default -> hub, per Google's guidance. Returns nil for notes
// with no language group.
func hreflangAlternates(note *model.NoteView, publicURL string) []xhtmlLink {
	if note.LangGroup == nil {
		return nil
	}
	group := note.LangGroup

	var links []xhtmlLink
	hubURL := publicURL + group.Hub.PermalinkEncoded()
	if group.Hub.Lang != "" {
		links = append(links, xhtmlLink{Rel: "alternate", HrefLang: group.Hub.Lang, Href: hubURL})
	}
	for _, lr := range group.Versions {
		if lr.Note == nil || lr.Note == group.Hub || lr.Lang == "" {
			continue
		}
		links = append(links, xhtmlLink{
			Rel:      "alternate",
			HrefLang: lr.Lang,
			Href:     publicURL + lr.Note.PermalinkEncoded(),
		})
	}
	if len(links) == 0 {
		return nil
	}
	links = append(links, xhtmlLink{Rel: "alternate", HrefLang: "x-default", Href: hubURL})
	return links
}

// hasAlternates returns the xhtml namespace URI when any entry carries hreflang
// alternates (so the attribute is only declared when used), else "".
func hasAlternates(urls []urlEntry) string {
	for _, u := range urls {
		if len(u.Alternates) > 0 {
			return xmlnsXHTML
		}
	}
	return ""
}

// Generate creates a sitemap.xml from NoteViews.
// Only free and visible notes are included.
func Generate(nvs *model.NoteViews, publicURL string) ([]byte, error) {
	var urls []urlEntry

	for _, note := range nvs.List {
		if !note.Free {
			continue
		}

		// Skip notes that require sign-in (they have noindex).
		requiresSignin := false
		for _, sgName := range note.SubgraphNames {
			if sg, found := nvs.Subgraphs[sgName]; found && sg.RequireSignin {
				requiresSignin = true
				break
			}
		}
		if requiresSignin {
			continue
		}

		if strings.Contains(note.Permalink, "/_") {
			continue
		}

		entry := urlEntry{
			Loc:        publicURL + note.PermalinkEncoded(),
			Alternates: hreflangAlternates(note, publicURL),
		}

		if !note.CreatedAt.IsZero() {
			entry.LastMod = note.CreatedAt.Format(time.RFC3339)
		}

		urls = append(urls, entry)
	}

	set := urlset{
		XMLNS:      xmlns,
		XMLNSXHTML: hasAlternates(urls),
		URLs:       urls,
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)

	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")

	err := enc.Encode(set)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// GenerateForDomain creates a sitemap for a specific custom domain.
// Includes notes accessible on this domain (from RouteMap[domain]).
// Only free notes are included.
func GenerateForDomain(nvs *model.NoteViews, domain, baseURL string) ([]byte, error) {
	routes, ok := nvs.RouteMap[domain]
	if !ok {
		return nil, nil
	}

	var urls []urlEntry

	for path, note := range routes {
		if !note.Free {
			continue
		}

		// Skip notes that require sign-in (they have noindex).
		requiresSignin := false
		for _, sgName := range note.SubgraphNames {
			if sg, found := nvs.Subgraphs[sgName]; found && sg.RequireSignin {
				requiresSignin = true
				break
			}
		}
		if requiresSignin {
			continue
		}

		if strings.Contains(path, "/_") {
			continue
		}

		entry := urlEntry{
			Loc: baseURL + path,
		}

		if !note.CreatedAt.IsZero() {
			entry.LastMod = note.CreatedAt.Format(time.RFC3339)
		}

		urls = append(urls, entry)
	}

	if len(urls) == 0 {
		return nil, nil
	}

	set := urlset{
		XMLNS: xmlns,
		URLs:  urls,
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)

	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")

	err := enc.Encode(set)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
