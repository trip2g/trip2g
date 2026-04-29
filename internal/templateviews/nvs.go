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
}

// NewNVS creates a new template NVS wrapper.
func NewNVS(nvs *model.NoteViews, defaultVersion string) *NVS {
	if nvs == nil {
		return nil
	}
	return &NVS{
		nvs:            nvs,
		defaultVersion: defaultVersion,
	}
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

	return NewNote(nv)
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

	return NewNote(nv)
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
			return NewNote(nv)
		}
		pathKey := strings.ReplaceAll(target, " ", "_") + ".md"
		if nv, ok := n.nvs.PathMap[pathKey]; ok {
			return NewNote(nv)
		}
		return nil
	}

	// Global basename lookup via BasenameMap (O(1)).
	key := strings.ToLower(strings.ReplaceAll(target, " ", "_"))
	candidates := n.nvs.BasenameMap[key]
	if len(candidates) == 1 {
		return NewNote(candidates[0])
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
		return NewNote(shortest)
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
		result = append(result, NewNote(s))
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
		result = append(result, NewNote(hp))
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
		if linked := n.nvs.GetByPath(path); linked != nil {
			result = append(result, NewNote(linked))
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
			result = append(result, NewNote(linked))
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
		result = append(result, NewNote(nv))
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
