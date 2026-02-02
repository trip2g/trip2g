package model

import "time"

// ExtractCreatedAt tries to extract created_at or created_on from frontmatter.
// If found and valid, it overrides the NoteView.CreatedAt field.
// Falls back to the database value if frontmatter field is missing or invalid.
func (n *NoteView) ExtractCreatedAt(loc *time.Location) {
	raw, ok := n.extractCreatedAtRaw()
	if !ok {
		return
	}

	str, ok := raw.(string)
	if !ok {
		n.AddWarning(NoteWarningWarning, "invalid created_at format, expected string")
		return
	}

	parsed, ok := parseDate(str, loc)
	if !ok {
		n.AddWarning(NoteWarningWarning, "failed to parse created_at: %s", str)
		return
	}

	n.CreatedAt = parsed
}

// extractCreatedAtRaw returns the raw value from frontmatter, trying created_at first, then created_on.
func (n *NoteView) extractCreatedAtRaw() (interface{}, bool) {
	if val, ok := n.RawMeta["created_at"]; ok {
		return val, true
	}

	if val, ok := n.RawMeta["created_on"]; ok {
		return val, true
	}

	return nil, false
}

// parseDate tries multiple date formats and returns the first successful parse.
func parseDate(s string, loc *time.Location) (time.Time, bool) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006-01-02",
	}

	// Try formats with timezone first.
	for _, format := range formats {
		t, err := time.Parse(format, s)
		if err == nil {
			return t, true
		}
	}

	// Try formats without timezone using the provided location.
	for _, format := range formats {
		t, err := time.ParseInLocation(format, s, loc)
		if err == nil {
			return t, true
		}
	}

	return time.Time{}, false
}
