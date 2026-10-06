package layoutloader

import (
	"fmt"
	"regexp"
	"sort"
	"trip2g/internal/model"

	"github.com/CloudyKit/jet/v6"
)

type blockRegistryEntry struct {
	SourceID   string
	Duplicates []string
}

var leadingExtendsRegexp = regexp.MustCompile(`^\s*{{-?\s*extends\s+"`)

// buildBlockRegistry scans the component files among sourceIDs and maps blockName → file.
// pageSourceID and every file that starts with {{ extends }} are pages, not components,
// and are skipped. Returns sorted-deterministic results. Emits NoteWarning for duplicate
// block names.
func buildBlockRegistry(
	views *jet.Set,
	templates map[string]string,
	sourceIDs []string,
	pageSourceID string,
) (map[string]blockRegistryEntry, []model.NoteWarning) {
	sorted := make([]string, 0, len(sourceIDs))
	for _, id := range sourceIDs {
		if id != pageSourceID && !leadingExtendsRegexp.MatchString(templates[id]) {
			sorted = append(sorted, id)
		}
	}
	sort.Strings(sorted)

	registry := make(map[string]blockRegistryEntry)
	var warnings []model.NoteWarning

	for _, id := range sorted {
		t, err := views.GetTemplate(id)
		if err != nil || t == nil {
			continue
		}
		finder := &blockNameFinder{}
		if msg := walkContained(t, finder); msg != "" {
			// Broken template (its own load reports the error) — skip its blocks
			// so it can't take down registry building for other pages.
			continue
		}
		for _, name := range finder.names {
			if existing, ok := registry[name]; ok {
				warnings = append(warnings, model.NoteWarning{
					Level:   model.NoteWarningWarning,
					Message: fmt.Sprintf("block %q defined in both %s and %s", name, existing.SourceID, id),
				})
				existing.Duplicates = append(existing.Duplicates, id)
				registry[name] = existing
			} else {
				registry[name] = blockRegistryEntry{SourceID: id}
			}
		}
	}
	return registry, warnings
}
