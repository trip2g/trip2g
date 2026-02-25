package mdloader

import (
	"bytes"
	"fmt"
	"html/template"

	"trip2g/internal/model"

	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/goldmark/wikilink"
)

// generateDomainHTMLs re-renders notes with domain-specific link resolution.
//
// Two passes:
//  1. Custom domain pass: for each note with custom domain routes, re-render
//     for each of its custom domain hosts. Links to notes on the same domain
//     use relative paths; links to notes on other custom domains use full URLs.
//  2. Main domain pass (host=""): for ALL notes, check whether any outgoing
//     link points to a note whose primary home is a custom domain (i.e. the
//     note has custom domain routes but no main-domain alias). If so,
//     generate DomainHTML[""] so those links use full URLs on the main domain.
func (ldr *loader) generateDomainHTMLs() {
	// Short-circuit: if no custom domains configured, nothing to do.
	domainHosts := ldr.nvs.CustomDomains()
	if len(domainHosts) == 0 {
		return
	}

	for _, p := range ldr.nvs.PathMap {
		// Pass 1: custom domain re-render for notes with explicit custom domain routes.
		if hasCustomDomainRoutes(p) {
			for _, host := range ldr.uniqueHostsForNote(p) {
				domainLinks, domainNoteIndex := ldr.buildDomainResolvedLinks(p, host)
				if domainLinks == nil {
					// No differences from main domain -- skip re-render.
					continue
				}

				if err := ldr.generateDomainHTML(p, host, domainLinks, domainNoteIndex); err != nil {
					ldr.log.Warn("failed to generate domain HTML",
						"path", p.Path, "host", host, "error", err)
				}
			}
		}

		// Pass 2: main domain re-render (host="").
		// Generates DomainHTML[""] when any linked note is "domain-only" (has custom
		// domain routes but is not accessible via a main-domain alias).
		// This makes [[wikilinks]] on the main domain use https://custom.com/path
		// for such notes, instead of the canonical permalink.
		mainLinks, mainNoteIndex := ldr.buildDomainResolvedLinks(p, "")
		if mainLinks != nil {
			if err := ldr.generateDomainHTML(p, "", mainLinks, mainNoteIndex); err != nil {
				ldr.log.Warn("failed to generate main-domain HTML",
					"path", p.Path, "error", err)
			}
		}
	}
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
func (ldr *loader) buildDomainResolvedLinks(
	p *model.NoteView,
	host string,
) (map[string]string, map[string]*model.NoteView) {
	// Find embed targets -- these must NOT be overridden so renderEmbed() works.
	embedTargets := ldr.findEmbedTargets(p)

	domainLinks := make(map[string]string, len(p.ResolvedLinks))
	domainNoteIndex := make(map[string]*model.NoteView)
	changed := false

	for target, mainPermalink := range p.ResolvedLinks {
		// Don't override embed targets -- renderEmbed uses GetByPath(permalink).
		if _, isEmbed := embedTargets[target]; isEmbed {
			domainLinks[target] = mainPermalink
			continue
		}

		targetNote := ldr.nvs.GetByPath(mainPermalink)
		if targetNote == nil {
			domainLinks[target] = mainPermalink
			continue
		}

		domainPath := resolveForDomain(targetNote, host)
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
func (ldr *loader) findEmbedTargets(p *model.NoteView) map[string]struct{} {
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
//   - Target has route on this host -> use domain path.
//   - Target has route on OTHER custom host -> full URL (https://other.com/path).
//   - Target has no custom routes -> use permalink (unchanged).
func resolveForDomain(target *model.NoteView, host string) string {
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

	// No custom domain routes -- use permalink (unchanged).
	return target.Permalink
}

// generateDomainHTML re-renders a single note with domain-specific ResolvedLinks.
// Uses save/restore with defer to ensure state is always recovered.
func (ldr *loader) generateDomainHTML(
	p *model.NoteView,
	host string,
	domainLinks map[string]string,
	domainNoteIndex map[string]*model.NoteView,
) error {
	// Save original state.
	origLinks := p.ResolvedLinks
	origDomainNotes := ldr.linkResolver.domainRenderNotes

	// Override for domain render; defer restore for safety.
	p.ResolvedLinks = domainLinks
	ldr.linkResolver.currentPage = p
	ldr.linkResolver.domainRenderNotes = domainNoteIndex
	defer func() {
		p.ResolvedLinks = origLinks
		ldr.linkResolver.domainRenderNotes = origDomainNotes
	}()

	var buf bytes.Buffer
	err := ldr.md.Renderer().Render(&buf, p.Content, p.Ast())
	if err != nil {
		return fmt.Errorf("render domain HTML for %s on %s: %w", p.Path, host, err)
	}

	if p.DomainHTML == nil {
		p.DomainHTML = make(map[string]template.HTML)
	}
	p.DomainHTML[host] = template.HTML(buf.String()) //nolint:gosec // content from trusted markdown source.
	// TODO(v2): DomainFreeHTML -- also re-render FreeHTML for custom domain context.

	return nil
}

// hasCustomDomainRoutes returns true if the note has at least one route
// with a non-empty Host (custom domain, not a main-domain alias).
func hasCustomDomainRoutes(n *model.NoteView) bool {
	for _, r := range n.Routes {
		if r.Host != "" {
			return true
		}
	}
	return false
}
