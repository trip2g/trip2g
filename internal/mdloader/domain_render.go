package mdloader

import (
	"bytes"
	"fmt"
	"html/template"

	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/goldmark/wikilink"
)

// domainRenderer renders a note with domain-specific link resolution. It holds
// only what a render needs, so a snapshot can keep it for request-time renders
// without keeping the loader.
type domainRenderer struct {
	nvs       *model.NoteViews
	md        goldmark.Markdown
	resolver  *myLinkResolver
	log       logger.Logger
	publicURL string
}

// generateDomainHTMLs renders domain-specific HTML for what a host serves at
// its own paths, and leaves the rest to request time.
//
//  1. Custom domain hosts: a note with custom domain routes is pre-rendered for
//     each of its hosts. Links to notes on the same domain use relative paths;
//     links to notes on other custom domains use full URLs.
//     A note without any -- a shared header, footer or sidebar -- is served as
//     part of pages that do live on a host, and its links have to resolve the
//     way the body's do. It is rendered for a host on first use (HTMLForHost),
//     once per snapshot.
//  2. Main domain (host=""): for ALL notes, check whether any outgoing link
//     points to a note whose primary home is a custom domain (i.e. the note has
//     custom domain routes but no main-domain alias). If so, pre-render
//     DomainHTML[""] so those links use full URLs on the main domain.
func (ldr *loader) generateDomainHTMLs() {
	if len(ldr.nvs.CustomDomains()) == 0 {
		return
	}

	for _, p := range ldr.nvs.PathMap {
		// Raw files (.canvas, .base, .excalidraw) have no AST and no domain
		// rendering applies to them.
		if p.Ast() == nil {
			continue
		}

		hosts := ldr.uniqueHostsForNote(p)
		if len(hosts) == 0 {
			p.SetDomainHTMLRenderer(ldr.domainRenderer)
		}
		for _, host := range hosts {
			ldr.storeDomainHTML(p, host)
		}

		ldr.storeDomainHTML(p, "")
	}
}

func (ldr *loader) storeDomainHTML(p *model.NoteView, host string) {
	html, ok := ldr.domainRenderer.RenderDomainHTML(p, host)
	if !ok {
		return
	}

	if p.DomainHTML == nil {
		p.DomainHTML = make(map[string]template.HTML)
	}
	p.DomainHTML[host] = html
}

// RendersHost reports whether host is a custom domain of the snapshot.
func (dr *domainRenderer) RendersHost(host string) bool {
	return dr.nvs.IsCustomDomain(host)
}

// RenderDomainHTML renders p for host. ok is false when its links resolve
// there the same as on the main domain, or the render failed.
func (dr *domainRenderer) RenderDomainHTML(p *model.NoteView, host string) (template.HTML, bool) {
	domainLinks, domainNoteIndex := dr.buildDomainResolvedLinks(p, host)
	if domainLinks == nil {
		return "", false
	}

	html, err := dr.render(p, domainLinks, domainNoteIndex)
	if err != nil {
		dr.log.Warn("failed to generate domain HTML", "path", p.Path, "host", host, "error", err)
		return "", false
	}

	return html, true
}

// uniqueHostsForNote returns unique custom domain hosts from note's Routes.
func (ldr *loader) uniqueHostsForNote(p *model.NoteView) []string {
	seen := make(map[string]struct{})
	var hosts []string
	for _, r := range p.Routes {
		if r.Host != "" {
			if _, exists := seen[r.Host]; !exists {
				seen[r.Host] = struct{}{}
				hosts = append(hosts, r.Host)
			}
		}
	}
	return hosts
}

