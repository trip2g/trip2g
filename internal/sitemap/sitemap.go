// Package sitemap generates sitemap.xml from published notes.
package sitemap

import (
	"bytes"
	"encoding/xml"
	"strings"
	"time"

	"trip2g/internal/model"
)

const xmlns = "http://www.sitemaps.org/schemas/sitemap/0.9"

type urlset struct {
	XMLName xml.Name   `xml:"urlset"`
	XMLNS   string     `xml:"xmlns,attr"`
	URLs    []urlEntry `xml:"url"`
}

type urlEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

// Generate creates a sitemap.xml from NoteViews.
// Only free and visible notes are included.
func Generate(nvs *model.NoteViews, publicURL string) ([]byte, error) {
	var urls []urlEntry

	for _, note := range nvs.List {
		if !note.Free {
			continue
		}

		if strings.Contains(note.Permalink, "/_") {
			continue
		}

		entry := urlEntry{
			Loc: publicURL + note.Permalink,
		}

		if !note.CreatedAt.IsZero() {
			entry.LastMod = note.CreatedAt.Format(time.RFC3339)
		}

		urls = append(urls, entry)
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
