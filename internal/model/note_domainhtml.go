package model

import (
	"html/template"
	"sync"
)

// DomainHTMLRenderer renders a note for a host at request time. ok is false
// when the note reads there the same as its HTML.
type DomainHTMLRenderer interface {
	RendersHost(host string) bool
	RenderDomainHTML(note *NoteView, host string) (html template.HTML, ok bool)
}

// lazyDomainHTML memoizes request-time domain renders of one note. The note
// belongs to one snapshot, so do the entries; NoteViews.Copy shares them.
type lazyDomainHTML struct {
	renderer DomainHTMLRenderer

	mu      sync.Mutex
	entries map[string]*lazyDomainHTMLEntry
}

type lazyDomainHTMLEntry struct {
	once sync.Once
	html template.HTML
	ok   bool
}

// SetDomainHTMLRenderer makes HTMLForHost render the hosts DomainHTML has no
// entry for. Called by the loader before the snapshot is published.
func (n *NoteView) SetDomainHTMLRenderer(renderer DomainHTMLRenderer) {
	n.lazyDomainHTML = &lazyDomainHTML{renderer: renderer}
}

// HTMLForHost returns the note's HTML as served on host ("" for the main
// domain): the pre-rendered DomainHTML, else a render made on first use for
// this snapshot, else HTML.
func (n *NoteView) HTMLForHost(host string) template.HTML {
	if html, ok := n.DomainHTML[host]; ok {
		return html
	}

	if n.lazyDomainHTML != nil {
		if html, ok := n.lazyDomainHTML.get(n, host); ok {
			return html
		}
	}

	return n.HTML
}

func (l *lazyDomainHTML) get(note *NoteView, host string) (template.HTML, bool) {
	if !l.renderer.RendersHost(host) {
		return "", false
	}

	l.mu.Lock()
	entry, found := l.entries[host]
	if !found {
		if l.entries == nil {
			l.entries = make(map[string]*lazyDomainHTMLEntry)
		}
		entry = &lazyDomainHTMLEntry{}
		l.entries[host] = entry
	}
	l.mu.Unlock()

	entry.once.Do(func() {
		entry.html, entry.ok = l.renderer.RenderDomainHTML(note, host)
	})

	return entry.html, entry.ok
}
