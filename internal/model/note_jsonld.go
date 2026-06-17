package model

import (
	"strings"
	"time"
)

// extractJSONLDFields pulls author, updated date, and tags from frontmatter for
// use in JSON-LD structured data (see internal/defaulttemplate/jsonld.go).
// Dates are parsed in UTC, consistent with ExtractCreatedAt.
func (n *NoteView) extractJSONLDFields() {
	if a, ok := n.RawMeta["author"].(string); ok {
		n.Author = strings.TrimSpace(a)
	}

	for _, key := range []string{"updated_at", "updated", "modified"} {
		v, ok := n.RawMeta[key].(string)
		if !ok {
			continue
		}
		if t, parsed := parseDate(v, time.UTC); parsed {
			n.UpdatedAt = t
			break
		}
	}

	n.Tags = extractTagList(n.RawMeta["tags"], n.RawMeta["keywords"])
}

// extractTagList returns the first non-empty tag list among the given frontmatter
// values (so "tags" wins over "keywords"). Each value may be a []string,
// []interface{} of strings, or a comma-separated string.
func extractTagList(values ...interface{}) []string {
	for _, v := range values {
		var out []string
		switch val := v.(type) {
		case string:
			for _, part := range strings.Split(val, ",") {
				if t := strings.TrimSpace(part); t != "" {
					out = append(out, t)
				}
			}
		case []interface{}:
			for _, item := range val {
				if s, ok := item.(string); ok {
					if t := strings.TrimSpace(s); t != "" {
						out = append(out, t)
					}
				}
			}
		case []string:
			for _, s := range val {
				if t := strings.TrimSpace(s); t != "" {
					out = append(out, t)
				}
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}
