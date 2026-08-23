package templateviews

import (
	"strings"

	"trip2g/internal/model"
)

// NVS wraps model.NoteViews for template usage.
// Provides methods to access notes by path or permalink.
type NVS struct {
	nvs            *model.NoteViews
	defaultVersion string
	domainHost     string
}

// NewNVS creates a new template NVS wrapper for the main domain.
func NewNVS(nvs *model.NoteViews, defaultVersion string) *NVS {
	return NewNVSWithDomain(nvs, defaultVersion, "")
}

// NewNVSWithDomain creates a template NVS wrapper bound to the host the page
// is being served on. Notes it returns render that host's HTML, so a header,
// footer or sidebar resolves its links exactly as the page body does.
func NewNVSWithDomain(nvs *model.NoteViews, defaultVersion, domainHost string) *NVS {
	if nvs == nil {
		return nil
	}
	return &NVS{
		nvs:            nvs,
		defaultVersion: defaultVersion,
		domainHost:     domainHost,
	}
}

// wrap builds a Note carrying this NVS's host, so every note reached through
// the wrapper renders for the same domain as the page around it.
func (n *NVS) wrap(nv *model.NoteView) *Note {
	return NewNoteWithDomain(nv, n.domainHost)
}

// ByPath returns a note by its file path (e.g., "/_sidebar.md", "_sidebar.md").
// Leading slash is trimmed automatically for convenience.
// Returns nil if note not found.
func (n *NVS) ByPath(path string) *Note {
	if n.nvs == nil {
		return nil
	}

	path = strings.TrimPrefix(path, "/")

	nv, ok := n.nvs.PathMap[path]
	if !ok {
		return nil
	}

	return n.wrap(nv)
}

// ByPermalink returns a note by its permalink (e.g., "/docs", "/about").
// Returns nil if note not found.
func (n *NVS) ByPermalink(permalink string) *Note {
	if n.nvs == nil {
		return nil
	}

	nv, ok := n.nvs.Map[permalink]
	if !ok {
		return nil
	}

	return n.wrap(nv)
}

// ByWikilink resolves a wikilink target using Obsidian's algorithm:
// 1. If target contains "/" — explicit path lookup
// 2. Otherwise — global basename lookup (shortest path from root wins)
// See docs/dev/obsidian_links.md for the full algorithm.
func (n *NVS) ByWikilink(target string) *Note {
	if n.nvs == nil || target == "" {
		return nil
	}

	// Explicit path: try permalink and PathMap.
	if strings.Contains(target, "/") {
		permalink := "/" + strings.ToLower(strings.ReplaceAll(target, " ", "_"))
		if nv, ok := n.nvs.Map[permalink]; ok {
			return n.wrap(nv)
		}
		pathKey := strings.ReplaceAll(target, " ", "_") + ".md"
		if nv, ok := n.nvs.PathMap[pathKey]; ok {
			return n.wrap(nv)
		}
		return nil
	}

	// Global basename lookup via BasenameMap (O(1)).
	key := strings.ToLower(strings.ReplaceAll(target, " ", "_"))
	candidates := n.nvs.BasenameMap[key]
	if len(candidates) == 1 {
		return n.wrap(candidates[0])
	}
	if len(candidates) > 1 {
		// Shortest path from root wins (Obsidian priority).
		shortest := candidates[0]
		shortestDepth := strings.Count(shortest.Path, "/")
		for _, c := range candidates[1:] {
			depth := strings.Count(c.Path, "/")
			if depth < shortestDepth {
				shortest = c
				shortestDepth = depth
			}
		}
		return n.wrap(shortest)
	}

	return nil
}

// Sidebars returns sidebar notes for a given note.
func (n *NVS) Sidebars(note *Note) []*Note {
	if n.nvs == nil || note == nil {
		return nil
	}

	sidebars := n.nvs.Sidebars(note.nv)
	result := make([]*Note, 0, len(sidebars))
	for _, s := range sidebars {
		result = append(result, n.wrap(s))
	}
	return result
}

// HomePages returns home page notes for a given note's subgraphs.
func (n *NVS) HomePages(note *Note) []*Note {
	if n.nvs == nil || note == nil {
		return nil
	}

	homePages := n.nvs.HomePages(note.nv)
	result := make([]*Note, 0, len(homePages))
	for _, hp := range homePages {
		result = append(result, n.wrap(hp))
	}
	return result
}

// BackLinks returns notes that link to the given note.
func (n *NVS) BackLinks(note *Note) []*Note {
	if n.nvs == nil || note == nil {
		return nil
	}

	result := make([]*Note, 0, len(note.nv.InLinks))
	for path := range note.nv.InLinks {
		if linked := n.nvs.GetByPath(path); linked != nil && !linked.IsSystem() {
			result = append(result, n.wrap(linked))
		}
	}
	return result
}

// OutLinks returns notes that this note links to (resolved wikilinks/paths).
func (n *NVS) OutLinks(note *Note) []*Note {
	if n.nvs == nil || note == nil {
		return nil
	}

	seen := make(map[string]struct{}, len(note.nv.ResolvedLinks))
	result := make([]*Note, 0, len(note.nv.ResolvedLinks))
	for _, permalink := range note.nv.ResolvedLinks {
		if _, ok := seen[permalink]; ok {
			continue
		}
		seen[permalink] = struct{}{}
		if linked := n.nvs.GetByPath(permalink); linked != nil {
			result = append(result, n.wrap(linked))
		}
	}
	return result
}

// ResolveURL returns the full URL for a note, including version if needed.
func (n *NVS) ResolveURL(note *Note) string {
	if n.nvs == nil || note == nil {
		return ""
	}
	return n.nvs.ResolveURL(note.nv)
}

// List returns all visible notes (excluding system notes starting with /_).
func (n *NVS) List() []*Note {
	if n.nvs == nil {
		return nil
	}

	visible := n.nvs.VisibleList()
	result := make([]*Note, 0, len(visible))
	for _, nv := range visible {
		result = append(result, n.wrap(nv))
	}
	return result
}

// ByGlob returns a query builder for notes matching a glob pattern.
// Supports ** for recursive matching: "blog/*.md", "projects/**/README.md".
func (n *NVS) ByGlob(pattern string) *NoteQuery {
	return &NoteQuery{
		nvs:  n,
		glob: pattern,
	}
}

// Query returns an empty query builder for all notes.
func (n *NVS) Query() *NoteQuery {
	return &NoteQuery{
		nvs: n,
	}
}