// buildDomainResolvedLinks creates a domain-specific ResolvedLinks map for re-rendering.
// Returns nil, nil if the domain links would be identical to the main ResolvedLinks
// (optimization: skip re-render when nothing differs).
// Also returns domainNoteIndex mapping domain-specific paths/URLs to NoteViews, so
// link_renderer can find notes by domain path for data-pid and paywall classes.
func (dr *domainRenderer) buildDomainResolvedLinks(
	p *model.NoteView,
	host string,
) (map[string]string, map[string]*model.NoteView) {
	mainBaseURL := dr.publicURL
	// Find embed targets -- these must NOT be overridden so renderEmbed() works.
	embedTargets := findEmbedTargets(p)

	domainLinks := make(map[string]string, len(p.ResolvedLinks))
	domainNoteIndex := make(map[string]*model.NoteView)
	changed := false

	for target, mainPermalink := range p.ResolvedLinks {
		// Don't override embed targets -- renderEmbed uses GetByPath(permalink).
		if _, isEmbed := embedTargets[target]; isEmbed {
			domainLinks[target] = mainPermalink
			continue
		}

		targetNote := dr.nvs.GetByPath(mainPermalink)
		if targetNote == nil {
			domainLinks[target] = mainPermalink
			continue
		}

		domainPath := resolveForDomain(targetNote, host, mainBaseURL)
		domainLinks[target] = domainPath

		if domainPath != mainPermalink {
			changed = true
			domainNoteIndex[domainPath] = targetNote
		}
	}

	if !changed {
		return nil, nil
	}

	return domainLinks, domainNoteIndex
}

// findEmbedTargets walks the AST and returns wikilink targets that are embed
// links (![[target]]). These keep their permalink so renderEmbed() works via GetByPath.
func findEmbedTargets(p *model.NoteView) map[string]struct{} {
	result := make(map[string]struct{})
	_ = ast.Walk(p.Ast(), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || n.Kind() != wikilink.Kind {
			return ast.WalkContinue, nil
		}
		link, ok := n.(*wikilink.Node)
		if ok && link.Embed {
			result[string(link.Target)] = struct{}{}
		}
		return ast.WalkContinue, nil
	})
	return result
}

// resolveForDomain determines the correct href for a target note in domain context.
//
// Rules:
//   - Target has route on this host -> use domain path (relative).
//   - Target has route on OTHER custom host -> full URL (https://other.com/path).
//   - Target not accessible on this custom domain -> full main-domain URL if mainBaseURL
//     is set (https://main.com/permalink), otherwise canonical permalink.
//   - Main domain pass (host=="") behaves as before: custom-domain-only notes get full
//     URL, main-domain alias notes get alias path, plain notes get permalink.
func resolveForDomain(target *model.NoteView, host string, mainBaseURL string) string {
	// Check if target has a route on the current domain.
	for _, r := range target.Routes {
		if r.Host == host {
			if r.Path != "" {
				return r.Path
			}
			return target.Permalink
		}
	}

	// Check if target has routes on other custom domains -- use full URL.
	for _, r := range target.Routes {
		if r.Host != "" && r.Host != host {
			path := r.Path
			if path == "" {
				path = target.Permalink
			}
			return fmt.Sprintf("https://%s%s", r.Host, path)
		}
	}

	// Target is not accessible on the current custom domain.
	// Use full main-domain URL to avoid broken links on isolated custom domains.
	if host != "" && mainBaseURL != "" {
		return mainBaseURL + target.Permalink
	}

	// No custom domain routes -- use permalink (unchanged).
	return target.Permalink
}

// render renders p with domainLinks in place of its ResolvedLinks. It renders
// a copy of p, so the snapshot's note is never written: warnings the render
// raises repeat the ones of p's main render and stay on the copy.
func (dr *domainRenderer) render(
	p *model.NoteView,
	domainLinks map[string]string,
	domainNoteIndex map[string]*model.NoteView,
) (template.HTML, error) {
	page := *p
	page.ResolvedLinks = domainLinks
	page.Warnings = nil

	dr.resolver.renderMu.Lock()
	defer dr.resolver.renderMu.Unlock()

	prevPage := dr.resolver.currentPage
	prevDomainNotes := dr.resolver.domainRenderNotes
	dr.resolver.currentPage = &page
	dr.resolver.domainRenderNotes = domainNoteIndex
	defer func() {
		dr.resolver.currentPage = prevPage
		dr.resolver.domainRenderNotes = prevDomainNotes
	}()

	var buf bytes.Buffer
	err := dr.md.Renderer().Render(&buf, p.Content, p.Ast())
	if err != nil {
		return "", fmt.Errorf("render domain HTML for %s: %w", p.Path, err)
	}

	return template.HTML(buf.String()), nil //nolint:gosec // content from trusted markdown source.
}
