package model

import (
	"path/filepath"
	"strings"
)

// WikilinkResolution selects the strategy for resolving bare [[Name]] wikilinks
// when several notes share the same basename.
type WikilinkResolution string

const (
	// WikilinkResolutionScoped is the opt-in ladder:
	// same folder → same language → global shallowest, ties broken by path.
	WikilinkResolutionScoped WikilinkResolution = "scoped"

	// WikilinkResolutionGlobal is the default rule matching Obsidian: global
	// shallowest path only (language- and folder-blind), ties broken by path.
	// A bare [[note]] resolves to the root-most match even from a subfolder
	// (see docs/dev/obsidian_links.md).
	WikilinkResolutionGlobal WikilinkResolution = "global"

	// DefaultWikilinkResolution is the default when no config is set.
	DefaultWikilinkResolution = WikilinkResolutionGlobal
)

// Valid returns true if the resolution mode is a recognized user choice.
func (r WikilinkResolution) Valid() bool {
	return r == WikilinkResolutionScoped || r == WikilinkResolutionGlobal
}

// knownLangFolders are top-level folder names treated as language prefixes.
// Deliberately a short allowlist of common ISO 639-1 codes: matching any
// two-letter folder would misfire on non-language folders like "go/".
//
//nolint:gochecknoglobals // static allowlist shared by resolver and doclint.
var knownLangFolders = map[string]bool{
	"en": true, "ru": true, "de": true, "fr": true, "es": true,
	"pt": true, "it": true, "nl": true, "pl": true, "uk": true,
	"tr": true, "zh": true, "ja": true, "ko": true, "ar": true,
}

// PathLangPrefix returns the language code when the note path starts with a
// known top-level language folder (e.g. "en/article.md" → "en"), otherwise "".
func PathLangPrefix(path string) string {
	seg, _, ok := strings.Cut(path, "/")
	if ok && knownLangFolders[seg] {
		return seg
	}
	return ""
}

// noteLang returns the note's language: explicit frontmatter lang if set,
// else the path's language folder prefix.
func noteLang(n *NoteView) string {
	if n == nil {
		return ""
	}
	if n.Lang != "" {
		return n.Lang
	}
	return PathLangPrefix(n.Path)
}

// pickBareCandidate resolves a bare [[Name]] link among candidates sharing the
// basename. Global mode (default) keeps only the global-shallowest step, matching
// Obsidian (language- and folder-blind). Scoped mode (opt-in) applies the ladder:
// same folder as source → same language as source (shallowest) → global shallowest.
// Every step breaks ties lexicographically by path.
func (nvs *NoteViews) pickBareCandidate(source *NoteView, candidates []*NoteView) *NoteView {
	if len(candidates) == 0 {
		return nil
	}
	if len(candidates) == 1 {
		return candidates[0]
	}

	if nvs.WikilinkResolution == WikilinkResolutionScoped && source != nil {
		srcDir := filepath.Dir(source.Path)
		if sameDir := filterNotes(candidates, func(c *NoteView) bool {
			return filepath.Dir(c.Path) == srcDir
		}); len(sameDir) > 0 {
			return shallowestNote(sameDir)
		}

		if lang := noteLang(source); lang != "" {
			if sameLang := filterNotes(candidates, func(c *NoteView) bool {
				return noteLang(c) == lang
			}); len(sameLang) > 0 {
				return shallowestNote(sameLang)
			}
		}
	}

	return shallowestNote(candidates)
}

func filterNotes(notes []*NoteView, keep func(*NoteView) bool) []*NoteView {
	var res []*NoteView
	for _, n := range notes {
		if keep(n) {
			res = append(res, n)
		}
	}
	return res
}

// shallowestNote returns the candidate with the fewest path segments,
// breaking ties lexicographically by path.
func shallowestNote(candidates []*NoteView) *NoteView {
	best := candidates[0]
	bestDepth := strings.Count(best.Path, "/")
	for _, c := range candidates[1:] {
		depth := strings.Count(c.Path, "/")
		if depth < bestDepth || (depth == bestDepth && c.Path < best.Path) {
			best = c
			bestDepth = depth
		}
	}
	return best
}
