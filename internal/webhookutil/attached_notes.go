package webhookutil

import (
	"sort"
	"strings"
	"time"

	"trip2g/internal/model"
)

// AttachedNote is a context note included in webhook payloads via attach_notes.
// Meta is an allowlist (never the full RawMeta) so the role only sees key fields.
type AttachedNote struct {
	Path      string            `json:"path"`
	Title     string            `json:"title"`
	Content   string            `json:"content"`
	UpdatedAt string            `json:"updated_at,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	Meta      map[string]string `json:"meta"`
}

// MaterializeAttachedNotes returns notes matching the non-negated attach globs,
// sorted by path for determinism.
func MaterializeAttachedNotes(patterns []string, nvs *model.NoteViews) []AttachedNote {
	if nvs == nil {
		return nil
	}
	var globs []string
	for _, p := range patterns {
		if !strings.HasPrefix(p, "!") {
			globs = append(globs, p)
		}
	}
	if len(globs) == 0 {
		return nil
	}
	var out []AttachedNote
	for path, nv := range nvs.PathMap {
		if !MatchesAny(path, globs) {
			continue
		}
		an := AttachedNote{
			Path:    nv.Path,
			Title:   nv.Title,
			Content: string(nv.Content),
			Tags:    nv.Tags,
			Meta:    map[string]string{},
		}
		if !nv.UpdatedAt.IsZero() {
			an.UpdatedAt = nv.UpdatedAt.Format(time.RFC3339)
		}
		if len(nv.Tags) > 0 {
			an.Meta["tags"] = strings.Join(nv.Tags, ",")
		}
		if nv.Layout != "" {
			an.Meta["layout"] = nv.Layout
		}
		out = append(out, an)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// AttachGateSatisfied reports whether the attach_notes presence conditions hold.
// A plain glob requires >=1 matching note; a "!glob" requires 0 matching notes.
// Empty patterns always satisfy.
func AttachGateSatisfied(patterns []string, nvs *model.NoteViews) bool {
	for _, pat := range patterns {
		if strings.HasPrefix(pat, "!") {
			if attachAnyNoteMatches(strings.TrimPrefix(pat, "!"), nvs) {
				return false
			}
			continue
		}
		if !attachAnyNoteMatches(pat, nvs) {
			return false
		}
	}
	return true
}

func attachAnyNoteMatches(glob string, nvs *model.NoteViews) bool {
	if nvs == nil {
		return false
	}
	for path := range nvs.PathMap {
		if MatchesAny(path, []string{glob}) {
			return true
		}
	}
	return false
}
