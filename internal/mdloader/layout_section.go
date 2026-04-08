package mdloader

import (
	"trip2g/internal/model"
)

// Layout section types recognized in frontmatter.
var layoutSectionTypes = []string{"header", "footer", "left_sidebar", "right_sidebar"}

// parseLayoutSections extracts layout section entries from a note's rawMeta.
// A note becomes a layout section when it has {section}_includes in its frontmatter.
// Returns nil if no layout section fields are found.
func parseLayoutSections(rawMeta map[string]interface{}, path string) []model.LayoutSectionEntry {
	var entries []model.LayoutSectionEntry

	for _, section := range layoutSectionTypes {
		includes := toStringSliceFlexible(rawMeta[section+"_includes"])
		if len(includes) == 0 {
			continue
		}

		entry := model.LayoutSectionEntry{
			NotePath:        path,
			Section:         section,
			Includes:        includes,
			Excludes:        toStringSliceFlexible(rawMeta[section+"_excludes"]),
			IncludeProperty: toString(rawMeta[section+"_include_property"]),
			ExcludeProperty: toString(rawMeta[section+"_exclude_property"]),
			Priority:        toInt(rawMeta[section+"_priority"]),
		}

		entries = append(entries, entry)
	}

	return entries
}

// toStringSliceFlexible converts a YAML value to []string.
// Accepts string (single value) or []interface{}/[]string (list).
func toStringSliceFlexible(v interface{}) []string {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case string:
		return []string{val}
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return val
	}

	return nil
}

// toString converts a YAML value to string. Returns "" for nil or non-string.
func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
